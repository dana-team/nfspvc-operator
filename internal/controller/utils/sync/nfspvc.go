package sync

import (
	"context"
	"fmt"

	"os"

	danaiov1alpha1 "dana.io/nfs-operator/api/v1alpha1"
	status_utils "dana.io/nfs-operator/internal/controller/utils/status"
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
)

var StorageClass = os.Getenv("STORAGE_CLASS")
var ReclaimPolicy = os.Getenv("RECLAIM_POLICY")

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
	if err := handlePvState(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	if err := handlePvcState(ctx, nfspvc, log, k8sClient); err != nil {
		return err
	}

	return nil

}

func handlePvState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	pv := corev1.PersistentVolume{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace + "-pv"}, &pv); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get pv - "+nfspvc.Name+"-"+nfspvc.Namespace+"-pv")
			return err
		}
		pvFromNfsPvc := preparePv(nfspvc)
		if err := k8sClient.Create(ctx, &pvFromNfsPvc); err != nil {
			return fmt.Errorf("failed to create pv: %s", err.Error())
		}
		return nil
	}

	if pv.Status.Phase == corev1.VolumeReleased || pv.Status.Phase == corev1.VolumeFailed ||
		(nfspvc.Status.PvPhase == string(corev1.VolumeBound) && nfspvc.Status.PvcPhase == string(corev1.ClaimPending)) {

		claimRefForPv := &corev1.ObjectReference{
			Name:      nfspvc.Name,
			Namespace: nfspvc.Namespace,
			Kind:      corev1.ResourcePersistentVolumeClaims.String(),
		}
		pv.Spec.ClaimRef = claimRefForPv
		// Use retry on conflict to update the PV
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			updateErr := k8sClient.Update(ctx, &pv)
			if errors.IsConflict(updateErr) {
				// Conflict occurred, let's re-fetch the latest version of PV and retry the update
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

func handlePvcState(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: nfspvc.Namespace, Name: nfspvc.Name}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			pvcFromNfsPvc := preparePvc(nfspvc)
			if err := k8sClient.Create(ctx, &pvcFromNfsPvc); err != nil {
				return fmt.Errorf("failed to create pvc: %s", err.Error())
			}
			return nil
		} else {
			return fmt.Errorf("failed to fetch pvc: %s", err.Error())
		}
	}

	if pvc.Status.Phase == corev1.ClaimLost { //if the pvc's phase is 'lost', so probably the associated pv was deleted. In order to fix that the "bind" annotation needs to be deleted.
		bindStatus, ok := pvc.ObjectMeta.Annotations[PvcBindStatusAnnotation]
		if ok && bindStatus == "yes" {
			delete(pvc.ObjectMeta.Annotations, PvcBindStatusAnnotation)
			// Use retry on conflict to update the PVC
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				updateErr := k8sClient.Update(ctx, &pvc)
				if errors.IsConflict(updateErr) {
					// Conflict occurred, let's re-fetch the latest version of PVC and retry the update
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

func preparePvc(nfspvc danaiov1alpha1.NfsPvc) corev1.PersistentVolumeClaim {
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

func preparePv(nfspvc danaiov1alpha1.NfsPvc) corev1.PersistentVolume {
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
