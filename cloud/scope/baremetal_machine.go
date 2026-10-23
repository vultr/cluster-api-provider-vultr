/*

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

package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
)

// terminalBareMetalReasons are the Ready=False reasons after which the
// controller stops reconciling a VultrBareMetalMachine.
var terminalBareMetalReasons = map[string]struct{}{
	infrav1.BareMetalUnexpectedStatusReason: {},
	infrav1.BareMetalVPCAttachFailedReason:  {},
}

// BareMetalMachineScopeParams defines the input parameters used to create a new BareMetalMachineScope.
type BareMetalMachineScopeParams struct {
	Client                client.Client
	Logger                logr.Logger
	Machine               *clusterv1.Machine
	Cluster               *clusterv1.Cluster
	VultrBareMetalMachine *infrav1.VultrBareMetalMachine
	VultrCluster          *infrav1.VultrCluster
}

// BareMetalMachineScope defines a scope around a VultrBareMetalMachine and its cluster.
type BareMetalMachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Machine               *clusterv1.Machine
	Cluster               *clusterv1.Cluster
	VultrBareMetalMachine *infrav1.VultrBareMetalMachine
	VultrCluster          *infrav1.VultrCluster
}

// NewBareMetalMachineScope creates a new scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewBareMetalMachineScope(params BareMetalMachineScopeParams) (*BareMetalMachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a BareMetalMachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a BareMetalMachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a BareMetalMachineScope")
	}
	if params.VultrCluster == nil {
		return nil, errors.New("VultrCluster is required when creating a BareMetalMachineScope")
	}
	if params.VultrBareMetalMachine == nil {
		return nil, errors.New("VultrBareMetalMachine is required when creating a BareMetalMachineScope")
	}

	helper, err := patch.NewHelper(params.VultrBareMetalMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &BareMetalMachineScope{
		client:                params.Client,
		Logger:                params.Logger,
		Cluster:               params.Cluster,
		Machine:               params.Machine,
		VultrCluster:          params.VultrCluster,
		VultrBareMetalMachine: params.VultrBareMetalMachine,
		patchHelper:           helper,
	}, nil
}

// Close persists the VultrBareMetalMachine spec and status.
func (m *BareMetalMachineScope) Close(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.VultrBareMetalMachine)
}

// Name returns the VultrBareMetalMachine name.
func (m *BareMetalMachineScope) Name() string {
	return m.VultrBareMetalMachine.Name
}

// Namespace returns the VultrBareMetalMachine namespace.
func (m *BareMetalMachineScope) Namespace() string {
	return m.VultrBareMetalMachine.Namespace
}

// GetProviderID returns the providerID from the spec, or an empty string.
func (m *BareMetalMachineScope) GetProviderID() string {
	if m.VultrBareMetalMachine.Spec.ProviderID != nil {
		return *m.VultrBareMetalMachine.Spec.ProviderID
	}
	return ""
}

// GetServerID returns the Vultr bare metal server id parsed from the providerID.
func (m *BareMetalMachineScope) GetServerID() string {
	return ProviderIDToResourceID(m.GetProviderID())
}

// SetProviderID sets the providerID in the spec from the server id.
func (m *BareMetalMachineScope) SetProviderID(serverID string) {
	m.VultrBareMetalMachine.Spec.ProviderID = ptr.To(ResourceIDToProviderID(serverID))
}

// GetBootstrapData returns the bootstrap data of the owning Machine.
func (m *BareMetalMachineScope) GetBootstrapData(ctx context.Context) (string, error) {
	return GetBootstrapData(ctx, m.client, m.Logger, m.Machine, m.Namespace())
}

// IsControlPlane returns true if the machine is a control plane machine.
func (m *BareMetalMachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the tag role value of the machine.
func (m *BareMetalMachineScope) Role() string {
	return MachineRole(m.Machine)
}

// SetReady marks the infrastructure as ready.
func (m *BareMetalMachineScope) SetReady() {
	m.VultrBareMetalMachine.Status.Ready = true
}

// SetProvisioned marks the infrastructure as provisioned.
func (m *BareMetalMachineScope) SetProvisioned() {
	m.VultrBareMetalMachine.Status.Initialization.Provisioned = ptr.To(true)
}

// SetAddresses sets the node addresses in the status.
func (m *BareMetalMachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.VultrBareMetalMachine.Status.Addresses = addrs
}

// SetServerStatus sets the Vultr subscription status.
func (m *BareMetalMachineScope) SetServerStatus(v infrav1.SubscriptionStatus) {
	m.VultrBareMetalMachine.Status.SubscriptionStatus = &v
}

// SetHardware records the server hardware as reported by Vultr.
func (m *BareMetalMachineScope) SetHardware(cpu int, ram, disk string) {
	m.VultrBareMetalMachine.Status.CPU = cpu
	m.VultrBareMetalMachine.Status.RAM = ram
	m.VultrBareMetalMachine.Status.Disk = disk
}

// SetVPC records the VPC attachment in the status.
func (m *BareMetalMachineScope) SetVPC(vpc *infrav1.VultrBareMetalVPCStatus) {
	m.VultrBareMetalMachine.Status.VPC = vpc
}

// SetReadyCondition sets the Ready condition, the single condition Cluster API
// mirrors into the owning Machine.
func (m *BareMetalMachineScope) SetReadyCondition(status metav1.ConditionStatus, reason, message string) {
	conditions.Set(m.VultrBareMetalMachine, metav1.Condition{
		Type:               infrav1.BareMetalMachineReadyCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetTerminalFailure marks the infrastructure as failed with a terminal reason.
// The controller stops reconciling the machine afterwards.
func (m *BareMetalMachineScope) SetTerminalFailure(reason, message string) {
	m.SetReadyCondition(metav1.ConditionFalse, reason, message)
	m.VultrBareMetalMachine.Status.Ready = false
}

// HasTerminalFailure returns true when Ready is False with a terminal reason.
func (m *BareMetalMachineScope) HasTerminalFailure() bool {
	c := conditions.Get(m.VultrBareMetalMachine, infrav1.BareMetalMachineReadyCondition)
	if c == nil || c.Status != metav1.ConditionFalse {
		return false
	}
	_, terminal := terminalBareMetalReasons[c.Reason]
	return terminal
}
