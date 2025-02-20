package resources

import (
	"context"
	"fmt"

	"github.com/dana-team/nfspvc-operator/internal/controller/utils"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pvcBindStatusAnnotation = "pv.kubernetes.io/bind-completed"
	desiredBindStatus       = "yes"
)

// HandleStorageObjectState handles the underlying PV and PVC when an NFSPVC is updated.
func HandleStorageObjectState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	if err := handlePVState(ctx, nfspvc, k8sClient); err != nil {
		return err
	}

	if err := handlePVCState(ctx, nfspvc, k8sClient); err != nil {
		return err
	}

	return nil

}

// handlePVState ensures the pv connected to an nfspvc exists and has a ClaimRef.
func handlePVState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	pv := corev1.PersistentVolume{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		pvFromNfsPvc := PreparePV(nfspvc, utils.StorageClass, utils.ReclaimPolicy)
		if err := k8sClient.Create(ctx, &pvFromNfsPvc); err != nil {
			return fmt.Errorf("failed to create pv %q: %v", pvFromNfsPvc.Name, err)
		}
		return nil
	}

	isClaimRefPVCDeleted, err := isConnectedPVCDeleted(ctx, k8sClient, pv, nfspvc)
	if err != nil {
		return fmt.Errorf("failed to fetch claimRef PVC: %s", err.Error())
	}
	if isClaimRefPVCDeleted {
		return UpdatePV(ctx, &nfspvc, k8sClient, &pv)
	}
	return nil
}

// handlePVCState ensures the pvc connected to an nfspvc exists and checks if it is bound to a pv.
func handlePVCState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, k8sClient client.Client) error {
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			pvcFromNfsPvc := PreparePVC(nfspvc, utils.StorageClass)
			if err := k8sClient.Create(ctx, &pvcFromNfsPvc); err != nil {
				return fmt.Errorf("failed to create pvc %q: %v", nfspvc.Name, err)
			}
			return nil
		} else {
			return fmt.Errorf("failed to fetch pvc %q: %v", nfspvc.Name, err)
		}
	}

	if pvc.Status.Phase == corev1.ClaimLost { // if the pvc's phase is 'lost', so probably the associated pv was deleted
		return deletePVCBindAnnotation(ctx, &nfspvc, k8sClient, &pvc)
	}
	return nil
}

// deletePVCBindAnnotation deletes the "bind" annotation from a pvc.
func deletePVCBindAnnotation(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, k8sClient client.Client, pvc *corev1.PersistentVolumeClaim) error {
	bindStatus, ok := pvc.Annotations[pvcBindStatusAnnotation]
	if ok && bindStatus == desiredBindStatus {
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, pvc); err != nil {
			return err
		}

		return utils.RetryOnConflictUpdate(ctx, k8sClient, pvc, nfspvc.Name, nfspvc.Namespace, func(obj *corev1.PersistentVolumeClaim) error {
			delete(obj.Annotations, pvcBindStatusAnnotation)
			return k8sClient.Update(ctx, obj)
		})
	}
	return nil
}

// isConnectedPVCDeleted returns if the connected PVC is deleted,
// connected PVC considered deleted when the PV phase is Released or Failed,
// or the PV phase in the nfspvc is bound, the PVC phase in the nfspvc is pending and the UID in the PV claimRef is different from the PVC UID.
func isConnectedPVCDeleted(ctx context.Context, k8sClient client.Client, PV corev1.PersistentVolume, nfspvc danaiov1alpha1.NfsPvc) (bool, error) {
	if isPVReleased(PV) {
		return true, nil
	}
	if isPVFailed(PV) {
		return true, nil
	}
	if isPVCInRecreation, err := isPVCInRecreationState(ctx, k8sClient, nfspvc, PV); err != nil {
		return false, err
	} else {
		return isPVCInRecreation, nil
	}
}

// isPVReleased returns true if the PV phase is Released.
func isPVReleased(PV corev1.PersistentVolume) bool {
	return PV.Status.Phase == corev1.VolumeReleased
}

// isPVFailed returns true if the PV phase is Failed.
func isPVFailed(PV corev1.PersistentVolume) bool {
	return PV.Status.Phase == corev1.VolumeFailed
}

// isPVCInRecreationState returns true if the PVC in recreation state, i.e. when
// the PV phase of the nfspvc is bound, and
// the PVC phase in the nfspvc is pending, and
// the UID in the PV claimRef is different from the PVC UID.
func isPVCInRecreationState(ctx context.Context, k8sClient client.Client, nfspvc danaiov1alpha1.NfsPvc, PV corev1.PersistentVolume) (bool, error) {
	isUIDEqual, err := isPVCUIDEqual(ctx, k8sClient, PV.Spec.ClaimRef.UID, nfspvc)
	if err != nil {
		return false, err
	}
	return nfspvc.Status.PvPhase == string(corev1.VolumeBound) &&
		nfspvc.Status.PvcPhase == string(corev1.ClaimPending) &&
		!isUIDEqual, nil
}

// isPVCUIDEqual returns true if the PVC uid and the claimRef uid of the PV are equal.
func isPVCUIDEqual(ctx context.Context, k8sClient client.Client, uid types.UID, nfspvc danaiov1alpha1.NfsPvc) (bool, error) {
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return pvc.GetUID() == uid, nil
}
