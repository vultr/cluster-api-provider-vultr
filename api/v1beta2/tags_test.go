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
	"reflect"
	"testing"
)

const (
	testTagMachine = "prod-md-0-abc"
	testTagName    = "name:" + testTagMachine
)

func TestBuildTags(t *testing.T) {
	params := BuildTagParams{ClusterName: "prod", ClusterUID: "uid-1", Name: testTagMachine, Role: WorkerRoleTagValue}
	want := Tags{"capvultr:cluster:prod", "capvultr:cluster-uid:uid-1", "capvultr:role:worker", testTagName}
	if got := BuildTags(params); !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTags = %v, want %v", got, want)
	}
}

func TestTagsOwnedByMachine(t *testing.T) {
	params := BuildTagParams{ClusterName: "prod", ClusterUID: "uid-1", Name: testTagMachine, Role: WorkerRoleTagValue}
	tests := []struct {
		name string
		tags []string
		want bool
	}{
		{"all tags", BuildTags(params), true},
		{"same name, other cluster", []string{"capvultr:cluster-uid:uid-2", testTagName}, false},
		{"same cluster, other machine", []string{"capvultr:cluster-uid:uid-1", "name:prod-md-0-xyz"}, false},
		{"no tags", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TagsOwnedByMachine(tt.tags, params); got != tt.want {
				t.Fatalf("TagsOwnedByMachine = %v, want %v", got, tt.want)
			}
		})
	}
}
