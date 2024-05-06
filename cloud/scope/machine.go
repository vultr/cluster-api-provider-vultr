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
	"golang.org/x/oauth2"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.VultrCluster == nil {
		return nil, errors.New("vultr cluster is required when creating a MachineScope")
	}
	if params.VultrMachine == nil {
		return nil, errors.New("vultr machine is required when creating a MachineScope")
	}

	config := &oauth2.Config{}
	tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

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

// SetFailureReason sets the VultrMachine status error reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.VultrMachine.Status.FailureReason = &v
}

// SetFailureMessage sets the VultrMachine status error message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.VultrMachine.Status.FailureMessage = ptr.To[string](v.Error())
}
