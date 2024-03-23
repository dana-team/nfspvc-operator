package mocks

import (
	"os"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	NSName     = "nfspvc-e2e-tests"
	NFSPVCName = "nfspvc-default-test"
)

func CreateBaseNfsPvc() *nfspvcv1alpha1.NfsPvc {
	return &nfspvcv1alpha1.NfsPvc{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NfsPvc",
			APIVersion: "nfspvc.dana.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NFSPVCName,
			Namespace: NSName,
		},
		Spec: nfspvcv1alpha1.NfsPvcSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteMany"},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
			Path:   "/test",
			Server: "vs-koki",
		},
	}
}

func CreateBasePVC(pvcName string) *corev1.PersistentVolumeClaim {
	storageClass := os.Getenv("STORAGE_CLASS")
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: NSName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			VolumeName:       pvcName + "-" + NSName + "-pv",
			AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
			},
		},
	}
}
