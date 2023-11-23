package utils

import (
	"context"
	"math/rand"
	"time"

	. "github.com/onsi/gomega"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
const RandStrLength = 10

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// generateRandomString returns a random string of the specified length using characters from the charset.
func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CreateNfsPvc creates a new NfsPvc instance with a unique name and returns it.
func CreateNfspvc(k8sClient client.Client, nfsPvc *nfspvcv1alpha1.NfsPvc) *nfspvcv1alpha1.NfsPvc {
	randString := generateRandomString(RandStrLength)
	nfsPvcName := nfsPvc.Name + "-" + randString
	newNfsPvc := nfsPvc.DeepCopy()
	newNfsPvc.Name = nfsPvcName
	Expect(k8sClient.Create(context.Background(), newNfsPvc)).To(Succeed())
	return newNfsPvc

}

// DeleteNfsPvc deletes an existing NfsPvc instance.
func DeleteNfsPvc(k8sClient client.Client, nfsPvc *nfspvcv1alpha1.NfsPvc) {
	Expect(k8sClient.Delete(context.Background(), nfsPvc)).To(Succeed(), "failed to delete NfsPvc")

}

// GetNfsPvc fetch existing and return an instance of NfsPvc.
func GetNfsPvc(k8sClient client.Client, name string, namespace string) *nfspvcv1alpha1.NfsPvc {
	nfsPvc := &nfspvcv1alpha1.NfsPvc{}
	Eventually(func() error {
		return k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, nfsPvc)
	}, 16*time.Second, 2*time.Second).Should(Succeed(), "should fetch nfsPvc")
	return nfsPvc
}
