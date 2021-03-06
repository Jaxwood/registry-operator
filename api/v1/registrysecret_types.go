/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var SchemeGroupVersion = schema.GroupVersion{Group: "apps.jaxwood.com", Version: "v1"}

// RegistrySecretSpec defines the desired state of RegistrySecret
type RegistrySecretSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ImagePullSecretName is the name of the image pull secret to sync across namespaces
	ImagePullSecretName string `json:"imagePullSecretName"`
	ImagePullSecretKey  string `json:"imagePullSecretKey"`
}

// RegistrySecretStatus defines the observed state of RegistrySecret
type RegistrySecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RegistrySecret is the Schema for the registrysecrets API
type RegistrySecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySecretSpec   `json:"spec,omitempty"`
	Status RegistrySecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RegistrySecretList contains a list of RegistrySecret
type RegistrySecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistrySecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegistrySecret{}, &RegistrySecretList{})
}
