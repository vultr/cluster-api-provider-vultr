# Bare metal nodes

Nodes can run on [Vultr Bare Metal](https://www.vultr.com/products/bare-metal/) servers through
the `VultrBareMetalMachine` kind. Two flavors use it, and they differ only in where the control
plane runs:

| Flavor | Control plane nodes | Worker nodes |
|---|---|---|
| `bare-metal` | cloud compute | bare metal |
| `bare-metal-cp` | bare metal | bare metal |

The bare metal workers are the pool `bm-0`. Both also define a cloud compute worker pool `md-0`, at
0 replicas unless you set `WORKER_MACHINE_COUNT`.

`bare-metal-standalone` is the same layout as `bare-metal` without ClusterClass.

## Creating a cluster

Both ClusterClass flavors need `CLUSTER_TOPOLOGY=true` and `clusterctl init --addon helm`, since
they ship Cilium and the Vultr CCM.

```sh
source scripts/capvultr-config-example      # after filling in the placeholders
clusterctl generate cluster ${CLUSTER_NAME} --flavor bare-metal | kubectl apply -f -
```

`BARE_METAL_PLAN_ID` sets the worker plan, `BARE_METAL_CONTROL_PLANE_PLAN_ID` the control plane
plan for `bare-metal-cp`. Neither has a default, so `clusterctl` names them if they are unset.

Expect six to eight minutes per server. Control planes come up one at a time, so three of them
take about half an hour.

## Choosing a plan

Plans and their regions come from `GET /v2/plans-metal`. Bare metal is in far fewer regions than
cloud compute, and a plan can be out of stock in a region the API lists, which fails the machine
with `ServerCreationFailed`.

VPC support is per plan. Verified with Ubuntu 24.04 on `vbm-4c-32gb` (no VPC) and on
`vbm-6c-32gb-amd`, `vbm-8c-132gb` and `vbm-24c-256gb-amd` (VPC capable).

## Networking

`VPC_ID` applies to the whole cluster. Set it and every node, cloud and bare metal, joins that VPC
and the CNI uses the private addresses. Leave it empty and every node uses its public IP.

Prefer a VPC, so etcd and the kubelet listen privately. Requesting one on a plan without support
fails the machine with `VPCAttachFailed`. Without a VPC, restrict the cluster ports with Vultr
firewall groups.

## Bootstrap

Kubernetes is installed at first boot rather than baked into an image. The bootstrap runs
`/usr/local/bin/install-k8s.sh`, which covers Ubuntu 24.04 only, disables the `ufw` default that
would block cluster traffic, and labels the node `vultr.com/baremetal=true` so the
[Vultr CCM](https://github.com/vultr/vultr-cloud-controller-manager) looks it up through the bare
metal API. Replace the bootstrap config and all three are yours to reproduce; without the label
the CCM deletes the node as a missing cloud instance.

## Checking on a machine

```sh
kubectl get vbm            # short for vultrbaremetalmachines
kubectl describe vbm <name>
```

One `Ready` condition, mirrored into the Machine's `InfrastructureReady`. The reason is the phase:

| Reason | Meaning |
|---|---|
| `WaitingForClusterInfrastructure` | load balancer and VPC not ready |
| `WaitingForBootstrapData` | no cloud-init data yet |
| `ServerPending` | provisioning at Vultr |
| `VPCAttachRequested` | attach requested, not visible yet |
| `WaitingForLoadBalancer` | control plane waiting to become a backend |
| `ServerActive` | ready |
| `Deleting` | being deleted |

`ServerCreationFailed`, `ServerNotFound`, `UnexpectedStatus` and `VPCAttachFailed` are terminal.
The controller stops and does not recreate the server; replace the Machine instead. A `Paused`
condition appears when the Cluster is paused or the machine has the `cluster.x-k8s.io/paused`
annotation.
