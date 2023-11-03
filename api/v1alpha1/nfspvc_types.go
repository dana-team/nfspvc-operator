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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NfsPvcSpec defines the desired state of NfsPvc
type NfsPvcSpec struct {

	// AccessModes is the type of the access on the pvc RWX, RWo, ROX
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty" protobuf:"bytes,3,rep,name=accessModes,casttype=PersistentVolumeAccessMode"`
	// Capacity is for the size of the nfs
	Capacity corev1.ResourceList `json:"capacity,omitempty" protobuf:"bytes,1,rep,name=capacity,casttype=ResourceList,castkey=ResourceName"`
	// Path is the path of the nfs volume
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`
	// Server is where you store you nfs
	Server string `json:"server" protobuf:"bytes,1,opt,name=server"`
}

// NfsPvcStatus defines the observed state of NfsPvc
type NfsPvcStatus struct {
	// PvcStatus is the current status of the NfsPvc object
	PvcPhase string `json:"pvcPhase,omitempty" protobuf:"bytes,3,opt,name=pvcPhase"`
	PvPhase  string `json:"pvPhase,omitempty" protobuf:"bytes,3,opt,name=pvPhase"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NfsPvc is the Schema for the nfspvcs API
type NfsPvc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NfsPvcSpec   `json:"spec,omitempty"`
	Status NfsPvcStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NfsPvcList contains a list of NfsPvc
type NfsPvcList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NfsPvc `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NfsPvc{}, &NfsPvcList{})
}