#!/bin/bash

#CAPV  example configuration
export KUBERNETES_VERSION=v1.30.0+1
export VULTR_REGION=ewr
export CLUSTER_NAME=another-capv-test-cluster
export CONTROL_PLANE_MACHINE_COUNT=1
export VULTR_CONTROL_PLANE_OS=1743
export VULTR_CONTROL_PLANE_MACHINE_PLAN_TYPE=voc-c-2c-4gb-75s-amd
export WORKER_MACHINE_COUNT=1
export VULTR_NODE_MACHINE_PLAN_TYPE=voc-c-2c-4gb-75s-amd
export VULTR_NODE_MACHINE_OS=1743