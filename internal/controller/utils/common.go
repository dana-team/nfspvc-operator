package utils

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"golang.org/x/exp/slices"
)

const (
	StorageClassEnv  = "STORAGE_CLASS"
	ReclaimPolicyEnv = "RECLAIM_POLICY"

	UndefinedEnvironmentVariableMsg = "failed to get configuration environment variable"
	InvalidReclaimPolicyMsg         = "invalid default Persistent Volume Reclaim Policy"

	NfsPvcDeletionFinalizer = "nfspvc.dana.io/nfspvc-protection"
)

var AllowedReclaimPolicies = []corev1.PersistentVolumeReclaimPolicy{
	corev1.PersistentVolumeReclaimRecycle,
	corev1.PersistentVolumeReclaimDelete,
	corev1.PersistentVolumeReclaimRetain,
}
var ReclaimPolicy string
var StorageClass string

// VerifyEnvironmentVariables ensures the StorageClass and ReclaimPolicy env variables are set and valid.
func VerifyEnvironmentVariables() (bool, string) {
	storageClass, ok := os.LookupEnv(StorageClassEnv)
	if !ok {
		return false, UndefinedEnvironmentVariableMsg
	}
	StorageClass = storageClass
	reclaimPolicy, ok := os.LookupEnv(ReclaimPolicyEnv)
	if !ok {
		return false, UndefinedEnvironmentVariableMsg
	}
	if !isReclaimPolicyValid(reclaimPolicy) {
		return false, InvalidReclaimPolicyMsg
	}
	ReclaimPolicy = reclaimPolicy

	return true, ""
}

// isReclaimPolicyValid checks if given reclaimPolicy is one of the AllowedReclaimPolicies.
func isReclaimPolicyValid(reclaimPolicy string) bool {
	policy := corev1.PersistentVolumeReclaimPolicy(reclaimPolicy)
	return slices.Contains(AllowedReclaimPolicies, policy)
}

// RetryOnConflictUpdate attempts to perform the given operation and retries if a conflict has occurred.
func RetryOnConflictUpdate[T client.Object](ctx context.Context, k8sClient client.Client, obj T, name, namespace string, updateOp func(T) error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj); err != nil {
			return err
		}
		return updateOp(obj)
	})
}
