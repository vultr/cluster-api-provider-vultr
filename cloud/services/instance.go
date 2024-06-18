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
	"net/http"

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/util"
	"github.com/vultr/govultr/v3"
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
		return nil, errors.Wrapf(err, "failed to get instance with ID %q", instanceID)
	}

	return instance, nil
}

func (s *Service) CreateInstance(scope *scope.MachineScope) (*govultr.Instance, error) {
	s.scope.V(2).Info("Creating an instance for a machine")

	bootstrapData, err := scope.GetBootstrapData()
	if err != nil {
		log.Error(err, "Error getting bootstrap data for machine")
		return nil, errors.Wrap(err, "failed to retrieve bootstrap data")
	}

	clusterName := s.scope.Name()
	instanceName := scope.Name()

	// Prepare the request payload
	instanceReq := &govultr.InstanceCreateReq{
		Label:      instanceName,
		Region:     s.scope.Region(),
		Plan:       scope.VultrMachine.Spec.PlanID,
		OsID:       scope.VultrMachine.Spec.OSID,
		UserData:   bootstrapData,
		EnableVPC2: util.Pointer(true),
	}

	instanceReq.Tags = infrav1.BuildTags(infrav1.BuildTagParams{
		ClusterName: clusterName,
		ClusterUID:  s.scope.UID(),
		Name:        instanceName,
		Role:        scope.Role(),
	})

	instance, _, err := s.scope.Instances.Create(s.ctx, instanceReq)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create new instance")
	}

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
