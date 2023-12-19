package e2e_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"
	mock "github.com/dana-team/nfspvc-operator/test/e2e_tests/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	k8sClient ctrl.Client
)

const (
	TimeoutNameSpace = time.Minute
	NsFetchInterval  = 5 * time.Second
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Second * 2)
	RunSpecs(t, "E2e Suite")
}

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = nfspvcv1alpha1.AddToScheme(s)

	_ = scheme.AddToScheme(s)
	return s
}

var _ = SynchronizedBeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	// Get the cluster configuration.
	// Get the k8sClient or die
	config, err := config.GetConfig()
	if err != nil {
		Fail(fmt.Sprintf("Couldn't get kubeconfig %v", err))
	}

	// Create the client using the controller-runtime
	k8sClient, err = ctrl.New(config, ctrl.Options{Scheme: newScheme()})
	Expect(err).NotTo(HaveOccurred())

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NsName,
		},
	}

	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())
}, func() {})

var _ = SynchronizedAfterSuite(func() {}, func() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NsName,
		},
	}
	Expect(k8sClient.Delete(context.Background(), namespace)).To(Succeed())
	Eventually(func() error {
		return k8sClient.Get(context.Background(), ctrl.ObjectKey{Name: mock.NsName}, namespace)
	}, TimeoutNameSpace, NsFetchInterval).Should(HaveOccurred(), "The namespace should be deleted")
})
