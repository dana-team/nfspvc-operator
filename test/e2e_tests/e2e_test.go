package e2e_tests

import (
	"context"
	"time"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"

	mock "github.com/dana-team/nfspvc-operator/test/e2e_tests/mocks"
	utilst "github.com/dana-team/nfspvc-operator/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TimeoutNfsPvc          = 60 * time.Second
	NfsPvcCreationInterval = 5 * time.Second
)

var _ = Describe("validate Suite acted correctly ", func() {

	It("should have created a namespace", func() {
		ns := &corev1.Namespace{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: mock.NsName}, ns)
		Expect(err).NotTo(HaveOccurred())
		Expect(ns).NotTo(BeNil())
	})
})

var _ = Describe("validate NFSPVC adapter", func() {

	It("should succeed all adapter functions", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
		assertionNfsPvc := &nfspvcv1alpha1.NfsPvc{}
		By("checks unique creation of NFSPVC")
		Eventually(func() string {
			assertionNfsPvc = utilst.GetNfsPvc(k8sClient, desiredNfsPvc.Name, desiredNfsPvc.Namespace)
			return desiredNfsPvc.Name
		}, TimeoutNfsPvc, NfsPvcCreationInterval).ShouldNot(Equal(baseNfsPvc.Name), "should fetch NFSPVC.")

		By("checks if deleted successfully")
		utilst.DeleteNfsPvc(k8sClient, assertionNfsPvc)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, assertionNfsPvc)
		}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(BeFalse(), "should not find a resource.")
	})
})

var _ = Describe("E2E tests", func() {
	Context("with basic NFSPVC object", func() {
		It("should bound pv and pvc objects and deletion will cleanup", func() {
			By("creating NFSPVC object")
			baseNfsPvc := mock.CreateBaseNfsPvc()
			desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
			By("check if the pvc exists")
			Eventually(func() bool {
				pvc := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      desiredNfsPvc.Name,
						Namespace: desiredNfsPvc.Namespace,
					},
				}
				return utilst.DoesResourceExist(k8sClient, &pvc)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(BeTrue(), "should fetch pvc")
			By("check if the pv exists")
			Eventually(func() bool {
				pv := corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
					},
				}
				return utilst.DoesResourceExist(k8sClient, &pv)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(BeTrue(), "should fetch pv.")
			By("checking if the pv and the pvc are in bound phase")
			Eventually(func(g Gomega) {
				nfspvc := utilst.GetNfsPvc(k8sClient, desiredNfsPvc.Name, desiredNfsPvc.Namespace)
				g.Expect(nfspvc.Status.PvcPhase).To(Equal(string(corev1.ClaimBound)), "PvcPhase should be Bound")
				g.Expect(nfspvc.Status.PvPhase).To(Equal(string(corev1.VolumeBound)), "PvPhase should be Bound")
			}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(Succeed(), "Pv and Pvc Phases should be bound")
			time.Sleep(time.Second * 2)
			By("deleting the nfspvc")
			utilst.DeleteNfsPvc(k8sClient, desiredNfsPvc)
			time.Sleep(time.Second * 3)
			By("checking if the pv exists")
			Eventually(func() bool {
				pv := corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
					},
				}
				return utilst.DoesResourceExist(k8sClient, &pv)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(BeFalse(), "should not find a pv.")
			By("checking if the pvc exists")
			Eventually(func() bool {
				pvc := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      desiredNfsPvc.Name,
						Namespace: desiredNfsPvc.Namespace,
					},
				}
				return utilst.DoesResourceExist(k8sClient, &pvc)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).Should(BeFalse(), "should not find a pvc.")
		})
		time.Sleep(time.Second * 5)
		It("should sync the pv and pvc to the desired state", func() {
			By("creating nfspvc object")
			baseNfsPvc := mock.CreateBaseNfsPvc()
			desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
			time.Sleep(time.Second * 3)
			By("deleting the associated pvc")
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      desiredNfsPvc.Name,
					Namespace: desiredNfsPvc.Namespace,
				},
			}
			previousPvcUid := utilst.GetResourceUid(k8sClient, &pvc)
			Expect(k8sClient.Delete(context.Background(), &pvc)).To(Succeed(), "failed to delete pvc.")
			By("checking if the pvc recreated")
			Eventually(func() string {
				pvc := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      desiredNfsPvc.Name,
						Namespace: desiredNfsPvc.Namespace,
					},
				}
				return utilst.GetResourceUid(k8sClient, &pvc)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).ShouldNot(Equal(previousPvcUid), "should be a different pvc.")
			By("deleting the associated pv")
			pv := corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
				},
			}
			previousPvUid := utilst.GetResourceUid(k8sClient, &pv)
			Expect(k8sClient.Delete(context.Background(), &pv)).To(Succeed(), "failed to delete Pv")
			time.Sleep(time.Second * 4)
			By("checking if the pv recreated")
			Eventually(func() string {
				pv := corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
					},
				}
				return utilst.GetResourceUid(k8sClient, &pv)
			}, TimeoutNfsPvc, NfsPvcCreationInterval).ShouldNot(Equal(previousPvUid), "should be a different pv.")
			By("deleting the nfspvc")
			utilst.DeleteNfsPvc(k8sClient, desiredNfsPvc)
			time.Sleep(time.Second * 3)
		})
		time.Sleep(time.Second * 5)
	})
})

var _ = Describe("validate webhook acted correctly ", func() {

	Context("with unsupported access mode", func() {
		It("should be denied", func() {
			baseNfsPvc := mock.CreateBaseNfsPvc()
			baseNfsPvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{"not good"}
			Expect(utilst.CreateResource(k8sClient, baseNfsPvc)).Should(BeFalse())
		})
	})
	Context("when PVC with same name already exists", func() {
		It("should be denied", func() {
			baseNfsPvc := mock.CreateBaseNfsPvc()
			pvc := mock.CreateBasePVC(baseNfsPvc.Name)
			By("creating PVC")
			Expect(utilst.CreateResource(k8sClient, pvc)).Should(BeTrue())
			time.Sleep(time.Second * 3)
			By("creating NFSPVC with same name")
			Expect(utilst.CreateResource(k8sClient, baseNfsPvc)).Should(BeFalse())
		})
	})
	Context("updating a NFSPVC", func() {
		It("should be denied", func() {
			By("creating nfspvc object")
			baseNfsPvc := mock.CreateBaseNfsPvc()
			desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
			time.Sleep(time.Second * 3)
			desiredNfsPvc.Spec.Server = "vs-change"
			Expect(k8sClient.Update(context.Background(), desiredNfsPvc)).ToNot(Succeed(), "failed to update NFSPVC.")
		})
	})
})
