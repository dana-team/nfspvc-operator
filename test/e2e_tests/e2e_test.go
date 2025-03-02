package e2e_tests

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	"github.com/dana-team/nfspvc-operator/test/e2e_tests/testconsts"

	mock "github.com/dana-team/nfspvc-operator/test/e2e_tests/mocks"
	utilst "github.com/dana-team/nfspvc-operator/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("validate NFSPVC controller functionality", func() {
	It("should succeed all adapter functions", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
		assertionNfsPvc := &nfspvcv1alpha1.NfsPvc{}

		By("checking unique creation of NFSPVC")
		Eventually(func() string {
			assertionNfsPvc = utilst.GetNfsPvc(k8sClient, desiredNfsPvc.Name, desiredNfsPvc.Namespace)
			return desiredNfsPvc.Name
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(baseNfsPvc.Name), "should fetch NFSPVC.")

		By("checking NFSPVC is deleted successfully")
		utilst.DeleteNfsPvc(k8sClient, assertionNfsPvc)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, assertionNfsPvc)
		}, testconsts.Timeout, testconsts.Interval).Should(BeFalse(), "should not find resource.")
	})

	It("should bound PV and PVC objects and make sure deletion cleanups", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)

		By("checking if PVC exists")
		Eventually(func() bool {
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      desiredNfsPvc.Name,
					Namespace: desiredNfsPvc.Namespace,
				},
			}
			return utilst.DoesResourceExist(k8sClient, &pvc)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "should fetch pvc.")

		By("checking if PV exists")
		Eventually(func() bool {
			pv := corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
				},
			}
			return utilst.DoesResourceExist(k8sClient, &pv)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "should fetch pv.")

		By("checking if the PV and the PVC are in bound phase")
		Eventually(func() bool {
			nfspvc := utilst.GetNfsPvc(k8sClient, desiredNfsPvc.Name, desiredNfsPvc.Namespace)
			return nfspvc.Status.PvcPhase == string(corev1.ClaimBound) && nfspvc.Status.PvPhase == string(corev1.VolumeBound)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "PV and PVC Phases should be bound.")

		By("deleting the NFSPVC")
		utilst.DeleteNfsPvc(k8sClient, desiredNfsPvc)

		By("checking if the PV exists")
		Eventually(func() bool {
			pv := corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
				},
			}
			return utilst.DoesResourceExist(k8sClient, &pv)
		}, testconsts.Timeout, testconsts.Interval).Should(BeFalse(), "should not find a pv.")

		By("checking if the PVC exists")
		Eventually(func() bool {
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      desiredNfsPvc.Name,
					Namespace: desiredNfsPvc.Namespace,
				},
			}
			return utilst.DoesResourceExist(k8sClient, &pvc)
		}, testconsts.Timeout, testconsts.Interval).Should(BeFalse(), "should not find a pvc.")
	})

	It("should sync the PV and PVC to the desired state", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)

		By("deleting the associated PVC")
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      desiredNfsPvc.Name,
				Namespace: desiredNfsPvc.Namespace,
			},
		}
		previousPVCUid := utilst.GetResourceUid(k8sClient, &pvc)
		Expect(k8sClient.Delete(context.Background(), &pvc)).To(Succeed(), "failed to delete pvc.")

		By("checking if the PVC has been recreated")
		Eventually(func() string {
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      desiredNfsPvc.Name,
					Namespace: desiredNfsPvc.Namespace,
				},
			}
			return utilst.GetResourceUid(k8sClient, &pvc)
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(previousPVCUid), "should be a different pvc.")

		// If the recreated PVC is not bound to the original PV, the PV should be deleted and recreated
		By("Checking if the PV is in bound phase")
		pvName := desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv"
		pv := &corev1.PersistentVolume{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: pvName}, pv)).To(Succeed(), "should find pv.")

		// Pv is not bound, delete it
		if pv.Status.Phase != corev1.VolumeBound {
			By("deleting the associated PV")
			previousPVUid := pv.UID
			Expect(k8sClient.Delete(context.Background(), pv)).To(Succeed(), "failed to delete pv.")

			By("checking if the PV has been recreated")
			Eventually(func() string {
				pv := corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
					},
				}
				return utilst.GetResourceUid(k8sClient, &pv)
			}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(previousPVUid), "should be a different pv.")
		}

		By("deleting the NFSPVC")
		utilst.DeleteNfsPvc(k8sClient, desiredNfsPvc)
	})

	It("Should add the nfs version to the mountOption in the pv when nfs version is specified", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)
		desiredNfsPvc.Spec.NfsVersion = testconsts.NfsVersion

		By("checking if PV exists")
		pv := corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
			},
			Spec: corev1.PersistentVolumeSpec{
				MountOptions: []string{fmt.Sprintf("nfsvers=%s", testconsts.NfsVersion)},
			},
		}
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, &pv)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "should fetch pv.")

		By("Checking if the pv's mountOption is not empty")
		Eventually(func() string {
			mountOptionsString := strings.Join(pv.Spec.MountOptions, ",")
			return mountOptionsString
		}, testconsts.Timeout, testconsts.Interval).Should(ContainSubstring(desiredNfsPvc.Spec.NfsVersion), "should have mountOptions.")
	})

	It("Shouldn't add the nfs version to the mountOption in the pv if there is not version in the nfspvc", func() {
		baseNfsPvc := mock.CreateBaseNfsPvc()
		desiredNfsPvc := utilst.CreateNfsPvc(k8sClient, baseNfsPvc)

		By("checking if PV exists")
		pv := corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: desiredNfsPvc.Name + "-" + desiredNfsPvc.Namespace + "-pv",
			},
		}
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, &pv)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "should fetch pv.")

		By("Checking if the pv's mountOption is empty")
		Eventually(func() string {
			mountOptionsString := strings.Join(pv.Spec.MountOptions, ",")
			return mountOptionsString
		}, testconsts.Timeout, testconsts.Interval).Should(BeEmpty(), "should not have mountOptions.")
	})
})
