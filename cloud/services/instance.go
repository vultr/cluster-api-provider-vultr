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

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
)

// GetInstance retrieves an instance by its ID.
func (s *Service) GetInstance(instanceID string) (*govultr.Instance, error) {
	if instanceID == "" {
		s.scope.Logger.Info("VultrInstance does not have an instance id")
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
	bootstrapData = bootstrapData + "\r  - ufw disable"
	encodedBootstrapData := base64.StdEncoding.EncodeToString([]byte(bootstrapData))
	if err != nil {
		log.Error(err, "Error getting bootstrap data for machine")
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}
	s.scope.V(2).Info("Successfully retrieved bootstrap data")

	clusterName := s.scope.Name()
	instanceName := scope.Name()

	// Prepare the request payload
	s.scope.V(2).Info("Preparing instance creation request payload")
	instanceReq := &govultr.InstanceCreateReq{
		Label:      instanceName,
		Hostname:   instanceName,
		Region:     s.scope.Region(),
		Plan:       scope.VultrMachine.Spec.PlanID,
		SnapshotID: scope.VultrMachine.Spec.Snapshot,
		UserData:   encodedBootstrapData,
		//EnableIPv6: util.Pointer(true),
	}

	if scope.VultrMachine.Spec.VPCID != "" {
		instanceReq.AttachVPC = append(instanceReq.AttachVPC, scope.VultrMachine.Spec.VPCID)
	} else if scope.VultrMachine.Spec.VPC2ID != "" {
		instanceReq.AttachVPC2 = append(instanceReq.AttachVPC2, scope.VultrMachine.Spec.VPCID)
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

	if len(addresses) == 0 {
		return addresses, errors.New("no IP addresses found for the instance")
	}

	return addresses, nil
}

func (s *Service) AddInstanceToVLB(vlbID, instanceID string) error {
	s.scope.Info("Attempting to retrieve current VLB", "vlb-id", vlbID)
	currentVlb, _, err := s.scope.LoadBalancers.Get(context.TODO(), vlbID)
	if err != nil {
		s.scope.Error(err, "Failed to retrieve current VLB", "vlb-id", vlbID)
		return err
	}

	s.scope.Info("Successfully retrieved current VLB", "vlb-id", vlbID)
	updateReq := govultr.LoadBalancerReq{Instances: append(currentVlb.Instances, instanceID)}
	err = s.scope.LoadBalancers.Update(context.TODO(), vlbID, &updateReq)
	if err != nil {
		s.scope.Error(err, "Failed to update VLB", "vlb-id", vlbID, "instance-id", instanceID)
		return err
	}

	s.scope.Info("Successfully updated VLB", "vlb-id", vlbID, "instance-id", instanceID)
	return nil
}
