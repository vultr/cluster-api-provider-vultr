/*
Copyright 2025.

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

package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// ClusterFinalizer allows ReconcileVultrCluster to clean up Vultr resources associated with VultrCluster
	// before removing it from the apiserver.
	ClusterFinalizer = "vultrcluster.infrastructure.cluster.x-k8s.io"
)

// VultrClusterSpec defines the desired state of VultrCluster.
type VultrClusterSpec struct {
	// The Vultr Region of the cluster
	Region string `json:"region"`

	// NetworkSpec encapsulates all things related to Vultr network.
	// +optional
	Network NetworkSpec `json:"network,omitempty"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// VPCID is the Vultr VPC ID used for the cluster's load balancer.
	// +optional
	VPCID string `json:"vpc_id,omitempty"`
}

// VultrClusterStatus defines the observed state of VultrCluster
type VultrClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready
	// +optional
	Ready bool `json:"ready"`

	// Initialization provides observations of the Cluster initialization process.
	// +optional
	Initialization ClusterInitializationStatus `json:"initialization,omitempty"`

	// Represents the observations of a Cluster's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Network encapsulates all things related to the Vultr network.
	// +optional
	Network VultrNetworkResource `json:"network"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrclusters,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this VultrCluster belongs"
//+kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].status",description="Cluster infrastructure availability"
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint",description="API Endpoint",priority=1

// VultrCluster is the Schema for the vultrclusters API.
type VultrCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrClusterSpec   `json:"spec,omitempty"`
	Status VultrClusterStatus `json:"status,omitempty"`
}

// ClusterInitializationStatus holds provisioning signals consumed by CAPI.
type ClusterInitializationStatus struct {
	// Provisioned is true when the infrastructure provider reports that the cluster infrastructure is fully provisioned.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// GetConditions returns the list of conditions for a VultrCluster.
func (r *VultrCluster) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions on a VultrCluster.
func (r *VultrCluster) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// VultrClusterList contains a list of VultrCluster.
type VultrClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrCluster `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &VultrCluster{}, &VultrClusterList{})
}
