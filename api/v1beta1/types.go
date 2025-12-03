/*

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

package v1beta1

// ServerStatus represents the status of subscription.
type SubscriptionStatus string

var (
	SubscriptionStatusPending   = SubscriptionStatus("pending")
	SubscriptionStatusActive    = SubscriptionStatus("active")
	SubscriptionStatusSuspended = SubscriptionStatus("suspended")
	SubscriptionStatusClosed    = SubscriptionStatus("closed")
)

// PowerStatus represents that the VPS is powerd on or not
type PowerStatus string

var (
	PowerStatusStarting = PowerStatus("starting")
	PowerStatusStopped  = PowerStatus("stopped")
	PowerStatusRunning  = PowerStatus("running")
)

// ServerState represents a detail of server state.
type ServerState string

var (
	ServerStateNone        = ServerState("none")
	ServerStateLocked      = ServerState("locked")
	ServerStateBooting     = ServerState("installingbooting")
	ServerStateIsoMounting = ServerState("isomounting")
	ServerStateOK          = ServerState("ok")
	ServerStateError       = ServerState("error")
)

// VultrResourceReference is a reference to a Vultr resource.
type VultrResourceReference struct {
	// ID of Vultr resource
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
	// Status of a Vultr resource
	// +optional
	ResourceSubscriptionStatus SubscriptionStatus `json:"resourceStatus,omitempty"`
	// Power Status of a Vultr resource
	// +optional
	ResourcePowerStatus PowerStatus `json:"powerStatus,omitempty"`
	// Server state of a Vultr resource
	// +optional
	ResourceServerState ServerState `json:"serverState,omitempty"`
}

// VultrNetworkResource encapsulates Vultr networking resources.
type VultrNetworkResource struct {
	// APIServerLoadbalancersRef is the id of apiserver loadbalancers.
	// +optional
	APIServerLoadbalancersRef VultrResourceReference `json:"apiServerLoadbalancersRef,omitempty"`
}

// NetworkSpec encapsulates Vultr networking configuration.
type NetworkSpec struct {
	// Configures an API Server loadbalancers
	// +optional
	APIServerLoadbalancers VultrLoadBalancer `json:"apiServerLoadbalancers,omitempty"`
}

// VultrLoadBalancer represents the structure of a Vultr load balancer
type VultrLoadBalancer struct {
	ID              string           `json:"id,omitempty"`
	DateCreated     string           `json:"date_created,omitempty"`
	Region          string           `json:"region,omitempty"`
	Label           string           `json:"label,omitempty"`
	Status          string           `json:"status,omitempty"`
	IPV4            string           `json:"ipv4,omitempty"`
	IPV6            string           `json:"ipv6,omitempty"`
	Instances       []string         `json:"instances,omitempty"`
	Nodes           int              `json:"nodes,omitempty"`
	HealthCheck     *HealthCheck     `json:"health_check,omitempty"`
	GenericInfo     *GenericInfo     `json:"generic_info,omitempty"`
	SSLInfo         *bool            `json:"has_ssl,omitempty"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules,omitempty"`
	FirewallRules   []LBFirewallRule `json:"firewall_rules,omitempty"`
}

// HealthCheck represents your health check configuration for your load balancer.
type HealthCheck struct {
	Protocol           string `json:"protocol,omitempty"`
	Port               int    `json:"port,omitempty"`
	Path               string `json:"path,omitempty"`
	CheckInterval      int    `json:"check_interval,omitempty"`
	ResponseTimeout    int    `json:"response_timeout,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
}

// GenericInfo represents generic configuration of your load balancer
type GenericInfo struct {
	BalancingAlgorithm string          `json:"balancing_algorithm,omitempty"`
	SSLRedirect        *bool           `json:"ssl_redirect,omitempty"`
	StickySessions     *StickySessions `json:"sticky_sessions,omitempty"`
	ProxyProtocol      *bool           `json:"proxy_protocol,omitempty"`
	VPC                string          `json:"vpc,omitempty"`
}

// StickySessions represents cookie for your load balancer
type StickySessions struct {
	CookieName string `json:"cookie_name,omitempty"`
}

// ForwardingRule represent a single forwarding rule
type ForwardingRule struct {
	RuleID           string `json:"id,omitempty"`
	FrontendProtocol string `json:"frontend_protocol,omitempty"`
	FrontendPort     int    `json:"frontend_port,omitempty"`
	BackendProtocol  string `json:"backend_protocol,omitempty"`
	BackendPort      int    `json:"backend_port,omitempty"`
}

// LBFirewallRule represents a single firewall rule
type LBFirewallRule struct {
	RuleID string `json:"id,omitempty"`
	Port   int    `json:"port,omitempty"`
	IPType string `json:"ip_type,omitempty"`
	Source string `json:"source,omitempty"`
}

// VultrMachineTemplateResource describes the data needed to create a VultrMachine from a template.
type VultrMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec VultrMachineSpec `json:"spec"`
}

// ApplyDefaults sets default values for VultrLoadBalancer fields if they are not set.
func (in *VultrLoadBalancer) ApplyDefaults() {
	// Set default for HealthCheck if it is not initialized
	if in.HealthCheck == nil {
		in.HealthCheck = &HealthCheck{}
	}

	// Set default HealthCheck values if they are not set
	if in.HealthCheck.Port == 0 {
		in.HealthCheck.Port = DefaultLBPort
	}
	if in.HealthCheck.CheckInterval == 0 {
		in.HealthCheck.CheckInterval = DefaultLBHealthCheckInterval
	}
	if in.HealthCheck.ResponseTimeout == 0 {
		in.HealthCheck.ResponseTimeout = DefaultLBHealthCheckTimeout
	}
	if in.HealthCheck.UnhealthyThreshold == 0 {
		in.HealthCheck.UnhealthyThreshold = DefaultLBHealthCheckUnhealthyThreshold
	}
	if in.HealthCheck.HealthyThreshold == 0 {
		in.HealthCheck.HealthyThreshold = DefaultLBHealthCheckHealthyThreshold
	}

	// Set default for GenericInfo if it is not initialized
	if in.GenericInfo == nil {
		in.GenericInfo = &GenericInfo{}
	}

	// Set default GenericInfo values if they are not set
	if in.GenericInfo.BalancingAlgorithm == "" {
		in.GenericInfo.BalancingAlgorithm = DefaultLBAlgorithm
	}
}

var (
	// Default values for VultrLoadBalancer fields
	DefaultLBPort                          = 6443
	DefaultLBAlgorithm                     = "roundrobin"
	DefaultLBHealthCheckInterval           = 15
	DefaultLBHealthCheckTimeout            = 5
	DefaultLBHealthCheckUnhealthyThreshold = 5
	DefaultLBHealthCheckHealthyThreshold   = 5
)
