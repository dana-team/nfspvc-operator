package finalizer

import (
	"context"
	"fmt"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FinalizerDeletionNfsPVc = "nfspvc.dana.io/nfspvc-protection"

type FailedCleanUpError struct {
	Message string
}

// Error implements the error interface for FailedCleanUpError.
func (e *FailedCleanUpError) Error() string {
	return e.Message
}

// IsFailedCleanUp returns true if the specified error is FailedCleanUpError.
func IsFailedCleanUp(err error) bool {
	if _, ok := err.(*FailedCleanUpError); ok {
		// This means the error is of type *FailedCleanUpError.
		return true
	}
	return false
}

// HandleResourceDeletion ensures the deletion of the nfspvc.
func HandleResourceDeletion(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) (error, bool) {
	if nfspvc.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&nfspvc, FinalizerDeletionNfsPVc) {
			// check if the pv and the pvc are deleted.
			pvcDeleted := false
			pvc := corev1.PersistentVolumeClaim{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "unable to get pvc - "+nfspvc.Name)
					return err, false
				}
				pvcDeleted = true
			}
			pvDeleted := false
			pv := corev1.PersistentVolume{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "unable to get pv - "+nfspvc.Name+"-"+nfspvc.Namespace+"-pv")
					return err, false
				}
				pvDeleted = true
			}

			if pvDeleted && pvcDeleted { // if the pv and the pvc were successfully deleted, remove the finalizer.
				return RemoveFinalizer(ctx, nfspvc, log, k8sClient), true
			} else if !pvDeleted && pvcDeleted { // if only the pvc was deleted successfully, the reconcile function will be triggered again for the pv cleanup.
				// if the pv still exists, delete it.
				if err := nfsPvcCleanUp(ctx, nfspvc, log, k8sClient); err != nil {
					return err, false
				}
				pvName := nfspvc.Name + "-" + nfspvc.Namespace + "-pv"
				return &FailedCleanUpError{Message: "the pv " + pvName + " is not deleted yet"}, false
			}

			// if the pv and the pvc still exist, delete them.
			if err := nfsPvcCleanUp(ctx, nfspvc, log, k8sClient); err != nil {
				return err, false
			}
		} else {
			return nil, true
		}
	}
	return nil, false
}

// nfsPvcCleanUp cleanup the pvc and the pv that related to the nfspvc.
func nfsPvcCleanUp(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := deleteResource(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, pvc, log, k8sClient); err != nil {
		return fmt.Errorf("failed to delete pvc - %s: %s", nfspvc.Name, err.Error())
	}

	pv := &corev1.PersistentVolume{}
	if err := deleteResource(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, pv, log, k8sClient); err != nil {
		return fmt.Errorf("failed to delete pv - %s: %s", nfspvc.Name+"-"+nfspvc.Namespace+"-pv", err.Error())
	}

	return nil
}

// deleteResource get resource to delete and delete that resource from the cluster.
func deleteResource(ctx context.Context, namespacedName types.NamespacedName, resource client.Object, log logr.Logger, k8sClient client.Client) error {
	if err := k8sClient.Get(ctx, namespacedName, resource); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get resource")
			return err
		}
		return nil
	}
	if err := k8sClient.Delete(ctx, resource); client.IgnoreNotFound(err) != nil {
		log.Error(err, "unable to delete resource")
		return err
	}
	return nil
}

// removeFinalizer remove the dana finalizer from the nfspvc object.
func RemoveFinalizer(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	controllerutil.RemoveFinalizer(&nfspvc, FinalizerDeletionNfsPVc)
	if err := k8sClient.Update(ctx, &nfspvc); err != nil {
		log.Error(err, "unable to remove the finalizer from the NfsPvc - "+nfspvc.Name)
		return err
	}
	return nil
}

// ensureFinalizer ensures the nfspvc has the finalizer.
func EnsureFinalizer(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client, log logr.Logger) error {
	if !controllerutil.ContainsFinalizer(&nfspvc, FinalizerDeletionNfsPVc) {
		controllerutil.AddFinalizer(&nfspvc, FinalizerDeletionNfsPVc)
		if err := k8sClient.Update(ctx, &nfspvc); err != nil {
			log.Error(err, "unable to add the finalizer to the nfspvc - "+nfspvc.Name)
			return err
		}
	}
	return nil
}
