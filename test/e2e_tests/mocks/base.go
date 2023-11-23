package mocks

import (
	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	NsName = "nfspvc-e2e-tests"
)

func CreateBaseNfsPvc() *nfspvcv1alpha1.NfsPvc {
	return &nfspvcv1alpha1.NfsPvc{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NfsPvc",
			APIVersion: "nfspvc.dana.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nfspvc-default-test",
			Namespace: NsName,
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
