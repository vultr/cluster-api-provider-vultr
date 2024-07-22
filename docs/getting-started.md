# Getting started

## Prerequisites

- A [vultr][vultr] Account
- Install [clusterctl][clusterctl]
- Install [kubectl][kubectl]
- Install [kustomize][kustomize] `v3.1.0+`
- [Packer][Packer] and [Ansible][Ansible] to build images
- Make to use `Makefile` targets
- A management cluster. You can use either a VM, container or existing Kubernetes cluster as management cluster.
   - If you want to use a VM, install [Minikube][Minikube] version 0.30.0 or greater. You'll also need to install the [Minikube driver][Minikube Driver]. For Linux, we recommend `kvm2`. For MacOS, we recommend `VirtualBox`.
   - If you want to use a container you'll need to install [Kind][kind].
   - If you want to use an existing Kubernetes cluster you'll need to prepare a kubeconfig for the cluster you intend to use.
- Install [vultr-cli][https://github.com/vultr/vultr-cli] (optional)

## Setup Environment

```bash
# Export the Vultr API Key
$ export VULTR_API_KEY=yourapikey
```

## Create SSH-Key
```
$ vultr-cli ssh create --name="cluster-api-key" --key="ssh-rsa AAAAB3NzaC1yc...."

```



## Building images

Clone the image builder repository if you haven't already:

    $ git clone https://github.com/kubernetes-sigs/image-builder.git

Change directory to images/capi within the image builder repository:

    $ cd image-builder/images/capi/packer/vultr 

Generate a Vultr image (choosing Ubuntu in the example below):

    $ make build-vultr-ubuntu-2204

Verify that the image is available in your account and remember the corresponding image ID:

    $  vultr-cli snapshot list


## Initialize the management cluster

```bash
# Initialize a management cluster 
$ clusterctl init 
```

The output will be similar to this:

```bash
Fetching providers
Installing cert-manager Version="v1.15.1"
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v1.7.4" TargetNamespace="capi-system"
Installing Provider="bootstrap-kubeadm" Version="v1.7.4" TargetNamespace="capi-kubeadm-bootstrap-system"
Installing Provider="control-plane-kubeadm" Version="v1.7.4" TargetNamespace="capi-kubeadm-control-plane-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl generate cluster [name] --kubernetes-version [version] | kubectl apply -f -

```

## Creating a workload cluster


Update value of image field below to your controller image URL in
```
../default/manager_image_patch.yaml
```
Add your VULTR_API_KEY to
```

../defaults/credentials.yaml
```

Setting up environment variables: Config example can be found in scripts/capvultr-config-example.sh 

```bash
$ export CLUSTER_NAME=<clustername>
$ export KUBERNETES_VERSION=v1.28.9
$ export CONTROL_PLANE_MACHINE_COUNT=1
$ export CONTROL_PLANE_PLANID=<plan_id>
$ export WORKER_MACHINE_COUNT=1
$ export WORKER_PLANID=<plan_id>
$ export MACHINE_IMAGE=<snapshot_id> # created in the step above.
$ export REGION=<region>
$ export PLANID=<plan_id>
$ export VPCID=<vpc_id> #VPC2ID Optional
$ export SSHKEY_ID=<sshKey_id>
```

```
chmod +x scripts/capvultr-config-example.sh 
source scripts/capvultr-config-example.sh 
```

Create the workload cluster on the management cluster:

---
clusterctl to generate the cluster definition
```
clusterctl generate cluster capi-test-cluster --from templates/cluster-template.yaml > cluster.yaml
```

```bash
$ kubectl apply -f cluster.yaml
 
cluster.cluster.x-k8s.io/capi-test-cluster created
vultrcluster.infrastructure.cluster.x-k8s.io/capi-test-cluster created
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/capi-test-cluster-control-plane created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capi-test-cluster-control-plane created
machinedeployment.cluster.x-k8s.io/capi-test-cluster-md-0 created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capi-test-cluster-md-0 created
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0 created

```

You can see the workload cluster resources by using:

```bash
$ kubectl get cluster-api

NAME                                                                             CLUSTER             AGE
kubeadmconfig.bootstrap.cluster.x-k8s.io/capi-test-cluster-control-plane-lq9gg   capi-test-cluster   20m
kubeadmconfig.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0-75lh2-mnqrp      capi-test-cluster   22m

NAME                                                                      AGE
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0   22m

NAME                                         CLUSTERCLASS   PHASE         AGE   VERSION
cluster.cluster.x-k8s.io/capi-test-cluster                  Provisioned   22m   

NAME                                                        CLUSTER             REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE       AGE   VERSION
machinedeployment.cluster.x-k8s.io/capi-test-cluster-md-0   capi-test-cluster   1                  1         1             ScalingUp   22m   v1.28.9

NAME                                                             CLUSTER             NODENAME   PROVIDERID                                     PHASE         AGE   VERSION
machine.cluster.x-k8s.io/capi-test-cluster-control-plane-lq9gg   capi-test-cluster              vultr://720af6bc-e6f4-40e4-8292-d59abc8fd591   Provisioned   20m   v1.28.9
machine.cluster.x-k8s.io/capi-test-cluster-md-0-75lh2-mnqrp      capi-test-cluster              vultr://cf607d61-1331-4955-97f9-f362fed84ed9   Provisioned   22m   v1.28.9

NAME                                                       CLUSTER             REPLICAS   READY   AVAILABLE   AGE   VERSION
machineset.cluster.x-k8s.io/capi-test-cluster-md-0-75lh2   capi-test-cluster   1                              22m   v1.28.9

NAME                                                                                CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/capi-test-cluster-control-plane   capi-test-cluster   true                                 1                  1         1             22m   v1.28.9

NAME                                                             CLUSTER             READY
vultrcluster.infrastructure.cluster.x-k8s.io/capi-test-cluster   capi-test-cluster   true

NAME                                                                                 CLUSTER             STATE    READY   INSTANCEID                                     MACHINE
vultrmachine.infrastructure.cluster.x-k8s.io/capi-test-cluster-control-plane-lq9gg   capi-test-cluster   active   true    vultr://720af6bc-e6f4-40e4-8292-d59abc8fd591   capi-test-cluster-control-plane-lq9gg
vultrmachine.infrastructure.cluster.x-k8s.io/capi-test-cluster-md-0-75lh2-mnqrp      capi-test-cluster   active   true    vultr://cf607d61-1331-4955-97f9-f362fed84ed9   capi-test-cluster-md-0-75lh2-mnqrp

NAME                                                                                   AGE
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capi-test-cluster-control-plane   22m
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capi-test-cluster-md-0            22m
```

> Note: The control planes won’t be ready until you install the CNI and Vultr Cloud Controller Manager.

To verify that the first control plane is up, use:

```bash
$ kubectl get kubeadmcontrolplane

NAME                              CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION
capi-test-cluster-control-plane   capi-test-cluster   true                                 1                  1         1             20m   v1.28.9

```

After the first control plane node has the `initialized` status, you can retrieve the workload cluster's Kubeconfig:

```bash
$ clusterctl get kubeconfig capi-test-cluster > capvultr-cluster.kubeconfig
```

You can verify what kubernetes nodes exist in the workload cluster by using:

```bash
$ KUBECONFIG=capvultr-cluster.kubeconfig kubectl get node
NAME                                    STATUS     ROLES           AGE   VERSION
capi-test-cluster-control-plane-jsvrz   NotReady   control-plane   20m   v1.28.9
capi-test-cluster-md-0-b54j9-2szdn      NotReady   <none>          14m   v1.28.9
capi-test-cluster-md-0-b54j9-vb5tz      NotReady   <none>          14m   v1.28.9
```

### Deploy CNI
Cilium is used here as an example.


```bash

# SSH into the newly created controlplane on your Vultr Account and run the following
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}


```
Install Cilium
```
cilium install --version 1.15.7
```

### Deploy Vultr CCM and CSI

```bash
# Create Vultr secret
$ KUBECONFIG=capvultr-cluster.kubeconfig kubectl create secret generic vultr-ccm --namespace kube-system --from-literal api-key=$VULTR_API_KEY

# Deploy Vultr Cloud Controller Manager
$ KUBECONFIG=capvultr-cluster.kubeconfig kubectl apply -f https://raw.githubusercontent.com/vultr/vultr-cloud-controller-manager/master/docs/releases/latest.yml

```

After the [CNI](https://github.com/containernetworking/cni) and the [CCM](https://github.com/vultr/vultr-cloud-controller-manager) have deployed your workload cluster nodes should be in the `ready` state. You can verify this by using:

```bash
$ KUBECONFIG=capvultr-cluster.kubeconfig kubectl get node

NAME                                    STATUS   ROLES           AGE     VERSION
capi-test-cluster-control-plane-jsvrz   Ready    control-plane   51m     v1.28.9
capi-test-cluster-md-0-b54j9-cw5jh      Ready    <none>          106s    v1.28.9
capi-test-cluster-md-0-b54j9-nvv2c      Ready    <none>          8m17s   v1.28.9

```

On the Mangement Cluster you should see the following:

```
$ k get cluster-api     
                                                                                                                                                   [0/1144]
NAME                                                                             CLUSTER             AGE                                                                                                                                   
kubeadmconfig.bootstrap.cluster.x-k8s.io/capi-test-cluster-control-plane-jsvrz   capi-test-cluster   56m                                                                                                                                   
kubeadmconfig.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-cw5jh      capi-test-cluster   6m38s                                                                                                                                 
kubeadmconfig.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-nvv2c      capi-test-cluster   13m                                                                                                                                   

NAME                                                                      AGE                                                                                                                                                              
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/capi-test-cluster-md-0   59m                                                                                                                                                              

NAME                                         CLUSTERCLASS   PHASE         AGE   VERSION                                                                                                                                                    
cluster.cluster.x-k8s.io/capi-test-cluster                  Provisioned   59m                                                                                                                                                              

NAME                                                        CLUSTER             REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE     AGE   VERSION                                                                                         
machinedeployment.cluster.x-k8s.io/capi-test-cluster-md-0   capi-test-cluster   2          2       2         0             Running   59m   v1.28.9                                                                                         

NAME                                                             CLUSTER             NODENAME                                PROVIDERID                                     PHASE     AGE     VERSION                                      
machine.cluster.x-k8s.io/capi-test-cluster-control-plane-jsvrz   capi-test-cluster   capi-test-cluster-control-plane-jsvrz   vultr://a657b222-4215-498a-9c7a-6886c0e1c397   Running   56m     v1.28.9                                      
machine.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-cw5jh      capi-test-cluster   capi-test-cluster-md-0-b54j9-cw5jh      vultr://4c6c4b0b-9801-4b8f-8de5-e37959f33aba   Running   6m38s   v1.28.9                                      
machine.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-nvv2c      capi-test-cluster   capi-test-cluster-md-0-b54j9-nvv2c      vultr://ccec9632-b7e3-482b-bfa6-3ba59a2e39d0   Running   13m     v1.28.9                                      

NAME                                                       CLUSTER             REPLICAS   READY   AVAILABLE   AGE   VERSION                                                                                                                
machineset.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9   capi-test-cluster   2          2       2           59m   v1.28.9                                                                                                                

NAME                                                                                CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION                                      
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/capi-test-cluster-control-plane   capi-test-cluster   true          true                   1          1       1         0             59m   v1.28.9                                      

NAME                                                             CLUSTER             READY                                                                                                                                                 
vultrcluster.infrastructure.cluster.x-k8s.io/capi-test-cluster   capi-test-cluster   true                                                                                                                                                  

NAME                                                                                 CLUSTER             STATE    READY   INSTANCEID                                     MACHINE                                                           
vultrmachine.infrastructure.cluster.x-k8s.io/capi-test-cluster-control-plane-jsvrz   capi-test-cluster   active   true    vultr://a657b222-4215-498a-9c7a-6886c0e1c397   capi-test-cluster-control-plane-jsvrz                             
vultrmachine.infrastructure.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-cw5jh      capi-test-cluster   active   true    vultr://4c6c4b0b-9801-4b8f-8de5-e37959f33aba   capi-test-cluster-md-0-b54j9-cw5jh                                
vultrmachine.infrastructure.cluster.x-k8s.io/capi-test-cluster-md-0-b54j9-nvv2c      capi-test-cluster   active   true    vultr://ccec9632-b7e3-482b-bfa6-3ba59a2e39d0   capi-test-cluster-md-0-b54j9-nvv2c
```

```
 clusterctl describe cluster capi-test-cluster
NAME                                                                              READY  SEVERITY  REASON  SINCE  MESSAGE                                                                    
Cluster/capi-test-cluster                                                         True                     53m                                                                                
├─ClusterInfrastructure - VultrCluster/capi-test-cluster                                                                                                                                      
├─ControlPlane - KubeadmControlPlane/capi-test-cluster-control-plane              True                     53m                                                                                
│ └─Machine/capi-test-cluster-control-plane-jsvrz                                 True                     59m                                                                                
│   └─MachineInfrastructure - VultrMachine/capi-test-cluster-control-plane-jsvrz                                                                                                              
└─Workers                                                                                                                                                                                     
  └─MachineDeployment/capi-test-cluster-md-0                                      True                     3m51s                                                                              
    └─2 Machines...                                                               True                     16m    See capi-test-cluster-md-0-b54j9-cw5jh, capi-test-cluster-md-0-b54j9-nvv2c 
```


## Deleting a workload cluster

You can delete the workload cluster from the management cluster using:

```bash
$ kubectl delete cluster capi-test-cluster
```

<!-- References -->
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kustomize]: https://github.com/kubernetes-sigs/kustomize/releases
[kind]: https://github.com/kubernetes-sigs/kind#installation-and-usage
[Minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/
[Minikube Driver]: https://minikube.sigs.k8s.io/docs/drivers
[Packer]: https://www.packer.io/intro/getting-started/install.html
[Ansible]: https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html
[vultr]: https://cloud.vultr.com/
[clusterctl]: https://github.com/kubernetes-sigs/cluster-api/releases
[CNI]: https://github.com/containernetworking/cni
[CCM]: https://github.com/vultr/vultr-cloud-controller-manager
[Vultr Docs]: https://docs.vultr.com/
