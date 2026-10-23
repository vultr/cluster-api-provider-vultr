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
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"
	"github.com/vultr/cluster-api-provider-vultr/util"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
)

// GetInstance looks up an instance by ID. It returns nil, nil when the ID is
// empty or Vultr answers 404; any other response, including a 200 with no
// instance, is an error.
func (s *Service) GetInstance(instanceID string) (*govultr.Instance, error) {
	if instanceID == "" {
		s.scope.Info("VultrInstance does not have an instance id")
		return nil, nil
	}

	s.scope.Logger.V(2).Info("Looking for instance by ID", "instance-id", instanceID)

	ctx, status := vultrerrors.Capture(s.ctx)
	instance, resp, err := s.scope.Instances.Get(ctx, instanceID)
	if err != nil {
		if status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance with ID %q", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("Vultr API returned HTTP %d without an instance for ID %q", responseCode(resp), instanceID)
	}

	return instance, nil
}

// FindInstanceForMachine finds the machine's instance by label and tags.
// Returns nil, nil when there is none and an error when there is more than one.
func (s *Service) FindInstanceForMachine(name, role string) (*govultr.Instance, error) {
	params := s.scope.MachineTagParams(name, role)

	var matches []govultr.Instance
	options := &govultr.ListOptions{Label: name}
	for {
		instances, meta, _, err := s.scope.Instances.List(s.ctx, options)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list instances labelled %q", name)
		}
		for i := range instances {
			if instanceOwnedByMachine(&instances[i], params) {
				matches = append(matches, instances[i])
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
		ids := make([]string, 0, len(matches))
		for i := range matches {
			ids = append(ids, matches[i].ID)
		}
		return nil, errors.Errorf("found %d instances for machine %q, expected at most one: %s", len(matches), name, strings.Join(ids, ", "))
	}
}

func instanceOwnedByMachine(instance *govultr.Instance, params infrav1.BuildTagParams) bool {
	if instance.Label != params.Name {
		return false
	}
	if infrav1.TagsOwnedByMachine(instance.Tags, params) {
		return true
	}
	// Tag step of the create failed; match on label and hostname.
	return len(instance.Tags) == 0 && instance.Hostname == params.Name
}

func responseCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func (s *Service) CreateInstance(scope *scope.MachineScope) (*govultr.Instance, error) {
	s.scope.V(2).Info("Creating an instance for a machine")

	s.scope.V(2).Info("Retrieving bootstrap data")
	bootstrapData, err := scope.GetBootstrapData()
	if err != nil {
		log.Error(err, "Error getting bootstrap data for machine")
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}
	s.scope.V(2).Info("Successfully retrieved bootstrap data")

	commands := []string{
		"ufw disable",
	}
	updatedBootstrapData := appendToUserDataCloudConfig(bootstrapData, commands)
	encodedBootstrapData := base64.StdEncoding.EncodeToString([]byte(updatedBootstrapData))

	var sshKeyIDs []string //nolint:prealloc
	for _, sshKeyID := range scope.VultrMachine.Spec.SSHKey {
		keys, err := s.GetSSHKey(sshKeyID)
		if err != nil {
			return nil, err
		}
		sshKeyIDs = append(sshKeyIDs, keys.ID)
	}
	instanceName := scope.Name()

	s.scope.V(2).Info("Preparing instance creation request payload")
	instanceReq := &govultr.InstanceCreateReq{
		Label:      instanceName,
		Hostname:   instanceName,
		Region:     scope.VultrMachine.Spec.Region,
		Plan:       scope.VultrMachine.Spec.PlanID,
		SSHKeys:    sshKeyIDs,
		SnapshotID: scope.VultrMachine.Spec.Snapshot,
		UserData:   encodedBootstrapData,
		EnableIPv6: util.Pointer(true),
	}

	if scope.VultrMachine.Spec.VPCID != "" {
		instanceReq.AttachVPC = append(instanceReq.AttachVPC, scope.VultrMachine.Spec.VPCID)
	}

	s.scope.V(2).Info("Building instance tags")
	instanceReq.Tags = infrav1.BuildTags(s.scope.MachineTagParams(instanceName, scope.Role()))
	s.scope.V(2).Info("Successfully built instance tags")

	// A failed create may still have made a VM; FindInstanceForMachine
	// picks it up on the next reconcile.
	s.scope.V(2).Info("Creating instance with Vultr API")
	instance, resp, err := s.scope.Instances.Create(s.ctx, instanceReq)
	if err != nil {
		log.Error(err, "Failed to create new instance")
		return nil, errors.Wrap(err, "Failed to create new instance")
	}
	if instance == nil || instance.ID == "" {
		return nil, errors.Errorf("Vultr API returned HTTP %d without an instance for the create of %q", responseCode(resp), instanceName)
	}
	s.scope.V(2).Info("Successfully created instance", "instance-id", instance.ID)

	return instance, nil
}

// DeleteInstance deletes the instance. It returns false, nil when Vultr says
// the instance does not exist.
func (s *Service) DeleteInstance(id string) (bool, error) {
	if id == "" {
		return false, errors.New("cannot delete instance. instance does not have an instance id")
	}

	s.scope.V(2).Info("Deleting instance", "instance-id", id)
	ctx, status := vultrerrors.Capture(s.ctx)
	if err := s.scope.Instances.Delete(ctx, id); err != nil {
		if status() == http.StatusNotFound {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to delete instance with id %q", id)
	}
	return true, nil
}

// GetInstanceAddress converts Vultr instance IPs to corev1.NodeAddresses.
func (s *Service) GetInstanceAddress(instance *govultr.Instance) ([]corev1.NodeAddress, error) {
	addresses := []corev1.NodeAddress{}

	// Add private IPv4 address
	if instance.InternalIP != "" {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: instance.InternalIP,
		})
	} else {
		s.scope.Info("No internal IPv4 address found for the instance", "instance-id", instance.ID)
	}

	// Add public IPv4 address
	if instance.MainIP != "" {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeExternalIP,
			Address: instance.MainIP,
		})
	} else {
		s.scope.Info("No external IPv4 address found for the instance", "instance-id", instance.ID)
	}

	return addresses, nil
}

func appendToUserDataCloudConfig(userData string, commands []string) string {
	runcmdIndex := strings.Index(userData, "runcmd:")
	runcmdIndex += len("runcmd:")

	// Append each command under runcmd: section
	for _, cmd := range commands {
		userData = userData[:runcmdIndex] + "\n  - " + cmd + userData[runcmdIndex:]
		runcmdIndex += len("\n  - " + cmd)
	}

	return userData
}
