/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"

	danaiov1 "dana.io/nfs-operator/api/v1"

	"dana.io/nfs-operator/internal/controller/utils"
)

// NfsPvcReconciler reconciles a NfsPvc object
type NfsPvcReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	NfsPvcFinalizerName = "nfspvc.dana.io/finalizer"
)

func (r *NfsPvcReconciler) deleteAssociatedResources(ctx context.Context, nfspvc *danaiov1.NfsPvc) error {
	pvName := nfspvc.Name + "-" + nfspvc.Namespace

	// Delete the PersistentVolume
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}
	if err := r.Delete(ctx, pv); client.IgnoreNotFound(err) != nil {
		return err
	}

	// Delete the PersistentVolumeClaim
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfspvc.Name,
			Namespace: nfspvc.Namespace,
		},
	}
	if err := r.Delete(ctx, pvc); client.IgnoreNotFound(err) != nil {
		return err
	}

	// Add more cleanup logic if needed...

	return nil
}

func (r *NfsPvcReconciler) HandleDeletion(ctx context.Context, nfspvc *danaiov1.NfsPvc) error {
	// Check if our finalizer is present
	if utils.ContainsString(nfspvc.ObjectMeta.Finalizers, NfsPvcFinalizerName) {
		// Handle the actual cleanup of associated resources
		if err := r.deleteAssociatedResources(ctx, nfspvc); err != nil {
			return err
		}

		// Remove the finalizer after cleanup
		nfspvc.ObjectMeta.Finalizers = utils.RemoveString(nfspvc.ObjectMeta.Finalizers, NfsPvcFinalizerName)
		if err := r.Update(ctx, nfspvc); err != nil {
			return err
		}
	}
	return nil
}

// HandleCreation handles the creation phase, including adding finalizers.
func (r *NfsPvcReconciler) HandleCreation(ctx context.Context, nfspvc *danaiov1.NfsPvc) error {
	// If PVC status is empty, set it to Pending
	if nfspvc.Status.PvcStatus == "" {
		nfspvc.Status.PvcStatus = danaiov1.PvcStatusPending
		if err := r.Status().Update(ctx, nfspvc); err != nil {
			return err
		}
	}

	if !controllerutil.ContainsFinalizer(nfspvc, NfsPvcFinalizerName) {
		controllerutil.AddFinalizer(nfspvc, NfsPvcFinalizerName)
		return r.Update(ctx, nfspvc)
	}
	return nil
}

//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=nfspvcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=nfspvcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dana.io.dana.io,resources=nfspvcs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NfsPvc object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

func (r *NfsPvcReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var nfspvcFetched bool

	// Fetch the NfsPvc instance
	nfspvc := &danaiov1.NfsPvc{}
	err := r.Get(ctx, req.NamespacedName, nfspvc)

	if err == nil {
		nfspvcFetched = true
	}

	defer func() {
		if err != nil && nfspvcFetched {
			nfspvc.Status.PvcStatus = danaiov1.PvcStatusError
			updateErr := r.Status().Update(ctx, nfspvc)
			if updateErr != nil {
				logger.Error(updateErr, "Failed to update NfsPvc status to error.")
			}
		}
	}()

	// Handle the error if any during the fetch
	if err != nil {
		if errors.IsNotFound(err) {
			// If the resource is not found, it might've been deleted after the reconcile request.
			// You can choose to log this if needed.
			logger.Info("NfsPvc resource not found. Ignoring since object must've been deleted.")
			return ctrl.Result{}, nil
		}
		// For any other errors, requeue the request with some back-off.
		logger.Error(err, "Failed to retrieve NfsPvc resource.")
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if !nfspvc.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if err := r.HandleDeletion(ctx, nfspvc); err != nil {
			logger.Error(err, "Failed to handle deletion") // Logging the error
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if err := r.HandleCreation(ctx, nfspvc); err != nil {
		logger.Error(err, "Failed to handle creation") // Logging the error
		return ctrl.Result{}, err
	}

	pvName := nfspvc.Name + "-" + nfspvc.Namespace

	pv := &corev1.PersistentVolume{}
	err = r.Get(ctx, types.NamespacedName{Name: pvName}, pv)
	if err != nil && errors.IsNotFound(err) {
		// PV doesn't exist, so we'll create it
		// When creating the PV, use 'pvName' as its name.
		// ...
		pv = &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(nfspvc.Spec.Capacity),
				},
				AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode(nfspvc.Spec.AccessModes)},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					NFS: &corev1.NFSVolumeSource{
						Server:   nfspvc.Spec.Server,
						Path:     nfspvc.Spec.Path,
						ReadOnly: false, // set to true if you want read-only
					},
				},
				ClaimRef: &corev1.ObjectReference{
					Namespace: nfspvc.Namespace,
					Name:      nfspvc.Name,
				},
			},
		}
		if err := r.Create(ctx, pv); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, pvc)
	if err != nil && errors.IsNotFound(err) {
		pvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nfspvc.Name,
				Namespace: nfspvc.Namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, // You need to map this from your CR
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(nfspvc.Spec.Capacity),
					},
				},
				VolumeName: pvName,
			},
		}
		if err := r.Create(ctx, pvc); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	nfspvc.Status.PvcStatus = danaiov1.PvcStatusCreated
	if err := r.Status().Update(ctx, nfspvc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NfsPvcReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1.NfsPvc{}).
		Complete(r)
}
