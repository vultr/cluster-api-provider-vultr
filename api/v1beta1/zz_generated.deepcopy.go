//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuildTagParams) DeepCopyInto(out *BuildTagParams) {
	*out = *in
	if in.Additional != nil {
		in, out := &in.Additional, &out.Additional
		*out = make(Tags, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuildTagParams.
func (in *BuildTagParams) DeepCopy() *BuildTagParams {
	if in == nil {
		return nil
	}
	out := new(BuildTagParams)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRule) DeepCopyInto(out *ForwardingRule) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRule.
func (in *ForwardingRule) DeepCopy() *ForwardingRule {
	if in == nil {
		return nil
	}
	out := new(ForwardingRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericInfo) DeepCopyInto(out *GenericInfo) {
	*out = *in
	if in.SSLRedirect != nil {
		in, out := &in.SSLRedirect, &out.SSLRedirect
		*out = new(bool)
		**out = **in
	}
	if in.StickySessions != nil {
		in, out := &in.StickySessions, &out.StickySessions
		*out = new(StickySessions)
		**out = **in
	}
	if in.ProxyProtocol != nil {
		in, out := &in.ProxyProtocol, &out.ProxyProtocol
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericInfo.
func (in *GenericInfo) DeepCopy() *GenericInfo {
	if in == nil {
		return nil
	}
	out := new(GenericInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheck) DeepCopyInto(out *HealthCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheck.
func (in *HealthCheck) DeepCopy() *HealthCheck {
	if in == nil {
		return nil
	}
	out := new(HealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LBFirewallRule) DeepCopyInto(out *LBFirewallRule) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LBFirewallRule.
func (in *LBFirewallRule) DeepCopy() *LBFirewallRule {
	if in == nil {
		return nil
	}
	out := new(LBFirewallRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkSpec) DeepCopyInto(out *NetworkSpec) {
	*out = *in
	in.APIServerLoadbalancers.DeepCopyInto(&out.APIServerLoadbalancers)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkSpec.
func (in *NetworkSpec) DeepCopy() *NetworkSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StickySessions) DeepCopyInto(out *StickySessions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StickySessions.
func (in *StickySessions) DeepCopy() *StickySessions {
	if in == nil {
		return nil
	}
	out := new(StickySessions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Tags) DeepCopyInto(out *Tags) {
	{
		in := &in
		*out = make(Tags, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tags.
func (in Tags) DeepCopy() Tags {
	if in == nil {
		return nil
	}
	out := new(Tags)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrCluster) DeepCopyInto(out *VultrCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrCluster.
func (in *VultrCluster) DeepCopy() *VultrCluster {
	if in == nil {
		return nil
	}
	out := new(VultrCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterList) DeepCopyInto(out *VultrClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VultrCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterList.
func (in *VultrClusterList) DeepCopy() *VultrClusterList {
	if in == nil {
		return nil
	}
	out := new(VultrClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterSpec) DeepCopyInto(out *VultrClusterSpec) {
	*out = *in
	in.Network.DeepCopyInto(&out.Network)
	out.ControlPlaneEndpoint = in.ControlPlaneEndpoint
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterSpec.
func (in *VultrClusterSpec) DeepCopy() *VultrClusterSpec {
	if in == nil {
		return nil
	}
	out := new(VultrClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterStatus) DeepCopyInto(out *VultrClusterStatus) {
	*out = *in
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(errors.ClusterStatusError)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Network = in.Network
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterStatus.
func (in *VultrClusterStatus) DeepCopy() *VultrClusterStatus {
	if in == nil {
		return nil
	}
	out := new(VultrClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterTemplate) DeepCopyInto(out *VultrClusterTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterTemplate.
func (in *VultrClusterTemplate) DeepCopy() *VultrClusterTemplate {
	if in == nil {
		return nil
	}
	out := new(VultrClusterTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrClusterTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterTemplateList) DeepCopyInto(out *VultrClusterTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VultrClusterTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterTemplateList.
func (in *VultrClusterTemplateList) DeepCopy() *VultrClusterTemplateList {
	if in == nil {
		return nil
	}
	out := new(VultrClusterTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrClusterTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterTemplateResource) DeepCopyInto(out *VultrClusterTemplateResource) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterTemplateResource.
func (in *VultrClusterTemplateResource) DeepCopy() *VultrClusterTemplateResource {
	if in == nil {
		return nil
	}
	out := new(VultrClusterTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrClusterTemplateSpec) DeepCopyInto(out *VultrClusterTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrClusterTemplateSpec.
func (in *VultrClusterTemplateSpec) DeepCopy() *VultrClusterTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(VultrClusterTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrLoadBalancer) DeepCopyInto(out *VultrLoadBalancer) {
	*out = *in
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.HealthCheck != nil {
		in, out := &in.HealthCheck, &out.HealthCheck
		*out = new(HealthCheck)
		**out = **in
	}
	if in.GenericInfo != nil {
		in, out := &in.GenericInfo, &out.GenericInfo
		*out = new(GenericInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.SSLInfo != nil {
		in, out := &in.SSLInfo, &out.SSLInfo
		*out = new(bool)
		**out = **in
	}
	if in.ForwardingRules != nil {
		in, out := &in.ForwardingRules, &out.ForwardingRules
		*out = make([]ForwardingRule, len(*in))
		copy(*out, *in)
	}
	if in.FirewallRules != nil {
		in, out := &in.FirewallRules, &out.FirewallRules
		*out = make([]LBFirewallRule, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrLoadBalancer.
func (in *VultrLoadBalancer) DeepCopy() *VultrLoadBalancer {
	if in == nil {
		return nil
	}
	out := new(VultrLoadBalancer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachine) DeepCopyInto(out *VultrMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachine.
func (in *VultrMachine) DeepCopy() *VultrMachine {
	if in == nil {
		return nil
	}
	out := new(VultrMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineList) DeepCopyInto(out *VultrMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VultrMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineList.
func (in *VultrMachineList) DeepCopy() *VultrMachineList {
	if in == nil {
		return nil
	}
	out := new(VultrMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineSpec) DeepCopyInto(out *VultrMachineSpec) {
	*out = *in
	if in.ProviderID != nil {
		in, out := &in.ProviderID, &out.ProviderID
		*out = new(string)
		**out = **in
	}
	if in.SSHKey != nil {
		in, out := &in.SSHKey, &out.SSHKey
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineSpec.
func (in *VultrMachineSpec) DeepCopy() *VultrMachineSpec {
	if in == nil {
		return nil
	}
	out := new(VultrMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineStatus) DeepCopyInto(out *VultrMachineStatus) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]v1.NodeAddress, len(*in))
		copy(*out, *in)
	}
	if in.SubscriptionStatus != nil {
		in, out := &in.SubscriptionStatus, &out.SubscriptionStatus
		*out = new(SubscriptionStatus)
		**out = **in
	}
	if in.PowerStatus != nil {
		in, out := &in.PowerStatus, &out.PowerStatus
		*out = new(PowerStatus)
		**out = **in
	}
	if in.ServerState != nil {
		in, out := &in.ServerState, &out.ServerState
		*out = new(ServerState)
		**out = **in
	}
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(errors.MachineStatusError)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineStatus.
func (in *VultrMachineStatus) DeepCopy() *VultrMachineStatus {
	if in == nil {
		return nil
	}
	out := new(VultrMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineTemplate) DeepCopyInto(out *VultrMachineTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineTemplate.
func (in *VultrMachineTemplate) DeepCopy() *VultrMachineTemplate {
	if in == nil {
		return nil
	}
	out := new(VultrMachineTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrMachineTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineTemplateList) DeepCopyInto(out *VultrMachineTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VultrMachineTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineTemplateList.
func (in *VultrMachineTemplateList) DeepCopy() *VultrMachineTemplateList {
	if in == nil {
		return nil
	}
	out := new(VultrMachineTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VultrMachineTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineTemplateResource) DeepCopyInto(out *VultrMachineTemplateResource) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineTemplateResource.
func (in *VultrMachineTemplateResource) DeepCopy() *VultrMachineTemplateResource {
	if in == nil {
		return nil
	}
	out := new(VultrMachineTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrMachineTemplateSpec) DeepCopyInto(out *VultrMachineTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrMachineTemplateSpec.
func (in *VultrMachineTemplateSpec) DeepCopy() *VultrMachineTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(VultrMachineTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrNetworkResource) DeepCopyInto(out *VultrNetworkResource) {
	*out = *in
	out.APIServerLoadbalancersRef = in.APIServerLoadbalancersRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrNetworkResource.
func (in *VultrNetworkResource) DeepCopy() *VultrNetworkResource {
	if in == nil {
		return nil
	}
	out := new(VultrNetworkResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VultrResourceReference) DeepCopyInto(out *VultrResourceReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VultrResourceReference.
func (in *VultrResourceReference) DeepCopy() *VultrResourceReference {
	if in == nil {
		return nil
	}
	out := new(VultrResourceReference)
	in.DeepCopyInto(out)
	return out
}