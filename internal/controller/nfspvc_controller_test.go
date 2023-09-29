package controller_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	danaiov1 "dana.io/nfs-operator/api/v1"
	"dana.io/nfs-operator/internal/controller"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	timeout       = time.Second * 20
	interval      = time.Second * 1
	defaultPath   = "/path/to/nfs"
	defaultServer = "nfs-server.example.com"
)

var (
	defaultAccessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	defaultCapacity   = corev1.ResourceList{
		corev1.ResourceStorage: resource.MustParse("1Gi"),
	}
)

var (
	reconciler *controller.NfsPvcReconciler
	request    reconcile.Request
	nfspvc     *danaiov1.NfsPvc
	ctx        context.Context
	logger     logr.Logger
)

var _ = Describe("NfspvcController", func() {

	np := &danaiov1.NfsPvc{}
	ctx = context.Background()
	Context("NfsPvc creation and deletion", func() {

		It("should create a corresponding PV and PVC when NfsPvc is created", func() {

			logger = log.FromContext(ctx)

			reconciler = &controller.NfsPvcReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
				Log:    logger,
			}

			nfspvc = &danaiov1.NfsPvc{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfspvc",
					Namespace: "default",
				},
				Spec: danaiov1.NfsPvcSpec{
					AccessModes: defaultAccessMode,
					Capacity:    defaultCapacity,
					Path:        defaultPath,
					Server:      defaultServer,
				},
			}

			// Create the NfsPvc CR
			Expect(k8sClient.Create(ctx, nfspvc)).Should(Succeed())

			request = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      nfspvc.Name,
					Namespace: nfspvc.Namespace,
				},
			}

			pv := &corev1.PersistentVolume{}

			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).ShouldNot(HaveOccurred())

			err = k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace}, pv)
			Expect(err).ShouldNot(HaveOccurred())

			pvc := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, pvc)
			Expect(err).ShouldNot(HaveOccurred())

		})

		It("should delete the pv and pvc when nfspvc is deleted", func() {
			// Step 1: Delete the NfsPvc CR
			err := k8sClient.Delete(ctx, nfspvc)
			Expect(err).ShouldNot(HaveOccurred())

			// Step 2: Run the reconciler
			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, request.NamespacedName, np)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			pv := &corev1.PersistentVolume{}

			// Step 3: Check if the corresponding PV has been deleted
			Eventually(func() bool {
				err = k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name + "-" + nfspvc.Namespace}, pv)
				logger.Info("debug pv deletion", "pv", pv)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue(), "PersistentVolume should be deleted when NfsPvc is deleted")

			pvc := &corev1.PersistentVolumeClaim{}

			// Step 4: Check if the corresponding PVC has been deleted
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: nfspvc.Name, Namespace: nfspvc.Namespace}, pvc)
			}, timeout, interval).Should(HaveOccurred(), "PersistentVolumeClaim should be deleted when NfsPvc is deleted")

		})

	})
	// ... More test cases ...
})
