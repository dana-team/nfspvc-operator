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
	StoragePhaseUnknown  = "Unknown"
	StoragePhaseNotFound = "NotFound"
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

	pvcPhase := getPvcStatus(ctx, nfspvc, k8sClient)
	pvPhase := getPvStatus(ctx, nfspvc, k8sClient)

	if pvcPhase != nfspvc.Status.PvcPhase || pvPhase != nfspvc.Status.PvPhase {
		nfspvcObject.Status.PvcPhase = pvcPhase
		nfspvcObject.Status.PvPhase = pvPhase
		if err := k8sClient.Status().Update(ctx, &nfspvcObject); err != nil {
			return fmt.Errorf("failed to update NfsPvc status: %s", err.Error())
		}
	}

	return nil
}

// getPvcStatus return the Pvc's Phase
func getPvcStatus(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) string {

	pvc := corev1.PersistentVolumeClaim{}
	pvcPhase := ""
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			pvcPhase = StoragePhaseNotFound
		} else {
			pvcPhase = StoragePhaseUnknown
		}
	} else {
		pvcPhase = string(pvc.Status.Phase)
	}

	return pvcPhase
}

// getPvStatus return the Pv's Phase
func getPvStatus(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) string {

	pv := corev1.PersistentVolume{}
	pvPhase := ""
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
		if errors.IsNotFound(err) {
			pvPhase = StoragePhaseNotFound
		} else {
			pvPhase = StoragePhaseUnknown
		}
	} else {
		pvPhase = string(pv.Status.Phase)
	}

	return pvPhase
}
