package sync

import (
	"context"
	"fmt"
	"reflect"

	danaiov1alpha1 "dana.io/nfs-operator/api/v1alpha1"
	status_utils "dana.io/nfs-operator/internal/controller/utils/status"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NfsPvcDanaLabel   = "nfspvc.dana.io/nfspvc-owner"
	StorageClassBrown = "brown"
	PvFailedPhase     = "Failed"
)

func SyncNfsPvc(ctx context.Context, nfspvc danaiov1alpha1.NfsPvc, log logr.Logger, k8sClient client.Client) error {

	if err := createOrUpdateStorageObjects(ctx, nfspvc, log, k8sClient); err != nil {
		return err
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
	} else {
		if !reflect.DeepEqual(nfspvc.Spec.Capacity, pvc.Spec.Resources.Requests) {
			pvc.Spec.Resources.Requests = nfspvc.Spec.Capacity
			if err := k8sClient.Update(ctx, &pvc); err != nil {
				return fmt.Errorf("unable to update Pvc of NfsPvc: %s", err.Error())
			}
			return nil
		}
		return nil
	}
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
	}

	if PvFailedPhase == pv.Status.Phase {
		if pv.ObjectMeta.DeletionTimestamp == nil{
			if err := k8sClient.Delete(ctx, &pv); client.IgnoreNotFound(err) != nil {
				return err
			}
		}else{
			//to do reque
			//adding pv is exist status
		}		
	}

	return nil
}

func preparePvc(nfspvc danaiov1alpha1.NfsPvc) corev1.PersistentVolumeClaim {
	storageClass := StorageClassBrown
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
			StorageClassName:              StorageClassBrown,
			Capacity:                      nfspvc.Spec.Capacity,
			AccessModes:                   nfspvc.Spec.AccessModes,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
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
