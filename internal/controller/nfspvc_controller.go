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
	"errors"
	"fmt"
	"time"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"github.com/dana-team/nfspvc-operator/internal/controller/finalizer"
	"github.com/dana-team/nfspvc-operator/internal/controller/resources"
	"github.com/dana-team/nfspvc-operator/internal/controller/status"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nfspvc.dana.io,resources=nfspvcs/finalizers,verbs=update

func (r *NfsPvcReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("NfsPvc", req.Name, "NfsPvcNamespace", req.Namespace)
	logger.Info("Starting Reconcile")
	nfspvc := danaiov1alpha1.NfsPvc{}
	if err := r.Client.Get(ctx, req.NamespacedName, &nfspvc); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Didn't find NfsPvc")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get NfsPvc: %s", err.Error())
	}
	if nfspvc.ObjectMeta.DeletionTimestamp != nil {
		deleted, err := resources.HandleDelete(ctx, nfspvc, r.Client)
		if err != nil {
			if errors.Is(err, resources.FailedCleanupError) {
				logger.Info(fmt.Sprintf("failed to handle NfsPvc deletion: %s, so trying again in a few seconds", err.Error()))
				return ctrl.Result{RequeueAfter: time.Second * RequeueIntervalSeconds}, nil
			}
			return ctrl.Result{}, fmt.Errorf("failed to handle NfsPvc deletion: %s", err.Error())
		}
		if deleted {
			if err := finalizer.Remove(ctx, nfspvc, r.Client); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	if err := finalizer.Ensure(ctx, nfspvc, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure finalizer in NfsPvc: %s", err.Error())
	}
	if err := r.Update(ctx, nfspvc); err != nil {
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

// Update handles any update to an NFSPVC.
func (r *NfsPvcReconciler) Update(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc) error {
	if nfspvc.ObjectMeta.DeletionTimestamp == nil {
		if err := resources.HandleStorageObjectState(ctx, nfspvc, r.Client); err != nil {
			return err
		}
	}
	if err := status.Update(ctx, nfspvc, r.Client); err != nil {
		return err
	}
	return nil
}
