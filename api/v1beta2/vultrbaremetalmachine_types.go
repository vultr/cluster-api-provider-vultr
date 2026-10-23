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
	// BareMetalMachineFinalizer allows the VultrBareMetalMachine controller to clean up
	// Vultr resources associated with a VultrBareMetalMachine before removing it from the apiserver.
	BareMetalMachineFinalizer = "vultrbaremetalmachine.infrastructure.cluster.x-k8s.io"
)

// BareMetalDiskMode is the RAID configuration applied when a bare metal server is provisioned.
// +kubebuilder:validation:Enum=raid1;jbod;none
type BareMetalDiskMode string

const (
	// BareMetalDiskModeRAID1 mirrors the disks.
	BareMetalDiskModeRAID1 BareMetalDiskMode = "raid1"
	// BareMetalDiskModeJBOD exposes the disks individually.
	BareMetalDiskModeJBOD BareMetalDiskMode = "jbod"
	// BareMetalDiskModeNone applies no RAID configuration.
	BareMetalDiskModeNone BareMetalDiskMode = "none"
)

// VultrBareMetalMachineSpec defines the desired state of VultrBareMetalMachine.
type VultrBareMetalMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider (vultr://<server-id>).
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// Region is the Vultr region the server is deployed in.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region"`

	// PlanID is the Vultr bare metal plan id. Availability is per region;
	// see GET /v2/plans-metal.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	PlanID string `json:"planID"`

	// OSID is the Vultr operating system id to install (e.g. 2284 for Ubuntu 24.04).
	// Kubernetes components are then installed by the bootstrap data at first boot.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	OSID int `json:"osID"`

	// SSHKey is the list of Vultr SSH key ids to install on the server.
	// +optional
	SSHKey []string `json:"sshKey,omitempty"`

	// VPCID is the id of the VPC to attach once the server is active. The plan must
	// support VPC networking.
	// +optional
	VPCID string `json:"vpcID,omitempty"`

	// MdiskMode is the RAID configuration to use. When unset the Vultr default for the plan applies.
	// +optional
	MdiskMode BareMetalDiskMode `json:"mdiskMode,omitempty"`

	// EnableIPv6 enables IPv6 on the server. Defaults to true.
	// +optional
	EnableIPv6 *bool `json:"enableIPv6,omitempty"`

	// AdditionalTags are added to the server on top of the tags the provider manages.
	// +optional
	AdditionalTags []string `json:"additionalTags,omitempty"`
}

// VultrBareMetalVPCStatus describes the VPC attachment of a bare metal server.
type VultrBareMetalVPCStatus struct {
	// ID is the VPC id.
	// +optional
	ID string `json:"id,omitempty"`

	// IPAddress is the address of the server inside the VPC.
	// +optional
	IPAddress string `json:"ipAddress,omitempty"`

	// MACAddress is the MAC address of the VPC interface.
	// +optional
	MACAddress string `json:"macAddress,omitempty"`
}

// BareMetalMachineInitializationStatus provides observations of the VultrBareMetalMachine initialization process.
// +kubebuilder:validation:MinProperties=1
type BareMetalMachineInitializationStatus struct {
	// Provisioned is true when the infrastructure provider reports that the machine's
	// infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// VultrBareMetalMachineStatus defines the observed state of VultrBareMetalMachine.
type VultrBareMetalMachineStatus struct {
	// Ready indicates the infrastructure is ready to be used.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Initialization provides observations of the machine initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used
	// to orchestrate initial machine provisioning.
	// The value of these fields is never updated after provisioning is completed.
	// +optional
	Initialization BareMetalMachineInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Addresses contains the associated node addresses.
	// +optional
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// CPU is the number of CPUs of the server.
	// +optional
	CPU int `json:"cpu,omitempty"`

	// RAM is the amount of memory as reported by Vultr (e.g. "32768 MB").
	// +optional
	RAM string `json:"ram,omitempty"`

	// Disk is the disk configuration as reported by Vultr (e.g. "2x 240GB SSD").
	// +optional
	Disk string `json:"disk,omitempty"`

	// SubscriptionStatus represents the status of the Vultr subscription.
	// +optional
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// VPC describes the VPC attachment, if any.
	// +optional
	VPC *VultrBareMetalVPCStatus `json:"vpc,omitempty"`

	// Conditions represent the observations of the machine's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GetConditions returns the list of conditions for a VultrBareMetalMachine.
func (r *VultrBareMetalMachine) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets the conditions on a VultrBareMetalMachine.
func (r *VultrBareMetalMachine) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=vultrbaremetalmachines,scope=Namespaced,categories=cluster-api,shortName=vbm
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this VultrBareMetalMachine belongs"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.subscriptionStatus",description="Vultr bare metal server state"
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Machine ready status"
//+kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.planID",description="Vultr bare metal plan"
//+kubebuilder:printcolumn:name="ServerID",type="string",JSONPath=".spec.providerID",description="Vultr bare metal server ID"
//+kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns this VultrBareMetalMachine"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VultrBareMetalMachine is the Schema for the vultrbaremetalmachines API.
type VultrBareMetalMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VultrBareMetalMachineSpec   `json:"spec,omitempty"`
	Status VultrBareMetalMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VultrBareMetalMachineList contains a list of VultrBareMetalMachine.
type VultrBareMetalMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VultrBareMetalMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &VultrBareMetalMachine{}, &VultrBareMetalMachineList{})
}
