package utils

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	. "github.com/onsi/gomega"

	nfspvcv1alpha1 "github.com/dana-team/nfspvc-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
const RandStrLength = 10

// generateRandomString returns a random string of the specified length using characters from the charset.
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)

	for i := range b {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[randomIndex.Int64()]
	}

	return string(b), nil
}

// CreateNfsPvc creates a new NfsPvc instance with a unique name and returns it.
func CreateNfsPvc(k8sClient client.Client, nfsPvc *nfspvcv1alpha1.NfsPvc) *nfspvcv1alpha1.NfsPvc {
	randString, err := generateRandomString(RandStrLength)
	Expect(err).To(BeNil())
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
