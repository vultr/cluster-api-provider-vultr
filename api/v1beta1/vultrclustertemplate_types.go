/*
Copyright 2024.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VultrClusterTemplateSpec defines the desired state of VultrClusterTemplate
type VultrClusterTemplateSpec struct {
	Template VultrClusterTemplateResource `json:"template"`
}

// VultrClusterTemplateResource contains spec for VultrClusterSpec.
type VultrClusterTemplateResource struct {
	Spec VultrClusterSpec `json:"spec"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=vultrclustertemplates,scope=Namespaced,categories=cluster-api,shortName=vct
//+kubebuilder:subresource:status

// VultrClusterTemplate is the Schema for the vultrclustertemplates API
type VultrClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VultrClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// VultrClusterTemplateList contains a list of VultrClusterTemplate
type VultrClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrClusterTemplate{}, &VultrClusterTemplateList{})
}
