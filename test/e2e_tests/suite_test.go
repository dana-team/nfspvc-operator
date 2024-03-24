package e2e_tests

import (
	"context"
	"testing"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	mock "github.com/dana-team/nfspvc-operator/test/e2e_tests/mocks"
	"github.com/dana-team/nfspvc-operator/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/nfspvc-operator/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var k8sClient ctrl.Client

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(testconsts.DefaultEventually)
	RunSpecs(t, "NFSPVC Suite")
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = nfspvcv1alpha1.AddToScheme(s)

	_ = scheme.AddToScheme(s)
	return s
}

var _ = SynchronizedBeforeSuite(func() {
	initClient()
	cleanUp()
	createE2ETestNamespace()
}, func() {
	initClient()
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	cleanUp()
})

// initClient initializes a k8s client.
func initClient() {
	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = ctrl.New(cfg, ctrl.Options{Scheme: newScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
}

// createE2ETestNamespace creates a namespace for the e2e tests.
func createE2ETestNamespace() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NSName,
		},
	}

	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())
	Eventually(func() bool {
		return utilst.DoesResourceExist(k8sClient, namespace)
	}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "The namespace should be created")
}

// cleanUp makes sure the test environment is clean.
func cleanUp() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NSName,
		},
	}
	if utilst.DoesResourceExist(k8sClient, namespace) {
		Expect(k8sClient.Delete(context.Background(), namespace)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), ctrl.ObjectKey{Name: mock.NSName}, namespace)
		}, testconsts.Timeout, testconsts.Interval).Should(HaveOccurred(), "The namespace should be deleted")
	}
}
