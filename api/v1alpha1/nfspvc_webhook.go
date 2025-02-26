/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"errors"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var c client.Client

const (
	pvcAlreadyExists       = "a PVC of this name already exists in the namespace. Please rename your NFSPVC"
	invalidAccessModeError = "forbidden: only the following AccessModes are permitted"
)

var supportedAccessModes = sets.New(
	corev1.ReadWriteOnce,
	corev1.ReadOnlyMany,
	corev1.ReadWriteMany,
	corev1.ReadWriteOncePod,
)

// log is for logging in this package.
var nfspvclog = logf.Log.WithName("nfspvc-resource")

func (r *NfsPvc) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-nfspvc-dana-io-v1alpha1-nfspvc,mutating=false,failurePolicy=fail,sideEffects=None,groups=nfspvc.dana.io,resources=nfspvcs,verbs=create;update,versions=v1alpha1,name=vnfspvc-v1alpha1.kb.io,admissionReviewVersions=v1

type NfsPvcValidator struct {
	NfsPvc
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NfsPvc) ValidateCreate() (admission.Warnings, error) {
	nfspvclog.Info("validate create", "name", r.Name)

	if r.doesPVCExist(c) {
		return admission.Warnings{pvcAlreadyExists}, errors.New(pvcAlreadyExists)
	}

	if !r.validateAccessMode(r.Spec.AccessModes) {
		return admission.Warnings{invalidAccessModeError}, fmt.Errorf(invalidAccessModeError+": %v", supportedAccessModes)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NfsPvc) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nfspvclog.Info("validate update", "name", r.Name)

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NfsPvc) ValidateDelete() (admission.Warnings, error) {
	nfspvclog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *NfsPvc) validateAccessMode(accessMode []corev1.PersistentVolumeAccessMode) bool {
	for _, mode := range accessMode {
		if !supportedAccessModes.Has(mode) {
			return false
		}
	}
	return true
}

func (r *NfsPvc) doesPVCExist(K8sClient client.Client) bool {
	pvc := corev1.PersistentVolumeClaim{}
	if err := K8sClient.Get(context.Background(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, &pvc); err != nil {
		if k8sErrors.IsNotFound(err) {
			return false
		}
	}
	return true
}
