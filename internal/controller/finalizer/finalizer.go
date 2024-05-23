package finalizer

import (
	"context"

	"github.com/dana-team/nfspvc-operator/internal/controller/utils"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Remove removes a finalizer from the nfspvc object.
func Remove(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	controllerutil.RemoveFinalizer(&nfspvc, utils.NfsPvcDeletionFinalizer)
	if err := k8sClient.Update(ctx, &nfspvc); err != nil {
		return err
	}
	return nil
}

// Ensure adds a finalizer to the nfspvc object if one does not exist.
func Ensure(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	if !controllerutil.ContainsFinalizer(&nfspvc, utils.NfsPvcDeletionFinalizer) {
		controllerutil.AddFinalizer(&nfspvc, utils.NfsPvcDeletionFinalizer)
		if err := k8sClient.Update(ctx, &nfspvc); err != nil {
			return err
		}
	}
	return nil
}
