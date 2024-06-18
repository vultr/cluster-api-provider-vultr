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
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/govultr/v3"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	VultrAPIClients
	Client       client.Client
	Logger       logr.Logger
	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VultrMachine *infrav1.VultrMachine
	VultrCluster *infrav1.VultrCluster
	APIKey       string
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	VultrClient *govultr.Client
	patchHelper *patch.Helper

	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VultrMachine *infrav1.VultrMachine
	VultrCluster *infrav1.VultrCluster
}

// NewMachineScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration
func NewMachineScope(ctx context.Context, apiKey string, params MachineScopeParams) (*MachineScope, error) {
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

	// config := &oauth2.Config{}
	// tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	// vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

	vultrClient, err := CreateVultrClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vultr Client: %w", err)
	}

	helper, err := patch.NewHelper(params.VultrMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &MachineScope{
		Logger:       params.Logger,
		VultrClient:  vultrClient,
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

// // SetFailureReason sets the VultrMachine status error reason.
// func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
// 	m.VultrMachine.Status.FailureReason = &v
// }

// // SetFailureMessage sets the VultrMachine status error message.
// func (m *MachineScope) SetFailureMessage(v error) {
// 	m.VultrMachine.Status.FailureMessage = ptr.To[string](v.Error())
// }

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
	id := m.GetProviderID()

	if id == "" ||
		!regexp.MustCompile("^[^:]+://.*[^/]$").MatchString(id) {
		return ""
	}

	colonIndex := strings.Index(id, ":")
	cloudProvider := id[0:colonIndex]

	lastSlashIndex := strings.LastIndex(id, "/")
	instance := id[lastSlashIndex+1:]

	if cloudProvider == "" || instance == "" {
		return ""
	}

	return instance
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
	pid := fmt.Sprintf("vultr://%s", instanceID)
	m.VultrMachine.Spec.ProviderID = ptr.To[string](pid)
}

// Name returns the VultrMachine name.
func (m *MachineScope) Name() string {
	return m.VultrMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.VultrMachine.Namespace
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for VultrMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return string(value), nil
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return infrav1.APIServerRoleTagValue
	}
	return infrav1.NodeRoleTagValue
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
