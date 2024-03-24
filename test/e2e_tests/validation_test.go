package e2e_tests

import (
	"context"

	mock "github.com/dana-team/nfspvc-operator/test/e2e_tests/mocks"
	"github.com/dana-team/nfspvc-operator/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/nfspvc-operator/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("validate NFSPVC webhook functionality ", func() {
	It("should deny creation of PVC with a name that already exists", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		pvc := mock.CreateBasePVC(baseNfsPvc.Name)

		By("creating a PVC")
		Expect(utilst.CreateResource(k8sClient, pvc)).Should(BeTrue())
		checkPvc := &corev1.PersistentVolumeClaim{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), client.ObjectKey{Name: baseNfsPvc.Name, Namespace: baseNfsPvc.Namespace}, checkPvc)

			return err == nil
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())

		By("creating NFSPVC with same name")
		Expect(utilst.CreateResource(k8sClient, baseNfsPvc)).Should(BeFalse())
	})

	It("should deny editing immutable fields", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)

		By("changing NFSPVC accessModes")
		nfspvcCopy := desiredNfsPvc.DeepCopy()
		nfspvcCopy.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"}
		err := utilst.UpdateResource(k8sClient, nfspvcCopy)
		Expect(err).To(HaveOccurred())

		By("changing NFSPVC capacity")
		nfspvcCopy = desiredNfsPvc.DeepCopy()
		nfspvcCopy.Spec.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("300Gi")}
		err = utilst.UpdateResource(k8sClient, nfspvcCopy)
		Expect(err).To(HaveOccurred())

		By("changing NFSPVC path")
		nfspvcCopy = desiredNfsPvc.DeepCopy()
		nfspvcCopy.Spec.Path = "/update"
		err = utilst.UpdateResource(k8sClient, nfspvcCopy)
		Expect(err).To(HaveOccurred())

		By("changing NFSPVC server")
		nfspvcCopy = desiredNfsPvc.DeepCopy()
		nfspvcCopy.Spec.Server = "vs-updated"
		err = utilst.UpdateResource(k8sClient, nfspvcCopy)
		Expect(err).To(HaveOccurred())
	})
})
