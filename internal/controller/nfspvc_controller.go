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
	"strings"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"

	danaiov1alpha1 "dana.io/nfs-operator/api/v1alpha1"

	nfspvcutils "dana.io/nfs-operator/internal/controller/utils"
)

// NfsPvcReconciler reconciles a NfsPvc object
type NfsPvcReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

const (
	NfsPvcFinalizerName     = "nfspvc.dana.io/finalizer"
	pvcBindStatusAnnotation = "pv.kubernetes.io/bind-completed"
)

// SetupWithManager sets up the controller with the Manager.
func (r *NfsPvcReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NfsPvc{}).
		Watches(&corev1.PersistentVolume{}, handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromPersistentVolume)).
		Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromPersistentVolumeClaim)).
		Complete(r)
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
	logger := r.Log
	// Fetch the NfsPvc instance
	nfspvc := &danaiov1alpha1.NfsPvc{}
	err := r.Get(ctx, req.NamespacedName, nfspvc)

	defer r.handleReconcileError(ctx, nfspvc, err)

	// Handle the error if any during the fetch
	if err != nil {
		if errors.IsNotFound(err) {
			// If the resource is not found, it might've been deleted after the reconcile request.
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
		if err := HandleDeletion(ctx, nfspvc, r); err != nil {
			logger.Error(err, "Failed to handle deletion") // Logging the error
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		logger.Info("debug pv and pvc deletion", "pv pvc", "pvc")
		return ctrl.Result{}, nil
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if err := HandleCreation(ctx, nfspvc, r); err != nil {
		logger.Error(err, "Failed to handle creation") // Logging the error
		return ctrl.Result{}, err
	}

	logger.Info("after handle creation")

	if err := HandleUpdate(ctx, nfspvc, r); err != nil {
		logger.Error(err, "Failed to handle Update") // Logging the error
		return ctrl.Result{}, err
	}

	logger.Info("after handle update")

	pvName := nfspvc.Name + "-" + nfspvc.Namespace

	pv := &corev1.PersistentVolume{}
	err = r.Get(ctx, types.NamespacedName{Name: pvName}, pv)

	if err != nil && errors.IsNotFound(err) {
		// PV doesn't exist, so we'll create it
		// When creating the PV, use 'pvName' as its name.
		// ...
		logger.Info("hello", "blablab", nfspvc.Spec.AccessModes)
		pv = &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity:                      nfspvc.Spec.Capacity,
				AccessModes:                   nfspvc.Spec.AccessModes,
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
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
	} else if err == nil {
		r.HandleExistingPv(ctx, pv)
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
					Requests: nfspvc.Spec.Capacity,
				},
				VolumeName: pvName,
			},
		}
		if err := r.Create(ctx, pvc); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else if err == nil {
		r.HandleExistingPvc(ctx, pvc)
	}

	err = r.Get(ctx, req.NamespacedName, nfspvc)
	nfspvc.Status.PvcStatus = danaiov1alpha1.PvcStatusCreated
	if err := r.Status().Update(ctx, nfspvc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Delete the pv associated to the NfsPvc
func (r *NfsPvcReconciler) deletePv(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc) error {
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

	return nil
}

// This function responsible to handle errors in the reconcile in update the NfsPvc accordingly
func (r *NfsPvcReconciler) handleReconcileError(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, err error) {
	logger := r.Log
	if err != nil {
		nfspvc.Status.PvcStatus = danaiov1alpha1.PvcStatusError
		updateErr := r.Status().Update(ctx, nfspvc)
		if updateErr != nil {
			logger.Error(updateErr, "Failed to update NfsPvc status to error.")
		}
	}
}

// Delete the pvc associated to the NfsPvc
func (r *NfsPvcReconciler) deletePvc(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc) error {
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

	return nil
}

// Delete resources that the NfsPvc is responsible for: pv and pvc
func deleteAssociatedResources(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, r *NfsPvcReconciler) error {

	if err := r.deletePv(ctx, nfspvc); err != nil {
		return err
	}

	if err := r.deletePvc(ctx, nfspvc); err != nil {
		return err
	}

	return nil
}

// Handle the deletion phase of the NfsPvc object.
func HandleDeletion(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, r *NfsPvcReconciler) error {

	// Check if our finalizer is present
	if nfspvcutils.ContainsString(nfspvc.ObjectMeta.Finalizers, NfsPvcFinalizerName) {
		// Handle the actual cleanup of associated resources
		if err := deleteAssociatedResources(ctx, nfspvc, r); err != nil {
			return err
		}

		// Remove the finalizer after cleanup
		nfspvc.ObjectMeta.Finalizers = nfspvcutils.RemoveString(nfspvc.ObjectMeta.Finalizers, NfsPvcFinalizerName)
		if err := r.Update(ctx, nfspvc); err != nil {
			return err
		}

	}
	return nil
}

// HandleCreation handles the creation phase, including adding finalizers.
func HandleCreation(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, r *NfsPvcReconciler) error {
	// If PVC status is empty, set it to Pending
	if nfspvc.Status.PvcStatus == "" {
		nfspvc.Status.PvcStatus = danaiov1alpha1.PvcStatusPending
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

// HandleUpdate handles nfspvc edit situation
func HandleUpdate(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, r *NfsPvcReconciler) error {
	// I need to check if someone edited the nfspvc object
	// first i will check if there are already bounded pv:
	pvName := nfspvc.Name + "-" + nfspvc.Namespace
	pv := &corev1.PersistentVolume{}
	err := r.Get(ctx, types.NamespacedName{Name: pvName}, pv)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if err == nil && pv.ObjectMeta.DeletionTimestamp.IsZero() {
		// the pv exists so i need to compare the nfspvc relevant field which are:
		// accessmodes, capacity, path, server
		// with the pv and check if there is a difference
		// if there is a difference it means that the nfspvc was updated
		pvAccessModes := pv.Spec.AccessModes[0]
		pvCapacity := pv.Spec.Capacity.Storage()
		pvPath := pv.Spec.NFS.Path
		pvServer := pv.Spec.NFS.Server
		isNfsPvcUpdated := pvAccessModes != nfspvc.Spec.AccessModes[0] ||
			!nfspvc.Spec.Capacity.Storage().Equal(*pvCapacity) ||
			pvPath != nfspvc.Spec.Path ||
			pvServer != nfspvc.Spec.Server
		// if the nfspvc object was updated then delete the pv and pvc,
		// this deletion will trigger pv and pvc recreation with the new fields
		if isNfsPvcUpdated {
			deleteAssociatedResources(ctx, nfspvc, r)
		}
	}

	return nil
}

// This function handles the scenario where the pv associated with the nfspvc is already exists
func (r *NfsPvcReconciler) HandleExistingPv(ctx context.Context, pv *corev1.PersistentVolume) error {

	// If pv is in failed status then delete it which will trigger recreation of the pv
	if pv.Status.Phase == "Failed" {
		if err := r.Delete(ctx, pv); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

// This function handles the scenario where the pvc associated with the nfspvc is already exists
func (r *NfsPvcReconciler) HandleExistingPvc(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	// If pvc is lost and was already bind delete the annotation
	bindStatus, ok := pvc.ObjectMeta.Annotations[pvcBindStatusAnnotation]
	if pvc.Status.Phase == "Lost" && ok && bindStatus == "yes" {
		delete(pvc.ObjectMeta.Annotations, pvcBindStatusAnnotation)
		return r.Update(ctx, pvc)
	}

	return nil
}

// This function triggers reconcile on the nfspvc when the associated pv changes
func (r *NfsPvcReconciler) enqueueRequestsFromPersistentVolume(ctx context.Context, object client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	pv := object.(*corev1.PersistentVolume)
	var requests []reconcile.Request

	// pvName is determined this way: nfspvc.Name + "-" + nfspvc.Namespace
	// so in order to find out the name of the related nfspvc and namespace the name
	// of the pv is splited by '-'

	pvName := pv.ObjectMeta.Name

	splitedPvName := strings.Split(pvName, "-")

	nfspvcName := splitedPvName[0]

	nfspvcNamespace := splitedPvName[1]

	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      nfspvcName,
			Namespace: nfspvcNamespace,
		},
	})

	// Log the number of requests enqueued for the given Namespace
	logger.Info("Enqueued requests for namespace", "Namespace", nfspvcNamespace, "nfspvc Name", nfspvcName)

	return requests
}

// This function triggers reconcile on the nfspvc when the associated pvc changes
func (r *NfsPvcReconciler) enqueueRequestsFromPersistentVolumeClaim(ctx context.Context, object client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	pvc := object.(*corev1.PersistentVolumeClaim)
	var requests []reconcile.Request

	// pvName is determined this way: nfspvc.Name + "-" + nfspvc.Namespace
	// so in order to find out the name of the related nfspvc and namespace i will split the name
	// of the pv by '-'

	nfspvcName := pvc.Name

	nfspvcNamespace := pvc.Namespace

	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      nfspvcName,
			Namespace: nfspvcNamespace,
		},
	})

	// Log the number of requests enqueued for the given Namespace
	logger.Info("Enqueued requests for namespace", "Namespace", nfspvcNamespace, "nfspvc Name", nfspvcName)

	return requests
}
