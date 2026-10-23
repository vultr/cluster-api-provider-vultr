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
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"

	"net/http"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/govultr/v3"
)

const tcpProtocol = "tcp"

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
		VPC:    s.scope.VPC(),
		ForwardingRules: []govultr.ForwardingRule{
			{
				FrontendProtocol: tcpProtocol,
				FrontendPort:     spec.HealthCheck.Port,
				BackendProtocol:  tcpProtocol,
				BackendPort:      spec.HealthCheck.Port,
			},
		},
		HealthCheck: &govultr.HealthCheck{
			Protocol:           tcpProtocol,
			Port:               spec.HealthCheck.Port,
			CheckInterval:      spec.HealthCheck.CheckInterval,
			ResponseTimeout:    spec.HealthCheck.ResponseTimeout,
			UnhealthyThreshold: spec.HealthCheck.UnhealthyThreshold,
			HealthyThreshold:   spec.HealthCheck.HealthyThreshold,
		},
		BalancingAlgorithm: spec.GenericInfo.BalancingAlgorithm,
	}

	if len(spec.FirewallRules) > 0 {
		//nolint:prealloc
		fwRules := make([]govultr.LBFirewallRule, 0)
		for _, r := range spec.FirewallRules {
			fwRules = append(fwRules, govultr.LBFirewallRule{
				IPType: r.IPType,
				Port:   r.Port,
				Source: r.Source,
			})
		}
		createReq.FirewallRules = fwRules
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

// EnsureLoadBalancerMember makes sure the resource (instance or bare metal
// server id) is a backend of the load balancer. It returns ready=false while
// the load balancer is not accepting members yet so the caller can requeue,
// and it is idempotent: a resource that is already a member is not appended
// again.
func (s *Service) EnsureLoadBalancerMember(lbID, resourceID string) (bool, error) {
	if lbID == "" {
		return false, errors.New("load balancer id is empty")
	}

	lb, _, err := s.scope.LoadBalancers.Get(s.ctx, lbID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get load balancer %q", lbID)
	}
	if lb.Status != "active" {
		return false, nil
	}
	for _, id := range lb.Instances {
		if id == resourceID {
			return true, nil
		}
	}

	req := &govultr.LoadBalancerReq{Instances: append(append([]string{}, lb.Instances...), resourceID)}
	ctx, status := vultrerrors.Capture(s.ctx)
	if err := s.scope.LoadBalancers.Update(ctx, lbID, req); err != nil {
		// 422 means the load balancer is not accepting members yet.
		if status() == http.StatusUnprocessableEntity {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to add %q to load balancer %q", resourceID, lbID)
	}
	return true, nil
}

// RemoveLoadBalancerMember removes the resource from the load balancer backends
// if present. A missing load balancer is not an error.
func (s *Service) RemoveLoadBalancerMember(lbID, resourceID string) error {
	if lbID == "" {
		return nil
	}

	ctx, status := vultrerrors.Capture(s.ctx)
	lb, _, err := s.scope.LoadBalancers.Get(ctx, lbID)
	if err != nil {
		if status() == http.StatusNotFound {
			return nil
		}
		return errors.Wrapf(err, "failed to get load balancer %q", lbID)
	}

	remaining := make([]string, 0, len(lb.Instances))
	for _, id := range lb.Instances {
		if id != resourceID {
			remaining = append(remaining, id)
		}
	}
	if len(remaining) == len(lb.Instances) {
		return nil
	}

	// govultr omits an empty instances list, so send a JSON null-safe empty slice
	// through an explicit update only when members remain; an empty backend set is
	// cleared by the load balancer deletion that follows the last control plane.
	if len(remaining) == 0 {
		return nil
	}
	if err := s.scope.LoadBalancers.Update(s.ctx, lbID, &govultr.LoadBalancerReq{Instances: remaining}); err != nil {
		return errors.Wrapf(err, "failed to remove %q from load balancer %q", resourceID, lbID)
	}
	return nil
}
