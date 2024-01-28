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
	"fmt"
	"time"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	finalizerutils "github.com/dana-team/nfspvc-operator/internal/controller/utils/finalizer"
	statusutils "github.com/dana-team/nfspvc-operator/internal/controller/utils/status"
	syncutils "github.com/dana-team/nfspvc-operator/internal/controller/utils/sync"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const RequeueIntervalSeconds = 4

// NfsPvcReconciler reconciles a NfsPvc object
type NfsPvcReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (r *NfsPvcReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&danaiov1alpha1.NfsPvc{}).
		Watches(&corev1.PersistentVolumeClaim{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromPersistentVolumeClaim),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs/finalizers,verbs=update

func (r *NfsPvcReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("NfsPvc", req.Name, "NfsPvcNamespace", req.Namespace)
	logger.Info("Starting Reconcile")

	nfspvc := danaiov1alpha1.NfsPvc{}
	if err := r.Client.Get(ctx, req.NamespacedName, &nfspvc); err != nil {
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Didn't find NfsPvc: %s, from the namespace: %s", nfspvc.Name, nfspvc.Namespace))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get NfsPvc: %s", err.Error())
	}

	err, deleted := finalizerutils.HandleResourceDeletion(ctx, nfspvc, logger, r.Client)
	if err != nil {
		if finalizerutils.IsFailedCleanUp(err) {
			// this means the error is of type *FailedCleanUpError.
			logger.Info(fmt.Sprintf("failed to handle NfsPvc deletion: %s, so trying again in a few seconds", err.Error()))
			return ctrl.Result{RequeueAfter: time.Second * RequeueIntervalSeconds}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to handle NfsPvc deletion: %s", err.Error())
	}
	if deleted {
		return ctrl.Result{}, nil
	}
	if err := finalizerutils.EnsureFinalizer(ctx, nfspvc, r.Client, logger); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure finalizer in Capp: %s", err.Error())
	}

	// now sync the objects to the nfspvc object.
	if err := SyncNfsPvc(ctx, nfspvc, logger, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to sync NfsPvc: %s", err.Error())
	}

	return ctrl.Result{}, nil

}

// enqueueRequestsFromPersistentVolumeClaim reconciles the nfspvc when the associated pvc changes.
func (r *NfsPvcReconciler) enqueueRequestsFromPersistentVolumeClaim(ctx context.Context, pvc client.Object) []reconcile.Request {
	nfspvcList := &danaiov1alpha1.NfsPvcList{}
	listOps := &client.ListOptions{
		Namespace: pvc.GetNamespace(),
	}
	err := r.List(ctx, nfspvcList, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(nfspvcList.Items))
	for i, item := range nfspvcList.Items {
		if item.GetName() == pvc.GetName() {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
		}
	}

	if len(requests) == 0 {
		return []reconcile.Request{}
	}
	return requests
}

func SyncNfsPvc(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	if nfspvc.ObjectMeta.DeletionTimestamp == nil {
		if err := syncutils.CreateOrUpdateStorageObjects(ctx, nfspvc, log, k8sClient); err != nil {
			return err
		}
	}

	if err := statusutils.SyncNfsPvcStatus(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	return nil
}
