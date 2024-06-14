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

// Package scope implements scope types.
package scope

import (
	"context"
	"fmt"

	"github.com/vultr/govultr/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VultrClients hold all necessary clients to work with the Vultr API.
type VultrAPIClients struct {
	Instances     govultr.InstanceService
	LoadBalancers govultr.LoadBalancerService
	VPC           govultr.VPCService
	SSHKEYS       govultr.SSHKeyService
	K8sClient     client.Client
}

func (v *VultrAPIClients) Update(ctx context.Context, obj client.Object) error {
	if err := v.K8sClient.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object: %w", err)
	}
	return nil
}

func (v *VultrAPIClients) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	if err := v.K8sClient.Get(ctx, key, obj); err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	return nil
}
