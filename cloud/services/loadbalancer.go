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
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/govultr/v3"
)

func (s *Service) GetLoadBalancer(id string) (*govultr.LoadBalancer, error) {
	if id == "" {
		return nil, nil
	}

	lb, _, err := s.scope.LoadBalancers.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}

	return lb, nil
}

// CreateLoadBalancer creates a new load balancer.
func (s *Service) CreateLoadBalancer(spec *infrav1.VultrLoadBalancer) (*govultr.LoadBalancer, error) {
	name := s.scope.Name() + "-" + s.scope.UID()
	createReq := &govultr.LoadBalancerReq{
		Label:  name,
		Region: s.scope.Region(),
		ForwardingRules: []govultr.ForwardingRule{
			{
				FrontendProtocol: "tcp",
				FrontendPort:     spec.HealthCheck.Port,
				BackendProtocol:  "tcp",
				BackendPort:      spec.HealthCheck.Port,
			},
		},
		HealthCheck: &govultr.HealthCheck{
			Protocol:           "tcp",
			Port:               spec.HealthCheck.Port,
			CheckInterval:      spec.HealthCheck.CheckInterval,
			ResponseTimeout:    spec.HealthCheck.ResponseTimeout,
			UnhealthyThreshold: spec.HealthCheck.UnhealthyThreshold,
			HealthyThreshold:   spec.HealthCheck.HealthyThreshold,
		},
		BalancingAlgorithm: spec.GenericInfo.BalancingAlgorithm,
	}

	lb, _, err := s.scope.LoadBalancers.Create(s.ctx, createReq)
	if err != nil {
		return nil, err
	}

	return lb, nil
}

// DeleteLoadBalancer deletes a load balancer by its ID.
func (s *Service) DeleteLoadBalancer(id string) error {
	if err := s.scope.LoadBalancers.Delete(s.ctx, id); err != nil {
		return err
	}

	return nil
}
