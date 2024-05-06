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
	// ProviderID is the unique identifer as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`

	//The ID of the operating system to be installed
	OSID int `json:"osID,omitempty"`

	// PlanID is the id of Vultr VPS plan (VPSPLANID).
	PlanID int `json:"planID,omitempty"`

	// SSHKeyName is the name of the ssh key to attach to the instance.
	SSHKeyName string `json:"sshKeyName,omitempty"`

	// ScriptID is the id of Startup Script (SCRIPTID).
	ScriptID int `json:"scriptID,omitempty"`
}

// VultrMachineStatus defines the observed state of VultrMachine
type VultrMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready represents the infrastructure is ready to be used or not.
	Ready bool `json:"ready"`

	// ServerStatus represents the status of subscription.
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`

	// PowerStatus represents whether the server is powered on or not.
	PowerStatus *PowerStatus `json:"powerStatus,omitempty"`

	// ServerState represents a more detailed server status.
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
