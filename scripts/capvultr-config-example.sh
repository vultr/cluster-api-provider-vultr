#!/bin/bash

#CAPVULTR  example configuration
export KUBERNETES_VERSION=v1.29.7
export CLUSTER_NAME="capi-test-cluster"
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=1
export MACHINE_IMAGE="beb9ffb1-33bf-42b3-aac0-83a37858e0a9"
export REGION="ewr"
export CONTROL_PLANE_PLANID="voc-c-2c-4gb-75s-amd"
export WORKER_PLANID="vc2-8c-32gb"
export VPCID="63692dfd-bea6-4e4e-8bb7-50de31887158"
export SSHKEY_ID="c9db76ee-7b7a-43zz-a9fb-b4c772acdd41"

#optional
#export VPC2ID=<vpc2_id> 