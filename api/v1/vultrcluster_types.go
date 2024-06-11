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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// ClusterFinalizer allows ReconcileVultrCluster to clean up Vultr resources associated with VultrCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "vultrcluster.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VultrClusterSpec defines the desired state of VultrCluster
type VultrClusterSpec struct {
	// The Vultr Region (DCID) the cluster lives on
	Region string `json:"region"`

	// NetworkSpec encapsulates all things related to Vultr network.
	// +optional
	Network NetworkSpec `json:"network"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the
	// control plane. If ControlPlaneDNS is unset, the Vultr load-balancer IP
	// of the Kubernetes API Server is used.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// CredentialsRef is a reference to a Secret that contains the credentials to use for provisioning this cluster. If not
	// supplied then the credentials of the controller will be used.
	// +optional
	CredentialsRef *corev1.SecretReference `json:"credentialsRef,omitempty"`
}

// VultrClusterStatus defines the observed state of VultrCluster
type VultrClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready
	Ready bool `json:"ready"`
	//Network Network `json:"network,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.ClusterStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the VultrCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	//Network VultrNetworkResource `json:"network,omitempty"`
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

func (r *VultrCluster) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrCluster) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
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
