package utils

import (
	"os"

	"golang.org/x/exp/slices"

	corev1 "k8s.io/api/core/v1"
)

var AllowedReclaimPolicies = []corev1.PersistentVolumeReclaimPolicy{
	corev1.PersistentVolumeReclaimRecycle,
	corev1.PersistentVolumeReclaimDelete,
	corev1.PersistentVolumeReclaimRetain,
}

const (
	STORAGE_CLASS_ENV  = "STORAGE_CLASS"
	RECLAIM_POLICY_ENV = "RECLAIM_POLICY"

	UNDEFINED_ENVIRONMENT_VARIABLE_MSG = "failed to get configuration environment variable"
	INVALID_RECLAIM_POLICY_MSG         = "invalid default Persistent Volume Reclaim Policy"
)

func VerifyEnvironmentVariables() (bool, string) {
	if !doesEnvironmentVariableExist() {
		return false, UNDEFINED_ENVIRONMENT_VARIABLE_MSG
	}

	if !doesReclaimPolicyValid(os.Getenv(RECLAIM_POLICY_ENV)) {
		return false, INVALID_RECLAIM_POLICY_MSG
	}

	return true, ""
}

func doesEnvironmentVariableExist() bool {
	storageClass := os.Getenv(STORAGE_CLASS_ENV)
	reclaimPolicy := os.Getenv(RECLAIM_POLICY_ENV)
	if storageClass == "" || reclaimPolicy == "" {
		return false
	}
	return true
}

func doesReclaimPolicyValid(reclaimPolicy string) bool {
	policy := corev1.PersistentVolumeReclaimPolicy(reclaimPolicy)
	return slices.Contains(AllowedReclaimPolicies, policy)
}
