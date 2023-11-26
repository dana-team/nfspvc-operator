package sync

import (
	"context"
	"fmt"

	"os"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	status_utils "github.com/dana-team/nfspvc-operator/internal/controller/utils/status"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NfsPvcDanaLabel         = "nfspvc.dana.io/nfspvc-owner"
	PvcBindStatusAnnotation = "pv.kubernetes.io/bind-completed"

	STORAGE_CLASS_ENV  = "STORAGE_CLASS"
	RECLAIM_POLICY_ENV = "RECLAIM_POLICY"
)

var StorageClass = os.Getenv(STORAGE_CLASS_ENV)
var ReclaimPolicy = os.Getenv(RECLAIM_POLICY_ENV)

func SyncNfsPvc(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	if nfspvc.ObjectMeta.DeletionTimestamp == nil {
		if err := createOrUpdateStorageObjects(ctx, nfspvc, log, k8sClient); err != nil {
			return err
		}
	}

	if err := status_utils.SyncNfsPvcStatus(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	return nil
}

func createOrUpdateStorageObjects(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	if err := handlePVState(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	if err := handlePVCState(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	return nil

}

func handlePVState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	pv := corev1.PersistentVolume{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get pv - "+nfspvc.Name+"-"+nfspvc.Namespace+"-pv")
			return err
		}
		pvFromNfsPvc := preparePV(nfspvc)
		if err := k8sClient.Create(ctx, &pvFromNfsPvc); err != nil {
			return fmt.Errorf("failed to create pv: %s", err.Error())
		}
		return nil
	}

	isClaimRefPVCDeleted, err := isConnectedPVCDeleted(ctx, k8sClient, pv, nfspvc)
	if err != nil {
		return fmt.Errorf("failed to fetch claimRef PVC: %s", err.Error())
	}
	if isClaimRefPVCDeleted {
		claimRefForPv := &corev1.ObjectReference{
			Name:      nfspvc.Name,
			Namespace: nfspvc.Namespace,
			Kind:      corev1.ResourcePersistentVolumeClaims.String(),
		}
		// Use retry on conflict to update the PV.
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			pv.Spec.ClaimRef = claimRefForPv
			updateErr := k8sClient.Update(ctx, &pv)
			if errors.IsConflict(updateErr) {
				// Conflict occurred, let's re-fetch the latest version of PV and retry the update.
				if getErr := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); getErr != nil {
					return getErr
				}
			}
			return updateErr
		})
		return err
	}
	return nil
}

func handlePVCState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			pvcFromNfsPvc := preparePVC(nfspvc)
			if err := k8sClient.Create(ctx, &pvcFromNfsPvc); err != nil {
				return fmt.Errorf("failed to create pvc: %s", err.Error())
			}
			return nil
		} else {
			return fmt.Errorf("failed to fetch pvc: %s", err.Error())
		}
	}

	if pvc.Status.Phase == corev1.ClaimLost { // if the pvc's phase is 'lost', so probably the associated pv was deleted. In order to fix that the "bind" annotation needs to be deleted.
		bindStatus, ok := pvc.ObjectMeta.Annotations[PvcBindStatusAnnotation]
		if ok && bindStatus == "yes" {
			// Use retry on conflict to update the PVC.
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				delete(pvc.ObjectMeta.Annotations, PvcBindStatusAnnotation)
				updateErr := k8sClient.Update(ctx, &pvc)
				if errors.IsConflict(updateErr) {
					// Conflict occurred, let's re-fetch the latest version of PVC and retry the update.
					if getErr := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); getErr != nil {
						return getErr
					}
				}
				return updateErr
			})
			return err
		}
		return nil
	}
	return nil
}

func preparePVC(nfspvc danaiov1alpha1.NfsPvc) corev1.PersistentVolumeClaim {
	storageClass := StorageClass
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfspvc.Name,
			Namespace: nfspvc.Namespace,
			Labels: map[string]string{
				NfsPvcDanaLabel: nfspvc.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			VolumeName:       nfspvc.Name + "-" + nfspvc.Namespace + "-pv",
			AccessModes:      nfspvc.Spec.AccessModes,
			Resources: corev1.ResourceRequirements{
				Requests: nfspvc.Spec.Capacity,
			},
		},
	}
	return pvc
}

func preparePV(nfspvc danaiov1alpha1.NfsPvc) corev1.PersistentVolume {
	pv := corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv",
			Labels: map[string]string{
				NfsPvcDanaLabel: nfspvc.Name,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName:              StorageClass,
			Capacity:                      nfspvc.Spec.Capacity,
			AccessModes:                   nfspvc.Spec.AccessModes,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimPolicy(ReclaimPolicy),
			ClaimRef: &corev1.ObjectReference{
				Name:      nfspvc.Name,
				Namespace: nfspvc.Namespace,
				Kind:      corev1.ResourcePersistentVolumeClaims.String(),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: nfspvc.Spec.Server,
					Path:   nfspvc.Spec.Path,
				},
			},
		},
	}
	return pv
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
	return (PV.Status.Phase == corev1.VolumeReleased)
}

// isPVFailed returns true if the PV phase is Failed.
func isPVFailed(PV corev1.PersistentVolume) bool {
	return (PV.Status.Phase == corev1.VolumeFailed)
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
