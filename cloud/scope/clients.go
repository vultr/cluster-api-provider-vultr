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

import "github.com/vultr/govultr/v3"

// VultrClients hold all necessary clients to work with the Vultr API.
type VultrAPIClients struct {
	Instances     govultr.InstanceService
	LoadBalancers govultr.LoadBalancerService
	VPC2s         govultr.VPC2Service
	VPCs          govultr.VPCService
	SSHKeys       govultr.SSHKeyService
	Snapshots     govultr.SnapshotService
}
