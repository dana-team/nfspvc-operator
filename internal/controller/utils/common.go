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

func IsEnvironmentVariablesExist() bool {
	storageClass := os.Getenv("STORAGE_CLASS")
	reclaimPolicy := os.Getenv("RECLAIM_POLICY")
	if storageClass == "" || reclaimPolicy == "" {
		return false
	}
	return true
}

func IsReclaimPolicyValid(reclaimPolicy string) bool {
	policy := corev1.PersistentVolumeReclaimPolicy(reclaimPolicy)
	return slices.Contains(AllowedReclaimPolicies, policy)
}
