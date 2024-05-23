package utils

import (
	"os"

	"golang.org/x/exp/slices"

	corev1 "k8s.io/api/core/v1"
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
