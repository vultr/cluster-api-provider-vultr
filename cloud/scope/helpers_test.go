package scope

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
)

func TestProviderIDToResourceID(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		want       string
	}{
		{name: "valid", providerID: "vultr://8b1a2c3d", want: "8b1a2c3d"},
		{name: "empty", providerID: "", want: ""},
		{name: "wrong scheme", providerID: "aws://i-123", want: ""},
		{name: "missing scheme", providerID: "8b1a2c3d", want: ""},
		{name: "too many separators", providerID: "vultr://a://b", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProviderIDToResourceID(tt.providerID); got != tt.want {
				t.Errorf("ProviderIDToResourceID(%q) = %q, want %q", tt.providerID, got, tt.want)
			}
		})
	}
}

func TestResourceIDToProviderID(t *testing.T) {
	if got := ResourceIDToProviderID("abc"); got != "vultr://abc" {
		t.Errorf("ResourceIDToProviderID() = %q, want %q", got, "vultr://abc")
	}
	if got := ProviderIDToResourceID(ResourceIDToProviderID("abc")); got != "abc" {
		t.Errorf("round trip = %q, want %q", got, "abc")
	}
}

func TestMachineRole(t *testing.T) {
	worker := &clusterv1.Machine{}
	if got := MachineRole(worker); got != infrav1.WorkerRoleTagValue {
		t.Errorf("MachineRole(worker) = %q, want %q", got, infrav1.WorkerRoleTagValue)
	}

	controlPlane := &clusterv1.Machine{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{clusterv1.MachineControlPlaneLabel: ""}}}
	if got := MachineRole(controlPlane); got != infrav1.ControlPlaneRoleTagValue {
		t.Errorf("MachineRole(control plane) = %q, want %q", got, infrav1.ControlPlaneRoleTagValue)
	}
}
