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

	"github.com/pkg/errors"

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
