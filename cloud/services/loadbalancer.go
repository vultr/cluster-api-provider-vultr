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
	"fmt"
	"net/http"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/govultr/v3"
)

// Service is a struct that would contain the context and client for Vultr interactions.
type Service struct {
	ctx    context.Context
	client *govultr.Client
	scope  *Scope
}

// Scope is a struct that encapsulates the necessary scope details, assumed to be defined elsewhere.
type Scope struct {
	Name   string
	Region string
	UID    string
}

// GetLoadBalancer retrieves a load balancer by its ID.
func (s *Service) GetLoadBalancer(id string) (*govultr.LoadBalancer, error) {
	if id == "" {
		return nil, nil
	}

	lb, resp, err := s.client.LoadBalancer.Get(s.ctx, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	return lb, nil
}

// CreateLoadBalancer creates a new load balancer.
func (s *Service) CreateLoadBalancer(spec *infrav1.VultrLoadBalancer) (*govultr.LoadBalancer, error) {
	if spec.HealthCheck == nil {
		return nil, fmt.Errorf("health check configuration is missing")
	}

	port := spec.HealthCheck.Port

	name := s.scope.Name + "-" + s.scope.UID
	createReq := &govultr.LoadBalancerReq{
		Label:  name,
		Region: s.scope.Region,
		ForwardingRules: []govultr.ForwardingRule{
			{
				FrontendProtocol: "tcp",
				FrontendPort:     port,
				BackendProtocol:  "tcp",
				BackendPort:      port,
			},
		},
		HealthCheck: &govultr.HealthCheck{
			Protocol:           spec.HealthCheck.Protocol,
			Port:               port,
			CheckInterval:      spec.HealthCheck.CheckInterval,
			ResponseTimeout:    spec.HealthCheck.ResponseTimeout,
			UnhealthyThreshold: spec.HealthCheck.UnhealthyThreshold,
			HealthyThreshold:   spec.HealthCheck.HealthyThreshold,
		},
		BalancingAlgorithm: spec.GenericInfo.BalancingAlgorithm,
	}

	lb, _, err := s.client.LoadBalancer.Create(s.ctx, createReq)
	if err != nil {
		return nil, err
	}

	return lb, nil
}

// // DeleteLoadBalancer delete a LB by ID.
// func (s *Service) DeleteLoadBalancer(id string) error {
// 	if _, err := s.scope.LoadBalancers.Delete(s.ctx, id); err != nil {
// 		return err
// 	}

// 	return nil
// }

// DeleteLoadBalancer deletes a load balancer by its ID.
func (s *Service) DeleteLoadBalancer(id string) error {
	if id == "" {
		return nil
	}

	if err := s.client.LoadBalancer.Delete(s.ctx, id); err != nil {
		return err
	}

	return nil
}
