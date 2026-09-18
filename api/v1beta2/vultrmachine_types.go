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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MachineFinalizer allows ReconcileVultrMachine to clean up Vultr resources
	// associated with VultrMachine before removing it from the apiserver.
	MachineFinalizer = "vultrmachine.infrastructure.cluster.x-k8s.io"
)

// VultrMachineSpec defines the desired state of VultrMachine
type VultrMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	//The Vultr snapshot_id to use when deploying this instance.
	Snapshot string `json:"snapshot_id,omitempty"`

	// PlanID is the id of Vultr VPS plan (VPSPLANID).
	PlanID string `json:"planID,omitempty"`

	// The Vultr Region (DCID) the cluster lives on
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// sshKey is the name of the ssh key to attach to the instance.
	// +optional
	SSHKey []string `json:"sshKey,omitempty"`

	// VPCID is the id of the VPC to be attached.
	// +optional
	VPCID string `json:"vpc_id,omitempty"`

	//VPCOnly indicates that the VPS will not receive a public IP or public NIC when true.
	// +optional
	VPCOnly bool `json:"vpc_only,omitempty"`

	//The Vultr firewall group ID to attach to the instance
	// +optional
	FirewallGroupID string `json:"firewall_group_id,omitempty"`

	// VPC2ID is the id of the VPC2.0 to be attached.
	// Deprecated: VPC2 is no longer supported and functionality will cease in a
	// future release
	// +optional
	VPC2ID string `json:"vpc2_id,omitempty"`
}

// VultrMachineStatus defines the observed state of VultrMachine.
type VultrMachineStatus struct {
	// Ready indicates the infrastructure is ready to be used.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// CPU represents the number of virtual CPUs of the instance.
	// +optional
	CPU int `json:"cpu,omitempty"`

	// RAM represents the amount of memory in MB of the instance.
	// +optional
	RAM int `json:"ram,omitempty"`

	// Storage represents the disk size in GB of the instance.
	// +optional
	Storage int `json:"storage,omitempty"`

	// Initialization provides observations of the machine initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used
	// to orchestrate initial machine provisioning.
	// The value of these fields is never updated after provisioning is completed.
	// +optional
	Initialization MachineInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Addresses contains the associated node addresses.
	// +optional
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// SubscriptionStatus represents the status of the Vultr subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// PowerStatus represents whether the VPS is powered on or not.
	// +optional
	PowerStatus *PowerStatus `json:"powerStatus,omitempty"`

	// ServerState provides details of the server state.
	// +optional
	ServerState *ServerState `json:"serverState,omitempty"`

	// Conditions represent the observations of the machine's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MachineInitializationStatus provides observations of the Machine initialization process.
// +kubebuilder:validation:MinProperties=1
type MachineInitializationStatus struct {
	// InfrastructureProvisioned is true when the infrastructure provider reports that
	// the machine's infrastructure is fully provisioned.
	// +optional
	Provisioned bool `json:"provisioned,omitempty"`
}

// GetConditions returns the list of conditions for a VultrMachine.
func (r *VultrMachine) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions on a VultrMachine.
func (r *VultrMachine) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrmachines,scope=Namespaced,categories=cluster-api
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this VultrMachine belongs"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.subscriptionStatus",description="Vultr instance state"
//+kubebuilder:printcolumn:name="Provisioned",type="string",JSONPath=".status.ready",description="Machine ready status"
//+kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="Vultr instance ID"
//+kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns this VultrMachine"

// VultrMachine is the Schema for the vultrmachines API.
type VultrMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrMachineSpec   `json:"spec,omitempty"`
	Status VultrMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VultrMachineList contains a list of VultrMachine.
type VultrMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VultrMachine{}, &VultrMachineList{})
}
