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

	"github.com/go-logr/logr"
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

	corev1 "k8s.io/api/core/v1"

	danaiov1alpha1 "dana.io/nfs-operator/api/v1alpha1"
	finalizer_utils "dana.io/nfs-operator/internal/controller/utils/finalizer"
	sync_utils "dana.io/nfs-operator/internal/controller/utils/sync"
)

const REQUEUE_INTERVAL_SECONDS = 4

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

	err, deleted := finalizer_utils.HandleResourceDeletion(ctx, nfspvc, logger, r.Client)
	if err != nil {
		if finalizer_utils.IsFailedCleanUp(err) {
			// this means the error is of type *FailedCleanUpError.
			logger.Info(fmt.Sprintf("failed to handle NfsPvc deletion: %s, so trying again in a few seconds", err.Error()))
			return ctrl.Result{RequeueAfter: time.Second * REQUEUE_INTERVAL_SECONDS}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to handle NfsPvc deletion: %s", err.Error())
	}
	if deleted {
		return ctrl.Result{}, nil
	}
	if err := finalizer_utils.EnsureFinalizer(ctx, nfspvc, r.Client, logger); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure finalizer in Capp: %s", err.Error())
	}

	// now sync the objects to the nfspvc object.
	if err := sync_utils.SyncNfsPvc(ctx, nfspvc, logger, r.Client); err != nil {
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
