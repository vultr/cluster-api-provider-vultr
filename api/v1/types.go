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

package v1

// APIEndpoint represents control-plane's apiserver endpoints.
type APIEndpoint struct {
	// ID is the id of Vultr reserved IP (SUBID).
	ID string `json:"SUBID,string"`

	// The hostname on which the API server is serving.
	Host string `json:"host"`

	// The port on which the API server is serving.
	Port int `json:"port"`
}

// ServerStatus represents the status of subscription.
type SubscriptionStatus string

var (
	SubscriptionStatusPending   = SubscriptionStatus("pending")
	SubscriptionStatusActive    = SubscriptionStatus("active")
	SubscriptionStatusSuspended = SubscriptionStatus("suspended")
	SubscriptionStatusClosed    = SubscriptionStatus("closed")
)

// PowerStatus represents that the VPS power state
type PowerStatus string

var (
	PowerStatusStarting = PowerStatus("starting")
	PowerStatusStopped  = PowerStatus("stopped")
	PowerStatusRunning  = PowerStatus("running")
)

// ServerState represents the server state.
type ServerState string

var (
	ServerStateNone        = ServerState("none")
	ServerStateLocked      = ServerState("locked")
	ServerStateInstalling  = ServerState("installing")
	ServerStateBooting     = ServerState("booting")
	ServerStateIsoMounting = ServerState("isomounting")
	ServerStateOK          = ServerState("ok")
)
