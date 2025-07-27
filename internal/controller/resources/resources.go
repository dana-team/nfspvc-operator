package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/dana-team/nfspvc-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrFailedCleanup = errors.New("failed nfspvc cleanup")

// HandleDelete ensures the deletion of the nfspvc.
func HandleDelete(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) (bool, error) {
	if controllerutil.ContainsFinalizer(&nfspvc, utils.NfsPvcDeletionFinalizer) {
		pvcDeleted, pvDeleted, err := areResourceDeleted(ctx, nfspvc, k8sClient)
		if err != nil {
			return false, err
		}
		if pvDeleted && pvcDeleted {
			return true, nil
		} else if !pvDeleted && pvcDeleted {
			if err := cleanup(ctx, nfspvc.Name, nfspvc.Namespace, k8sClient); err != nil {
				return false, err
			}
			pvName := nfspvc.Name + "-" + nfspvc.Namespace + "-pv"
			return false, fmt.Errorf("pv %q has not been deleted yet: %w", pvName, ErrFailedCleanup)
		}
		if err := cleanup(ctx, nfspvc.Name, nfspvc.Namespace, k8sClient); err != nil {
			return false, err
		}
	}
	return false, nil
}

// deleteResource gets resource to delete and delete that resource from the cluster.
func deleteResource(ctx context.Context, resource client.Object, k8sClient client.Client) error {
	if err := k8sClient.Delete(ctx, resource); client.IgnoreNotFound(err) != nil {
		return err
	}
	return nil
}

// cleanup deletes the pvc and the pv that related to the nfspvc.
func cleanup(ctx context.Context, nfsPvcName string, nfsPvcNamespace string, k8sClient client.Client) error {
	pvc := &corev1.PersistentVolumeClaim{}
	pvcDeleted, err := isDeleted(ctx, k8sClient, pvc, types.NamespacedName{Name: nfsPvcName, Namespace: nfsPvcNamespace})
	if err != nil {
		return err
	}
	if !pvcDeleted {
		if err := deleteResource(ctx, pvc, k8sClient); err != nil {
			return fmt.Errorf("failed to delete pvc %q: %v", nfsPvcName, err)
		}
	}
	pvName := nfsPvcName + "-" + nfsPvcNamespace + "-pv"
	pv := &corev1.PersistentVolume{}
	pvDeleted, err := isDeleted(ctx, k8sClient, pv, types.NamespacedName{Name: pvName})
	if err != nil {
		return err
	}
	if !pvDeleted {
		if err := deleteResource(ctx, pv, k8sClient); err != nil {
			return fmt.Errorf("failed to delete pv %q: %v", pvName, err)
		}
	}
	return nil
}

// areResourcesDeleted checks if the underlying PV and PVC of an nfspvc are deleted.
func areResourceDeleted(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) (bool, bool, error) {
	pvName := nfspvc.Name + "-" + nfspvc.Namespace + "-pv"
	pvc := corev1.PersistentVolumeClaim{}
	pv := corev1.PersistentVolume{}
	pvcDeleted, err := isDeleted(ctx, k8sClient, &pvc, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name})
	if err != nil {
		return false, false, err
	}

	pvDeleted, err := isDeleted(ctx, k8sClient, &pv, types.NamespacedName{Name: pvName})
	if err != nil {
		return false, false, err
	}

	return pvcDeleted, pvDeleted, nil
}

// isDeleted checks if the given object exists in the cluster.
func isDeleted(ctx context.Context, k8sClient client.Client, k8sObject client.Object, namespacedName types.NamespacedName) (bool, error) {
	if err := k8sClient.Get(ctx, namespacedName, k8sObject); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
