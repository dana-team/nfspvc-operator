package status

import (
	"context"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	phaseUnknown  = "Unknown"
	phaseNotFound = "NotFound"
)

// Update fetches the phase of the pv and the pvc that is created by the nfspvc and updates the nfspvc status.
func Update(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	pvcPhase := getPVCStatus(ctx, nfspvc, k8sClient)
	pvPhase := getPVStatus(ctx, nfspvc, k8sClient)
	if pvcPhase != nfspvc.Status.PvcPhase || pvPhase != nfspvc.Status.PvPhase {
		return ensure(ctx, pvcPhase, pvPhase, nfspvc, k8sClient)
	}
	return nil
}

// ensure updates the status of the nfspvc to match the state of the underlying PVC.
func ensure(ctx context.Context, pvcPhase string, pvPhase string, nfspvcObject danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		nfspvcObject.Status.PvcPhase = pvcPhase
		nfspvcObject.Status.PvPhase = pvPhase
		updateErr := k8sClient.Status().Update(ctx, &nfspvcObject)
		if errors.IsConflict(updateErr) {
			if getErr := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvcObject.Name, Namespace: nfspvcObject.Namespace}, &nfspvcObject); getErr != nil {
				return getErr
			}
		}
		return updateErr
	})
	return err
}

// getPVCStatus returns the phase of the pvc.
func getPVCStatus(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) string {
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			return phaseNotFound
		}
		return phaseUnknown
	}
	return string(pvc.Status.Phase)
}

// getPVStatus returns the phase of the pv.
func getPVStatus(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) string {
	pv := corev1.PersistentVolume{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
		if errors.IsNotFound(err) {
			return phaseNotFound
		}
		return phaseUnknown
	}
	return string(pv.Status.Phase)
}
