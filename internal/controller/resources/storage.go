package resources

import (
	"context"
	"fmt"

	"github.com/dana-team/nfspvc-operator/internal/controller/utils"

	danaiov1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nfsPvcDanaLabel = "nfspvc.dana.io/nfspvc-owner"
)

// PreparePVC returns a PVC with the given storageclass.
func PreparePVC(nfspvc danaiov1alpha1.NfsPvc, StorageClass string) corev1.PersistentVolumeClaim {
	storageClass := StorageClass
	return corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfspvc.Name,
			Namespace: nfspvc.Namespace,
			Labels: map[string]string{
				nfsPvcDanaLabel: nfspvc.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			VolumeName:       nfspvc.Name + "-" + nfspvc.Namespace + "-pv",
			AccessModes:      nfspvc.Spec.AccessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: nfspvc.Spec.Capacity,
			},
		},
	}
}

// PreparePV returns a PV with the given storageclass and reclaimpolicy.
func PreparePV(nfspvc danaiov1alpha1.NfsPvc, StorageClass string, ReclaimPolicy string) corev1.PersistentVolume {
	var pvName = nfspvc.Name + "-" + nfspvc.Namespace + "-pv"
	var mountOptions []string = nil

	if nfspvc.Spec.NfsVersion != "" {
		mountOptions = []string{fmt.Sprintf("nfsvers=%s", nfspvc.Spec.NfsVersion)}
	}

	return corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
			Labels: map[string]string{
				nfsPvcDanaLabel: nfspvc.Name,
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
			MountOptions: mountOptions,
		},
	}
}

// UpdatePV updates the PV claim reference when the NFSPVC is updated.
func UpdatePV(ctx context.Context, nfspvc *danaiov1alpha1.NfsPvc, k8sClient client.Client, pv *corev1.PersistentVolume) error {
	claimRefForPv := &corev1.ObjectReference{
		Name:      nfspvc.Name,
		Namespace: nfspvc.Namespace,
		Kind:      corev1.ResourcePersistentVolumeClaims.String(),
	}
	pvName := nfspvc.Name + "-" + nfspvc.Namespace + "-pv"

	if err := k8sClient.Get(ctx, types.NamespacedName{Name: pvName, Namespace: nfspvc.Namespace}, pv); err != nil {
		return err
	}

	return utils.RetryOnConflictUpdate(ctx, k8sClient, pv, pvName, nfspvc.Namespace, func(obj *corev1.PersistentVolume) error {
		obj.Spec.ClaimRef = claimRefForPv
		return k8sClient.Update(ctx, obj)
	})
}
