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
	"time"

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/util"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
)

// GetInstance retrieves an instance by its ID.
func (s *Service) GetInstance(instanceID string) (*govultr.Instance, error) {
	if instanceID == "" {
		s.scope.Info("VultrInstance does not have an instance id")
		return nil, nil
	}

	s.scope.Logger.V(2).Info("Looking for instance by ID", "instance-id", instanceID)

	instance, resp, err := s.scope.Instances.Get(s.ctx, instanceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance with ID %q", instanceID)
	}

	return instance, nil
}

func (s *Service) CreateInstance(scope *scope.MachineScope) (*govultr.Instance, error) {
	s.scope.V(2).Info("Creating an instance for a machine")

	s.scope.V(2).Info("Retrieving bootstrap data")
	bootstrapData, err := scope.GetBootstrapData()

	commands := []string{
		"ufw disable",
	}
	updatedBootstrapData := appendToUserDataCloudConfig(bootstrapData, commands)
	encodedBootstrapData := base64.StdEncoding.EncodeToString([]byte(updatedBootstrapData))

	if err != nil {
		log.Error(err, "Error getting bootstrap data for machine")
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}
	s.scope.V(2).Info("Successfully retrieved bootstrap data")

	var sshKeyIDs []string //nolint:prealloc
	for _, sshKeyID := range scope.VultrMachine.Spec.SSHKey {
		keys, err := s.GetSSHKey(sshKeyID)
		if err != nil {
			return nil, err
		}
		sshKeyIDs = append(sshKeyIDs, keys.ID)
	}
	clusterName := s.scope.Name()
	instanceName := scope.Name()

	s.scope.V(2).Info("Preparing instance creation request payload")
	instanceReq := &govultr.InstanceCreateReq{
		Label:           instanceName,
		Hostname:        instanceName,
		Region:          scope.VultrMachine.Spec.Region,
		Plan:            scope.VultrMachine.Spec.PlanID,
		SSHKeys:         sshKeyIDs,
		SnapshotID:      scope.VultrMachine.Spec.Snapshot,
		UserData:        encodedBootstrapData,
		EnableIPv6:      util.Pointer(true),
		FirewallGroupID: scope.VultrMachine.Spec.FirewallGroupID,
	}

	if scope.VultrMachine.Spec.VPCID != "" {
		instanceReq.AttachVPC = append(instanceReq.AttachVPC, scope.VultrMachine.Spec.VPCID)
	} else if scope.VultrMachine.Spec.VPC2ID != "" {
		// Deprecated: VPC2 is no longer supported and functionality will cease in a
		// future release
		instanceReq.AttachVPC2 = append(instanceReq.AttachVPC2, scope.VultrMachine.Spec.VPCID) //nolint:staticcheck
	}

	s.scope.V(2).Info("Building instance tags")
	instanceReq.Tags = infrav1.BuildTags(infrav1.BuildTagParams{
		ClusterName: clusterName,
		ClusterUID:  s.scope.UID(),
		Name:        instanceName,
		Role:        scope.Role(),
	})
	s.scope.V(2).Info("Successfully built instance tags")

	s.scope.V(2).Info("Creating instance with Vultr API")
	instance, _, err := s.scope.Instances.Create(s.ctx, instanceReq)
	if err != nil {
		log.Error(err, "Failed to create new instance")
		return nil, errors.Wrap(err, "Failed to create new instance")
	}
	s.scope.V(2).Info("Successfully created instance", "instance-id", instance.ID)

	return instance, nil

}

func (s *Service) DeleteInstance(id string) error {
	log.Info("Deleting instance resources")
	s.scope.V(2).Info("Attempting to delete instance", "instance-id", id)
	if id == "" {
		s.scope.Info("Instance does not have an instance id")
		return errors.New("cannot delete instance. instance does not have an instance id")
	}

	// Call the Delete method directly with the string id
	if err := s.scope.Instances.Delete(s.ctx, id); err != nil {
		return errors.Wrapf(err, "failed to delete instance with id %q", id)
	}

	s.scope.V(2).Info("Deleted instance", "instance-id", id)
	return nil
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

func (s *Service) AddInstanceToVLB(vlbID, instanceID string) error {
	for {
		currentVlb, _, err := s.scope.LoadBalancers.Get(context.TODO(), vlbID)
		if err != nil {
			return err
		}

		if currentVlb.Status != "active" {
			time.Sleep(10 * time.Second)
		} else {
			updateReq := govultr.LoadBalancerReq{}
			updateReq.Instances = append(currentVlb.Instances, instanceID)
			err := s.scope.LoadBalancers.Update(context.TODO(), vlbID, &updateReq)
			return err
		}
	}
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
