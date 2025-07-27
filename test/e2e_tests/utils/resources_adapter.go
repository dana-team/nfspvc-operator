package utils

import (
	"context"
	"fmt"

	gingko "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateResource(k8sClient client.Client, obj client.Object) bool {
	copyObject := obj.DeepCopyObject().(client.Object)
	err := k8sClient.Create(context.Background(), copyObject)
	return err == nil
}

func UpdateResource(k8sClient client.Client, obj client.Object) error {
	err := k8sClient.Update(context.Background(), obj)
	return err
}

// DoesResourceExist checks if a given Kubernetes object exists in the cluster.
func DoesResourceExist(k8sClient client.Client, obj client.Object) bool {
	copyObject := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(copyObject)
	err := k8sClient.Get(context.Background(), key, copyObject)
	if errors.IsNotFound(err) {
		return false
	} else if err != nil {
		gingko.Fail(fmt.Sprintf("The function failed with error: \n %s", err.Error()))
	}
	return true
}

// GetResourceUid returns a given Kubernetes object UID.
func GetResourceUid(k8sClient client.Client, obj client.Object) string {
	copyObject := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(copyObject)
	err := k8sClient.Get(context.Background(), key, copyObject)
	if errors.IsNotFound(err) {
		return ""
	} else if err != nil {
		gingko.Fail(fmt.Sprintf("The function failed with error: \n %s", err.Error()))
	}
	return string(copyObject.GetUID())
}
