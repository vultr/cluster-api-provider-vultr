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

package scope

import (
	"testing"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMachineScope_SetCPU(t *testing.T) {
	tests := []struct {
		name     string
		cpu      int
		expected int
	}{
		{
			name:     "set cpu to 2",
			cpu:      2,
			expected: 2,
		},
		{
			name:     "set cpu to 8",
			cpu:      8,
			expected: 8,
		},
		{
			name:     "set cpu to 0",
			cpu:      0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machineScope := &MachineScope{
				VultrMachine: &infrav1.VultrMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-machine",
						Namespace: "default",
					},
				},
			}

			machineScope.SetCPU(tt.cpu)

			if machineScope.VultrMachine.Status.CPU != tt.expected {
				t.Errorf("SetCPU() = %v, want %v", machineScope.VultrMachine.Status.CPU, tt.expected)
			}
		})
	}
}

func TestMachineScope_SetRAM(t *testing.T) {
	tests := []struct {
		name     string
		ram      int
		expected int
	}{
		{
			name:     "set ram to 1024 MB",
			ram:      1024,
			expected: 1024,
		},
		{
			name:     "set ram to 8192 MB",
			ram:      8192,
			expected: 8192,
		},
		{
			name:     "set ram to 0",
			ram:      0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machineScope := &MachineScope{
				VultrMachine: &infrav1.VultrMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-machine",
						Namespace: "default",
					},
				},
			}

			machineScope.SetRAM(tt.ram)

			if machineScope.VultrMachine.Status.RAM != tt.expected {
				t.Errorf("SetRAM() = %v, want %v", machineScope.VultrMachine.Status.RAM, tt.expected)
			}
		})
	}
}

func TestMachineScope_SetStorage(t *testing.T) {
	tests := []struct {
		name     string
		storage  int
		expected int
	}{
		{
			name:     "set storage to 25 GB",
			storage:  25,
			expected: 25,
		},
		{
			name:     "set storage to 100 GB",
			storage:  100,
			expected: 100,
		},
		{
			name:     "set storage to 0",
			storage:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machineScope := &MachineScope{
				VultrMachine: &infrav1.VultrMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-machine",
						Namespace: "default",
					},
				},
			}

			machineScope.SetStorage(tt.storage)

			if machineScope.VultrMachine.Status.Storage != tt.expected {
				t.Errorf("SetStorage() = %v, want %v", machineScope.VultrMachine.Status.Storage, tt.expected)
			}
		})
	}
}
