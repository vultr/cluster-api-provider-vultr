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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
)

// terminalMachineReasons are the Ready=False reasons after which the
// controller stops reconciling a VultrMachine.
var terminalMachineReasons = map[string]struct{}{
	infrav1.InstanceNotFoundReason:         {},
	infrav1.InstanceUnexpectedStatusReason: {},
}

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	VultrAPIClients
	Client       client.Client
	Logger       logr.Logger
	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VultrMachine *infrav1.VultrMachine
	VultrCluster *infrav1.VultrCluster
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VultrMachine *infrav1.VultrMachine
	VultrCluster *infrav1.VultrCluster
}

// NewMachineScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.VultrCluster == nil {
		return nil, errors.New("VultrCluster is required when creating a MachineScope")
	}
	if params.VultrMachine == nil {
		return nil, errors.New("VultrMachine is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.VultrMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &MachineScope{
		client:       params.Client,
		Logger:       params.Logger,
		Cluster:      params.Cluster,
		Machine:      params.Machine,
		VultrCluster: params.VultrCluster,
		VultrMachine: params.VultrMachine,
		patchHelper:  helper,
	}, nil
}

func (s *MachineScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.VultrMachine)
}

// PatchObject persists the machine spec and status.
func (m *MachineScope) PatchObject(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.VultrMachine)
}

// SetReady sets the VultrMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.VultrMachine.Status.Ready = true
}

// SetReadyCondition sets the Ready condition on the VultrMachine.
func (m *MachineScope) SetReadyCondition(status metav1.ConditionStatus, reason, message string) {
	conditions.Set(m.VultrMachine, metav1.Condition{
		Type:               infrav1.MachineReadyCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetTerminalFailure marks the infrastructure as failed with a terminal reason.
// The controller stops reconciling the machine afterwards.
func (m *MachineScope) SetTerminalFailure(reason, message string) {
	m.SetReadyCondition(metav1.ConditionFalse, reason, message)
	m.VultrMachine.Status.Ready = false
	m.VultrMachine.Status.Initialization.Provisioned = false
}

// HasTerminalFailure returns true when Ready is False with a terminal reason.
func (m *MachineScope) HasTerminalFailure() bool {
	c := conditions.Get(m.VultrMachine, infrav1.MachineReadyCondition)
	if c == nil || c.Status != metav1.ConditionFalse {
		return false
	}
	_, terminal := terminalMachineReasons[c.Reason]
	return terminal
}

// AddFinalizer adds a finalizer if not present and immediately patches the
// object to avoid any race conditions.
func (m *MachineScope) AddFinalizer(ctx context.Context) error {
	if controllerutil.AddFinalizer(m.VultrMachine, infrav1.GroupVersion.String()) {
		return m.Close()
	}

	return nil
}

// GetInstanceID returns the VultrMachine instance id by parsing Spec.ProviderID.
func (m *MachineScope) GetInstanceID() string {
	return ProviderIDToResourceID(m.GetProviderID())
}

// GetProviderID returns the VultrMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.VultrMachine.Spec.ProviderID != nil {
		return *m.VultrMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the VultrMachine providerID in spec from instance id.
func (m *MachineScope) SetProviderID(instanceID string) {
	m.VultrMachine.Spec.ProviderID = ptr.To(ResourceIDToProviderID(instanceID))
}

// Name returns the VultrMachine name.
func (m *MachineScope) Name() string {
	return m.VultrMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.VultrMachine.Namespace
}

func (m *MachineScope) GetBootstrapData() (string, error) {
	return GetBootstrapData(context.TODO(), m.client, m.Logger, m.Machine, m.Namespace())
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	return MachineRole(m.Machine)
}

// GetInstanceStatus returns the VultrMachine instance status from the status.
func (m *MachineScope) GetInstanceStatus() *infrav1.SubscriptionStatus {
	return m.VultrMachine.Status.SubscriptionStatus
}

// SetInstanceStatus sets the VultrMachine Instance.
func (m *MachineScope) SetInstanceStatus(v infrav1.SubscriptionStatus) {
	m.VultrMachine.Status.SubscriptionStatus = &v
}

// GetInstanceStatus returns the VultrMachine instance status from the status.
func (m *MachineScope) GetInstancePowerStatus() *infrav1.PowerStatus {
	return m.VultrMachine.Status.PowerStatus
}

// SetInstanceStatus sets the VultrMachine Instance.
func (m *MachineScope) SetInstancePowerStatus(v infrav1.PowerStatus) {
	m.VultrMachine.Status.PowerStatus = &v
}

// GetInstanceStatus returns the VultrMachine instance server state status .
func (m *MachineScope) GetInstanceServerState() *infrav1.ServerState {
	return m.VultrMachine.Status.ServerState
}

// GetInstanceStatus returns the VultrMachine instance server state status .
func (m *MachineScope) SetInstanceServerState(v infrav1.ServerState) {
	m.VultrMachine.Status.ServerState = &v
}

// SetAddresses sets the address status.
func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.VultrMachine.Status.Addresses = addrs
}

// SetCPU sets the VultrMachine CPU status.
func (m *MachineScope) SetCPU(v int) {
	m.VultrMachine.Status.CPU = v
}

// SetRAM sets the VultrMachine RAM status.
func (m *MachineScope) SetRAM(v int) {
	m.VultrMachine.Status.RAM = v
}

// SetStorage sets the VultrMachine Storage status.
func (m *MachineScope) SetStorage(v int) {
	m.VultrMachine.Status.Storage = v
}
