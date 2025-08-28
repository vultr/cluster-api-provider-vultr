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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileVultrMachine to clean up Vultr resources associated with VultrMachine before
	// removing it from the apiserver.
	MachineFinalizer = "vultrmachine.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VultrMachineSpec defines the desired state of VultrMachine
type VultrMachineSpec struct {
	// Foo is an example field of VultrMachine. Edit vultrmachine_types.go to remove/update
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	//The Application image_id to use when deploying this instance.
	Snapshot string `json:"snapshot_id,omitempty"`

	// PlanID is the id of Vultr VPS plan (VPSPLANID).
	PlanID string `json:"planID,omitempty"`

	// The Vultr Region (DCID) the cluster lives on
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// sshKey is the name of the ssh key to attach to the instance.
	// +optional
	SSHKey []string `json:"sshKey,omitempty"`

	// VPCID is the id of the VPC to be attched .
	// +optional
	VPCID string `json:"vpc_id,omitempty"`

	// VPC2ID is the id of the VPC2.0 to be attched .
	// Deprecated: VPC2 is no longer supported and functionality will cease in a
	// future release
	// +optional
	VPC2ID string `json:"vpc2_id,omitempty"`
}

// VultrMachineStatus defines the observed state of VultrMachine
type VultrMachineStatus struct {

	// Ready represents the infrastructure is ready to be used or not.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the Vultr instance associated addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// ServerStatus represents the status of subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// PowerStatus represents that the VPS is powerd on or not
	// +optional
	PowerStatus *PowerStatus `json:"powerStatus,omitempty"`

	// ServerState represents a detail of server state.
	// +optional
	ServerState *ServerState `json:"serverState,omitempty"`

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
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

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
}

func (r *VultrMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *VultrMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrmachines,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this VultrMachine belongs"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.subscriptionStatus",description="Vultr instance state"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
//+kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="Vultr instance ID"
//+kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this VultrMachine"

// VultrMachine is the Schema for the vultrmachines API
type VultrMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrMachineSpec   `json:"spec,omitempty"`
	Status VultrMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VultrMachineList contains a list of VultrMachine
type VultrMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrMachine{}, &VultrMachineList{})
}
