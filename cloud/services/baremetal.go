/*
Copyright 2020 The Kubernetes Authors.

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

package services

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"
	"github.com/vultr/cluster-api-provider-vultr/util"
)

// bareMetalListPageSize is the page size used when listing bare metal servers.
const bareMetalListPageSize = 100

// GetBareMetalServer retrieves a bare metal server by id. It returns nil, nil
// when the server does not exist.
func (s *Service) GetBareMetalServer(id string) (*govultr.BareMetalServer, error) {
	if id == "" {
		return nil, nil
	}

	s.scope.V(2).Info("Looking for bare metal server by ID", "server-id", id)
	ctx, status := vultrerrors.Capture(s.ctx)
	server, resp, err := s.scope.BareMetal.Get(ctx, id)
	if err != nil {
		if status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get bare metal server with ID %q", id)
	}
	if server == nil {
		return nil, errors.Errorf("Vultr API returned HTTP %d without a server for ID %q", responseCode(resp), id)
	}

	return server, nil
}

// FindBareMetalServerByTags looks up a bare metal server created for the named
// machine of this cluster by the tags the provider applies at creation. It is
// used to adopt a server whose id was never persisted in the providerID.
func (s *Service) FindBareMetalServerByTags(name, role string) (*govultr.BareMetalServer, error) {
	params := s.scope.MachineTagParams(name, role)

	var matches []govultr.BareMetalServer
	options := &govultr.ListOptions{PerPage: bareMetalListPageSize, Tag: infrav1.NameTagFromName(name)}
	for {
		servers, meta, _, err := s.scope.BareMetal.List(s.ctx, options)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list bare metal servers")
		}
		for i := range servers {
			if infrav1.TagsOwnedByMachine(servers[i].Tags, params) {
				matches = append(matches, servers[i])
			}
		}
		if meta == nil || meta.Links == nil || meta.Links.Next == "" {
			break
		}
		options.Cursor = meta.Links.Next
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return &matches[0], nil
	default:
		return nil, errors.Errorf("found %d bare metal servers tagged for machine %q, expected at most one", len(matches), name)
	}
}

func hasTags(tags []string, wanted ...string) bool {
	for _, w := range wanted {
		found := false
		for _, t := range tags {
			if t == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// CreateBareMetalServer creates a bare metal server for the machine.
func (s *Service) CreateBareMetalServer(ctx context.Context, machineScope *scope.BareMetalMachineScope) (*govultr.BareMetalServer, error) {
	s.scope.V(2).Info("Creating a bare metal server for a machine")

	bootstrapData, err := machineScope.GetBootstrapData(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}
	userData, err := PrependCloudConfigRunCmds(bootstrapData, []string{"ufw disable"})
	if err != nil {
		return nil, err
	}
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	sshKeyIDs := make([]string, 0, len(machineScope.VultrBareMetalMachine.Spec.SSHKey))
	for _, sshKeyID := range machineScope.VultrBareMetalMachine.Spec.SSHKey {
		if sshKeyID == "" {
			continue
		}
		key, err := s.GetSSHKey(sshKeyID)
		if err != nil {
			return nil, err
		}
		sshKeyIDs = append(sshKeyIDs, key.ID)
	}

	tags := infrav1.BuildTags(s.scope.MachineTagParams(machineScope.Name(), machineScope.Role()))

	req := buildBareMetalCreateRequest(machineScope.VultrBareMetalMachine.Spec, machineScope.Name(), sshKeyIDs, encodedUserData, tags)

	s.scope.V(2).Info("Creating bare metal server with Vultr API", "region", req.Region, "plan", req.Plan)
	server, resp, err := s.scope.BareMetal.Create(s.ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create bare metal server")
	}
	if server == nil || server.ID == "" {
		return nil, errors.Errorf("Vultr API returned HTTP %d without a server for the create of %q", responseCode(resp), machineScope.Name())
	}
	s.scope.V(2).Info("Successfully created bare metal server", "server-id", server.ID)

	return server, nil
}

// buildBareMetalCreateRequest translates the machine spec into a Vultr create request.
func buildBareMetalCreateRequest(spec infrav1.VultrBareMetalMachineSpec, name string, sshKeyIDs []string, encodedUserData string, tags []string) *govultr.BareMetalCreate {
	enableIPv6 := true
	if spec.EnableIPv6 != nil {
		enableIPv6 = *spec.EnableIPv6
	}

	allTags := make([]string, 0, len(tags)+len(spec.AdditionalTags))
	allTags = append(allTags, tags...)
	allTags = append(allTags, spec.AdditionalTags...)

	return &govultr.BareMetalCreate{
		Region:          spec.Region,
		Plan:            spec.PlanID,
		OsID:            spec.OSID,
		Label:           name,
		Hostname:        name,
		SSHKeyIDs:       sshKeyIDs,
		UserData:        encodedUserData,
		MdiskMode:       string(spec.MdiskMode),
		EnableIPv6:      util.Pointer(enableIPv6),
		ActivationEmail: util.Pointer(false),
		Tags:            allTags,
	}
}

// DeleteBareMetalServer deletes a bare metal server.
// DeleteBareMetalServer deletes the server. It returns false, nil when Vultr
// says the server does not exist.
func (s *Service) DeleteBareMetalServer(id string) (bool, error) {
	if id == "" {
		return false, errors.New("cannot delete bare metal server: missing server id")
	}

	s.scope.V(2).Info("Deleting bare metal server", "server-id", id)
	ctx, status := vultrerrors.Capture(s.ctx)
	if err := s.scope.BareMetal.Delete(ctx, id); err != nil {
		if status() == http.StatusNotFound {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to delete bare metal server with id %q", id)
	}
	return true, nil
}

// EnsureBareMetalVPC makes sure the VPC is attached to the server. It returns
// attached=true with the attachment details once the VPC shows up on the
// server; otherwise it requests the attachment and returns attached=false so
// the caller can verify on a later reconcile. The attachment is asynchronous
// on the Vultr side.
func (s *Service) EnsureBareMetalVPC(serverID, vpcID string) (bool, *govultr.VPCInfo, error) {
	infos, _, err := s.scope.BareMetal.ListVPCInfo(s.ctx, serverID)
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to list VPCs of bare metal server %q", serverID)
	}
	for i := range infos {
		if infos[i].ID == vpcID {
			return true, &infos[i], nil
		}
	}

	s.scope.V(2).Info("Attaching VPC to bare metal server", "server-id", serverID, "vpc-id", vpcID)
	if err := s.scope.BareMetal.AttachVPC(s.ctx, serverID, vpcID); err != nil {
		return false, nil, errors.Wrapf(err, "failed to attach VPC %q to bare metal server %q", vpcID, serverID)
	}

	return false, nil, nil
}

// IsTerminalVPCAttachError reports whether a VPC attach error will never
// succeed on retry, such as the plan not supporting VPC networking.
func IsTerminalVPCAttachError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not support vpc") || strings.Contains(msg, "not support vpc networking")
}

// GetBareMetalServerAddresses converts the server addresses to node addresses.
// The VPC address, when attached, is the node internal IP.
func GetBareMetalServerAddresses(server *govultr.BareMetalServer, vpc *govultr.VPCInfo) []corev1.NodeAddress {
	addresses := []corev1.NodeAddress{}

	if vpc != nil && vpc.IPAddress != "" {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: vpc.IPAddress,
		})
	}

	if server.MainIP != "" {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeExternalIP,
			Address: server.MainIP,
		})
	}

	return addresses
}
