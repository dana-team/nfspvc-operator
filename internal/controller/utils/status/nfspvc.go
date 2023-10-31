package status

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	danaiov1alpha1 "dana.io/nfs-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PvcPhaseUnknown = "Unknown"
)

func SyncNfsPvcStatus(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {

	nfspvcObject := danaiov1alpha1.NfsPvc{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &nfspvcObject); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Didn't find NfsPvc: %s, from the namespace: %s", nfspvc.Name, nfspvc.Namespace))
			return nil
		}
		return fmt.Errorf("failed to get NfsPvc: %s", err.Error())
	}

	pvc := corev1.PersistentVolumeClaim{}

	pvcPhase := ""
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			pvcPhase = PvcPhaseUnknown
		} else {
			return fmt.Errorf("failed to fetch Pvc: %s", err.Error())
		}
	} else {
		pvcPhase = string(pvc.Status.Phase)
	}

	if pvcPhase != "" && pvcPhase != PvcPhaseUnknown && nfspvcObject.Status.PvcPhase != pvcPhase {
		nfspvcObject.Status.PvcPhase = pvcPhase
		if err := k8sClient.Status().Update(ctx, &nfspvcObject); err != nil {
			return fmt.Errorf("failed to update NfsPvc status: %s", err.Error())
		}
	}

	return nil
}
