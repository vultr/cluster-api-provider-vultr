package v1beta2

// Condition types
const (
	VultrClusterReadyCondition string = "Ready"
	LoadBalancerReadyCondition string = "LoadBalancerReady"
	MachineReadyCondition      string = "Ready"

	ErrorFetchingLBReason          string = "ErrorFetchingLoadBalancer"
	ErrorCreatingLBReason          string = "ErrorCreatingLoadBalancer"
	LoadBalancerProvisioningFailed string = "LoadBalancerProvisioningFailed"
	WaitingForIPReason             string = "WaitingForIP"
)

// VultrMachine Ready condition reasons.
const (
	InstanceWaitingForClusterInfrastructureReason string = "WaitingForClusterInfrastructure"
	InstanceWaitingForBootstrapDataReason         string = "WaitingForBootstrapData"
	InstanceWaitingForLoadBalancerReason          string = "WaitingForLoadBalancer"
	InstancePendingReason                         string = "InstancePending"
	InstanceActiveReason                          string = "InstanceActive"
	InstanceCreationFailedReason                  string = "InstanceCreationFailed"
	InstanceAddressFailedReason                   string = "FailedToGetInstanceAddress"
	InstanceDeletingReason                        string = "Deleting"

	InstanceNotFoundReason         string = "InstanceNotFound"
	InstanceUnexpectedStatusReason string = "UnexpectedStatus"
)

// VultrBareMetalMachine condition types and reasons.
const (
	BareMetalMachineReadyCondition string = "Ready"

	BareMetalWaitingForClusterInfrastructureReason string = "WaitingForClusterInfrastructure"
	BareMetalWaitingForBootstrapDataReason         string = "WaitingForBootstrapData"
	BareMetalServerPendingReason                   string = "ServerPending"
	BareMetalVPCAttachRequestedReason              string = "VPCAttachRequested"
	BareMetalWaitingForLoadBalancerReason          string = "WaitingForLoadBalancer"
	BareMetalServerActiveReason                    string = "ServerActive"
	BareMetalDeletingReason                        string = "Deleting"

	// Terminal reasons: the machine will not be reconciled further and should be
	// replaced (for example by a MachineHealthCheck).
	BareMetalServerCreationFailedReason string = "ServerCreationFailed"
	BareMetalServerNotFoundReason       string = "ServerNotFound"
	BareMetalUnexpectedStatusReason     string = "UnexpectedStatus"
	BareMetalVPCAttachFailedReason      string = "VPCAttachFailed"
)
