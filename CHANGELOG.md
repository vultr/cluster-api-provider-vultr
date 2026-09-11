# Changelog
## [v0.5.0](https://github.com/vultr/cluster-api-provider-vultr/compare/v0.4.0...v0.5.0) (2026-09-11)
### Breaking Changes
* Migrate the API to `v1beta2` (Cluster API `v1beta2` contract). The `v1beta1` API is removed and there is no conversion webhook, so existing `v1beta1` `VultrCluster`/`VultrMachine` objects are not upgraded in place — recreate clusters or migrate manually. Requires Cluster API v1.11 or newer. [PR 143](https://github.com/vultr/cluster-api-provider-vultr/pull/143)

### Enhancements
* Add ClusterClass and `cluster-template` flavors (`clusterclass-kubeadm`, `cilium`, `vultr-ccm`, `vultr-csi`, `full`) published as release assets [PR 143](https://github.com/vultr/cluster-api-provider-vultr/pull/143)
* Add `cpu`, `ram` and `storage` to `VultrMachine` status [PR 147](https://github.com/vultr/cluster-api-provider-vultr/pull/147)

### Bug Fixes
* Set `kubernetesAPICallSeconds: 300` in the control-plane kubeadm config of all templates so `kubeadm init` waits for the Vultr load balancer instead of timing out after 60s [PR 162](https://github.com/vultr/cluster-api-provider-vultr/pull/162)
* Grant the controller `create`/`patch`/`update` on `events.k8s.io` events; the controllers use the `events.k8s.io` recorder and were denied [PR 162](https://github.com/vultr/cluster-api-provider-vultr/pull/162)

### Automation
* Replace deprecated `gcr.io/kubebuilder/kube-rbac-proxy` image with `registry.k8s.io/kubebuilder/kube-rbac-proxy:v0.16.0` [PR 160](https://github.com/vultr/cluster-api-provider-vultr/pull/160)
* Fix the release workflow so `infrastructure-components.yaml` pins the released image tag; add the `0.5` release series to `metadata.yaml` [PR 162](https://github.com/vultr/cluster-api-provider-vultr/pull/162)

### Dependencies
* Go 1.24 → 1.25; cluster-api v1.10.5 → v1.13.6; controller-runtime v0.20.4 → v0.23.3; k8s.io/* v0.32.3 → v0.35.8 [PR 143](https://github.com/vultr/cluster-api-provider-vultr/pull/143), [PR 163](https://github.com/vultr/cluster-api-provider-vultr/pull/163)
* Update govultr from v3.25.0 to v3.33.0 [PR 163](https://github.com/vultr/cluster-api-provider-vultr/pull/163)
* Update kustomize to v5.8.1, golangci-lint to v2.13.2, kubectl to v1.35.8; pin setup-envtest to v0.25.0 and test against Kubernetes 1.35.0 [PR 163](https://github.com/vultr/cluster-api-provider-vultr/pull/163)

## [v0.4.0](https://github.com/vultr/cluster-api-provider-vultr/compare/v0.3.0...v0.4.0) (2025-12-04)
### Enhancements
* Add VPC ID field to VultrCluster CRDs [PR 118](https://github.com/vultr/cluster-api-provider-vultr/pull/118)
* Add VPC Only field to VultrMachine CRDs [PR 120](https://github.com/vultr/cluster-api-provider-vultr/pull/120)

### Dependencies
* Update govultr from v3.23.0 to v3.25.0 [PR 119](https://github.com/vultr/cluster-api-provider-vultr/pull/119)

### Documentation
* Update metadata.yaml contract [PR 138](https://github.com/vultr/cluster-api-provider-vultr/pull/138)

## [v0.3.0](https://github.com/vultr/cluster-api-provider-vultr/compare/v0.2.1...v0.3.0) (2025-11-03)
### Enhancements
* Add firewall group support to VultrMacheine spec [PR 109](https://github.com/vultr/cluster-api-provider-vultr/pull/109)

### Dependencies
* Update cluster-api to v1.10.5, update controller-gen to v0.17.1, replace deprecated predicates [PR 87](https://github.com/vultr/cluster-api-provider-vultr/pull/87)

## [v0.2.1](https://github.com/vultr/cluster-api-provider-vultr/compare/v0.2.0...v0.2.1) (2025-10-14)

### Automation
* Update Makefile controller-image to include tag [PR 105](https://github.com/vultr/cluster-api-provider-vultr/pull/105)

### Enhancements
* Add firewall rule support to VultrCluster spec [PR 104](https://github.com/vultr/cluster-api-provider-vultr/pull/104)

## [v0.2.0](https://github.com/vultr/cluster-api-provider-vultr/compare/v0.1.0...v0.2.0) (2025-09-29)

### Dependencies
* Bump Go from v1.21 to v1.24 [PR 73](https://github.com/vultr/cluster-api-provider-vultr/pull/73)
* Bump govultr from v3.8.1 to v3.23.0 [PR 76](https://github.com/vultr/cluster-api-provider-vultr/pull/76)
* Bump github.com/onsi/ginkgo/v2 from 2.17.1 to 2.25.2 [PR 81](https://github.com/vultr/cluster-api-provider-vultr/pull/81)
* Bump golang.org/x/oauth2 from 0.18.0 to 0.30.0 [PR 83](https://github.com/vultr/cluster-api-provider-vultr/pull/83)
* Run code generation to update VultrMachine CRDs [PR 86](https://github.com/vultr/cluster-api-provider-vultr/pull/86)

### Automation
* Migrate golangci-lint configuration to v2 and lint fixes [PR 71](https://github.com/vultr/cluster-api-provider-vultr/pull/71)
* Update workflows to use GITHUB_OUTPUT environment variable [PR 68](https://github.com/vultr/cluster-api-provider-vultr/pull/68)
* Update github workflows to use shared-action workflows

### Enhancements
* Support Multi Node Control Planes in the default template [PR 66](https://github.com/vultr/cluster-api-provider-vultr/pull/66)
* Use environment variable for manager credentials secret [PR 88](https://github.com/vultr/cluster-api-provider-vultr/pull/88)
* Add machine-only check in reconcilation logic [PR 70](https://github.com/vultr/cluster-api-provider-vultr/pull/70)

### Documentation
* Add deprecation notice for all VPC2 elements [PR 72](https://github.com/vultr/cluster-api-provider-vultr/pull/72)
* Add License [PR 43](https://github.com/vultr/cluster-api-provider-vultr/pull/43)
* Update Getting Started documentation [PR 85](https://github.com/vultr/cluster-api-provider-vultr/pull/85)

#### New Contributers
* @vrabbi made their first contribution in [PR 66](https://github.com/vultr/cluster-api-provider-vultr/pull/66)
* @huseyinbabal made their first contribution in [PR 70](https://github.com/vultr/cluster-api-provider-vultr/pull/70)

## v0.1.0 (2024-08-21)
* Initial release
