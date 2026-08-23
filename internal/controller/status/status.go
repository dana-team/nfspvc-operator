package status

import (
	"context"
	"fmt"

	"github.com/dana-team/nfspvc-operator/internal/controller/utils"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	phaseUnknown  = "Unknown"
	phaseNotFound = "NotFound"

	reasonBound    = "Bound"
	reasonNotReady = "NotReady"
)

// Update fetches the phase of the pv and the pvc that is created by the nfspvc and updates the nfspvc status.
func Update(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	pvcPhase := getPVCStatus(ctx, nfspvc, k8sClient)
	pvPhase := getPVStatus(ctx, nfspvc, k8sClient)
	if pvcPhase != nfspvc.Status.PvcPhase || pvPhase != nfspvc.Status.PvPhase {
		return ensure(ctx, pvcPhase, pvPhase, &nfspvc, k8sClient)
	}
	return nil
}

// ensure updates the status of the nfspvc to match the state of the underlying PVC.
func ensure(ctx context.Context, pvcPhase, pvPhase string, nfspvc *danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, nfspvc); err != nil {
		return err
	}

	return utils.RetryOnConflictUpdate(ctx, k8sClient, nfspvc, nfspvc.Name, nfspvc.Namespace, func(obj *danaiov1alpha1.NfsPvc) error {
		obj.Status.PvcPhase = pvcPhase
		obj.Status.PvPhase = pvPhase
		setReadyCondition(obj, pvcPhase, pvPhase)
		return k8sClient.Status().Update(ctx, obj)
	})
}

func setReadyCondition(nfspvc *danaiov1alpha1.NfsPvc, pvcPhase, pvPhase string) {
	ready := pvcPhase == string(corev1.ClaimBound) && pvPhase == string(corev1.VolumeBound)

	condition := metav1.Condition{
		Type:               danaiov1alpha1.ConditionReady,
		ObservedGeneration: nfspvc.Generation,
	}

	if ready {
		condition.Status = metav1.ConditionTrue
		condition.Reason = reasonBound
		condition.Message = "PV and PVC are bound"
	} else {
		condition.Status = metav1.ConditionFalse
		condition.Reason = reasonNotReady
		condition.Message = fmt.Sprintf("PVC phase: %s, PV phase: %s", pvcPhase, pvPhase)
	}

	meta.SetStatusCondition(&nfspvc.Status.Conditions, condition)
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
