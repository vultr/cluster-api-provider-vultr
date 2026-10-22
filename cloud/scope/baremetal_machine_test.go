package scope

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
)

func newTestBareMetalScope(providerID *string) *BareMetalMachineScope {
	return &BareMetalMachineScope{
		Machine: &clusterv1.Machine{},
		VultrBareMetalMachine: &infrav1.VultrBareMetalMachine{
			ObjectMeta: metav1.ObjectMeta{Name: "bm-0", Namespace: "default"},
			Spec:       infrav1.VultrBareMetalMachineSpec{ProviderID: providerID},
		},
	}
}

func TestBareMetalMachineScope_GetServerID(t *testing.T) {
	tests := []struct {
		name       string
		providerID *string
		want       string
	}{
		{name: "unset", providerID: nil, want: ""},
		{name: "valid", providerID: ptr.To("vultr://f34d1696"), want: "f34d1696"},
		{name: "malformed", providerID: ptr.To("f34d1696"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newTestBareMetalScope(tt.providerID).GetServerID(); got != tt.want {
				t.Errorf("GetServerID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBareMetalMachineScope_SetProviderID(t *testing.T) {
	s := newTestBareMetalScope(nil)
	s.SetProviderID("f34d1696")
	if got := s.GetProviderID(); got != "vultr://f34d1696" {
		t.Errorf("GetProviderID() after SetProviderID = %q", got)
	}
	if got := s.GetServerID(); got != "f34d1696" {
		t.Errorf("GetServerID() after SetProviderID = %q", got)
	}
}

func TestBareMetalMachineScope_Role(t *testing.T) {
	s := newTestBareMetalScope(nil)
	if s.IsControlPlane() || s.Role() != infrav1.WorkerRoleTagValue {
		t.Errorf("worker machine reported as control plane: IsControlPlane=%v Role=%q", s.IsControlPlane(), s.Role())
	}
	s.Machine.Labels = map[string]string{clusterv1.MachineControlPlaneLabel: ""}
	if !s.IsControlPlane() || s.Role() != infrav1.ControlPlaneRoleTagValue {
		t.Errorf("control plane machine not detected: IsControlPlane=%v Role=%q", s.IsControlPlane(), s.Role())
	}
}

func TestBareMetalMachineScope_TerminalFailure(t *testing.T) {
	s := newTestBareMetalScope(nil)
	if s.HasTerminalFailure() {
		t.Fatal("new machine must not report a terminal failure")
	}

	// A non-terminal Ready=False (e.g. still provisioning) is not terminal.
	conditions.Set(s.VultrBareMetalMachine, metav1.Condition{
		Type:   infrav1.BareMetalMachineReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: infrav1.BareMetalServerPendingReason,
	})
	if s.HasTerminalFailure() {
		t.Fatal("pending server must not be a terminal failure")
	}

	s.SetTerminalFailure(infrav1.BareMetalVPCAttachFailedReason, "Plan does not support VPC networking")
	if !s.HasTerminalFailure() {
		t.Fatal("expected terminal failure after SetTerminalFailure")
	}
	if s.VultrBareMetalMachine.Status.Ready {
		t.Fatal("Ready must be false after a terminal failure")
	}
	c := conditions.Get(s.VultrBareMetalMachine, infrav1.BareMetalMachineReadyCondition)
	if c == nil || c.Reason != infrav1.BareMetalVPCAttachFailedReason || c.Message == "" {
		t.Fatalf("unexpected Ready condition: %+v", c)
	}
}

func TestBareMetalMachineScope_Setters(t *testing.T) {
	s := newTestBareMetalScope(nil)
	s.SetHardware(4, "32768 MB", "2x 240GB SSD")
	s.SetServerStatus(infrav1.SubscriptionStatusActive)
	s.SetProvisioned()
	s.SetReady()
	s.SetVPC(&infrav1.VultrBareMetalVPCStatus{ID: "vpc", IPAddress: "10.1.96.10"})

	st := s.VultrBareMetalMachine.Status
	if st.CPU != 4 || st.RAM != "32768 MB" || st.Disk != "2x 240GB SSD" {
		t.Errorf("unexpected hardware status: %+v", st)
	}
	if st.SubscriptionStatus == nil || *st.SubscriptionStatus != infrav1.SubscriptionStatusActive {
		t.Errorf("unexpected subscription status: %v", st.SubscriptionStatus)
	}
	if st.Initialization.Provisioned == nil || !*st.Initialization.Provisioned || !st.Ready {
		t.Errorf("expected provisioned and ready, got %+v", st)
	}
	if st.VPC == nil || st.VPC.IPAddress != "10.1.96.10" {
		t.Errorf("unexpected vpc status: %+v", st.VPC)
	}
}
