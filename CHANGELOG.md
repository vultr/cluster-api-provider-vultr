# Changelog
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
