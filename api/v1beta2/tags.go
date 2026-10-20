/*
Copyright 2024.

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

package v1beta2

import (
	"fmt"
	"slices"
)

// Tags defines a slice of tags.
type Tags []string

const (
	// TagPrefix prefixes every tag the provider sets. Each tag carries one
	// fact, so Vultr's tag filter can select on any of them.
	TagPrefix = "capvultr"
	// ControlPlaneRoleTagValue is the role tag value for control plane machines.
	ControlPlaneRoleTagValue = "control-plane"
	// WorkerRoleTagValue is the role tag value for worker machines.
	WorkerRoleTagValue = "worker"
)

// ClusterTag returns `capvultr:cluster:{clusterName}`.
func ClusterTag(clusterName string) string {
	return fmt.Sprintf("%s:cluster:%s", TagPrefix, clusterName)
}

// ClusterUIDTag returns `capvultr:cluster-uid:{uid}`.
func ClusterUIDTag(clusterUID string) string {
	return fmt.Sprintf("%s:cluster-uid:%s", TagPrefix, clusterUID)
}

// RoleTag returns `capvultr:role:{role}`.
func RoleTag(role string) string {
	return fmt.Sprintf("%s:role:%s", TagPrefix, role)
}

// NameTagFromName returns `name:{name}`.
func NameTagFromName(name string) string {
	return fmt.Sprintf("name:%s", name)
}

// BuildTagParams is used to build tags around a Vultr resource.
type BuildTagParams struct {
	// ClusterName is the cluster associated with the resource.
	ClusterName string
	// ClusterUID is the cluster uid from clusters.cluster.x-k8s.io uid
	ClusterUID string
	// Name is the name of the resource, it's applied as the tag "name" on Vultr.
	Name string
	// Role is the role associated to the resource.
	Role string
}

// BuildTags returns the tags for a resource of the cluster.
func BuildTags(params BuildTagParams) Tags {
	return Tags{
		ClusterTag(params.ClusterName),
		ClusterUIDTag(params.ClusterUID),
		RoleTag(params.Role),
		NameTagFromName(params.Name),
	}
}

// TagsOwnedByMachine reports whether tags identify the named machine of the cluster.
func TagsOwnedByMachine(tags []string, params BuildTagParams) bool {
	return slices.Contains(tags, NameTagFromName(params.Name)) &&
		slices.Contains(tags, ClusterUIDTag(params.ClusterUID))
}
