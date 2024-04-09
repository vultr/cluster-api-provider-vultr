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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VultrClusterSpec defines the desired state of VultrCluster
type VultrClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of VultrCluster. Edit vultrcluster_types.go to remove/update
	// The Vultr Region (DCID) the cluster lives on
	Region string `json:"region"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the
	// control plane. If ControlPlaneDNS is unset, the Vultr load-balancer IP
	// of the Kubernetes API Server is used.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// VultrClusterStatus defines the observed state of VultrCluster
type VultrClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready denotes that the cluster (infrastructure) is ready
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VultrCluster is the Schema for the vultrclusters API
type VultrCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrClusterSpec   `json:"spec,omitempty"`
	Status VultrClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VultrClusterList contains a list of VultrCluster
type VultrClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrCluster{}, &VultrClusterList{})
}
