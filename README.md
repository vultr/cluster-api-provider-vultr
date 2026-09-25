# Kubernetes Cluster API Provider Vultr

<p align="center">
<img src="https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/main/docs/book/src/images/introduction.svg"  width="80" style="vertical-align: middle;">
<img src="./docs/vultr.svg" width="100" style="vertical-align: middle;">
</p>
<p align="center">

## What is the Cluster API Provider Vultr (CAPVULTR)

The [Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings declarative Kubernetes-style APIs to cluster creation, configuration and management.

The API itself is shared across multiple cloud providers allowing for true Vultr hybrid deployments of Kubernetes.

## Quick Start

Check out the [Cluster API Quick Start](docs/getting-started.md) to create your first Kubernetes cluster.

## Documentation

- [Getting started](docs/getting-started.md), including the list of cluster template flavors
- [Bare metal nodes](docs/bare-metal.md)
- [Deploying with a custom registry](docs/deployment-with-custom-registry.md)

## Compatibility

### Cluster API Versions

This provider's versions are compatible with the following versions of Cluster API:

| CAPVULTR Version | API Version | CAPI v1.10 | CAPI v1.11 | CAPI v1.12 | CAPI v1.13 |
|------------------|:------------:|:----------:|:----------:|:----------:|:----------:|
| v0.4.0           | v1beta1      |      ✓     |            |            |            |
| v0.5.0           | v1beta2      |            |      ✓     |      ✓     |      ✓     |

## Kubernetes versions with published Images

Pre-built images are pushed to the [Docker Hub](https://hub.docker.com/u/vultr/cluster-api-provider-vultr).