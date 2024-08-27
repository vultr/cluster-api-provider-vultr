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

    $ cd image-builder/images/capi

Generate a Vultr image (choosing Ubuntu in the example below):

    $ make build-vultr-ubuntu-2204


List of available make commands for Vultr
```
../capi$ make help | grep vultr

  deps-vultr                           Installs/checks dependencies for Vultr builds
  build-vultr-ubuntu-2204              Builds Ubuntu 22.04 Vultr Snapshot
  validate-vultr-ubuntu-2204           Validates Ubuntu 22.04 Vultr Snapshot Packer config
```

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


Update value of image field below to your controller image URL in the Vultr provider.
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
$ export KUBERNETES_VERSION=v1.29.7
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
clusterctl generate cluster capvultr-quickstart --from templates/cluster-template.yaml > cluster.yaml
```

Apply the template

```bash
$ kubectl apply -f cluster.yaml
 
cluster.cluster.x-k8s.io/capvultr-quickstart created
vultrcluster.infrastructure.cluster.x-k8s.io/capvultr-quickstart created
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/capvultr-quickstart-control-plane created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capvultr-quickstart-control-plane created
machinedeployment.cluster.x-k8s.io/capvultr-quickstart-md-0 created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/capvultr-quickstart-md-0 created
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/capvultr-quickstart-md-0 created

```

You can see the workload cluster resources by using:

```bash
$ kubectl get cluster-api
```

> Note: The control planes won’t be ready until you install the CNI and Vultr Cloud Controller Manager.

To verify that the first control plane is up, use:

```bash
$ kubectl get kubeadmcontrolplane

NAME                              CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION
capvultr-quickstart-control-plane   capvultr-quickstart   true                                 1                  1         1             20m   v1.28.9

```

After the first control plane node has the `initialized` status, you can retrieve the workload cluster's Kubeconfig:

```bash
$ clusterctl get kubeconfig capvultr-quickstart > capvultr-quickstart.kubeconfig
```

You can verify what kubernetes nodes exist in the workload cluster by using:

```bash
$ KUBECONFIG=capvultr-quickstart.kubeconfig kubectl get node
NAME                                    STATUS     ROLES           AGE   VERSION
capvultr-quickstart-control-plane-jsvrz   NotReady   control-plane   20m   v1.28.9
capvultr-quickstart-md-0-b54j9-2szdn      NotReady   <none>          14m   v1.28.9
capvultr-quickstart-md-0-b54j9-vb5tz      NotReady   <none>          14m   v1.28.9
```

### Deploy CNI

Cilium is used here as an example but you can bring your own CNI.

https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#cilium-quick-installation



### Deploy Vultr CCM and CSI

```bash
# Create Vultr secret
$ KUBECONFIG=capvultr-quickstart.kubeconfig kubectl create secret generic vultr-ccm --namespace kube-system --from-literal api-key=$VULTR_API_KEY

# Deploy Vultr Cloud Controller Manager
$ KUBECONFIG=capvultr-quickstart.kubeconfig kubectl apply -f https://raw.githubusercontent.com/vultr/vultr-cloud-controller-manager/master/docs/releases/latest.yml

```

After the [CNI](https://github.com/containernetworking/cni) and the [CCM](https://github.com/vultr/vultr-cloud-controller-manager) have deployed your workload cluster nodes should be in the `ready` state. You can verify this by using:

```bash
$ KUBECONFIG=capvultr-quickstart.kubeconfig kubectl get node

NAME                                    STATUS   ROLES           AGE     VERSION
capvultr-quickstart-control-plane-jsvrz   Ready    control-plane   51m     v1.28.9
capvultr-quickstart-md-0-b54j9-cw5jh      Ready    <none>          106s    v1.28.9
capvultr-quickstart-md-0-b54j9-nvv2c      Ready    <none>          8m17s   v1.28.9

```

On the Mangement Cluster you should see the following:

```
$ k get cluster-api     
                                                                                                                                                   [0/1144]
NAME                                                                             CLUSTER             AGE                                                                                                                                   
kubeadmconfig.bootstrap.cluster.x-k8s.io/capvultr-quickstart-control-plane-jsvrz   capvultr-quickstart   56m                                                                                                                                   
kubeadmconfig.bootstrap.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-cw5jh      capvultr-quickstart   6m38s                                                                                                                                 
kubeadmconfig.bootstrap.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-nvv2c      capvultr-quickstart   13m                                                                                                                                   

NAME                                                                      AGE                                                                                                                                                              
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/capvultr-quickstart-md-0   59m                                                                                                                                                              

NAME                                         CLUSTERCLASS   PHASE         AGE   VERSION                                                                                                                                                    
cluster.cluster.x-k8s.io/capvultr-quickstart                  Provisioned   59m                                                                                                                                                              

NAME                                                        CLUSTER             REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE     AGE   VERSION                                                                                         
machinedeployment.cluster.x-k8s.io/capvultr-quickstart-md-0   capvultr-quickstart   2          2       2         0             Running   59m   v1.28.9                                                                                         

NAME                                                             CLUSTER             NODENAME                                PROVIDERID                                     PHASE     AGE     VERSION                                      
machine.cluster.x-k8s.io/capvultr-quickstart-control-plane-jsvrz   capvultr-quickstart   capvultr-quickstart-control-plane-jsvrz   vultr://a657b222-4215-498a-9c7a-6886c0e1c397   Running   56m     v1.28.9                                      
machine.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-cw5jh      capvultr-quickstart   capvultr-quickstart-md-0-b54j9-cw5jh      vultr://4c6c4b0b-9801-4b8f-8de5-e37959f33aba   Running   6m38s   v1.28.9                                      
machine.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-nvv2c      capvultr-quickstart   capvultr-quickstart-md-0-b54j9-nvv2c      vultr://ccec9632-b7e3-482b-bfa6-3ba59a2e39d0   Running   13m     v1.28.9                                      

NAME                                                       CLUSTER             REPLICAS   READY   AVAILABLE   AGE   VERSION                                                                                                                
machineset.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9   capvultr-quickstart   2          2       2           59m   v1.28.9                                                                                                                

NAME                                                                                CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION                                      
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/capvultr-quickstart-control-plane   capvultr-quickstart   true          true                   1          1       1         0             59m   v1.28.9                                      

NAME                                                             CLUSTER             READY                                                                                                                                                 
vultrcluster.infrastructure.cluster.x-k8s.io/capvultr-quickstart   capvultr-quickstart   true                                                                                                                                                  

NAME                                                                                 CLUSTER             STATE    READY   INSTANCEID                                     MACHINE                                                           
vultrmachine.infrastructure.cluster.x-k8s.io/capvultr-quickstart-control-plane-jsvrz   capvultr-quickstart   active   true    vultr://a657b222-4215-498a-9c7a-6886c0e1c397   capvultr-quickstart-control-plane-jsvrz                             
vultrmachine.infrastructure.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-cw5jh      capvultr-quickstart   active   true    vultr://4c6c4b0b-9801-4b8f-8de5-e37959f33aba   capvultr-quickstart-md-0-b54j9-cw5jh                                
vultrmachine.infrastructure.cluster.x-k8s.io/capvultr-quickstart-md-0-b54j9-nvv2c      capvultr-quickstart   active   true    vultr://ccec9632-b7e3-482b-bfa6-3ba59a2e39d0   capvultr-quickstart-md-0-b54j9-nvv2c
```

```
 clusterctl describe cluster capvultr-quickstart
NAME                                                                              READY  SEVERITY  REASON  SINCE  MESSAGE                                                                    
Cluster/capvultr-quickstart                                                         True                     53m                                                                                
├─ClusterInfrastructure - VultrCluster/capvultr-quickstart                                                                                                                                      
├─ControlPlane - KubeadmControlPlane/capvultr-quickstart-control-plane              True                     53m                                                                                
│ └─Machine/capvultr-quickstart-control-plane-jsvrz                                 True                     59m                                                                                
│   └─MachineInfrastructure - VultrMachine/capvultr-quickstart-control-plane-jsvrz                                                                                                              
└─Workers                                                                                                                                                                                     
  └─MachineDeployment/capvultr-quickstart-md-0                                      True                     3m51s                                                                              
    └─2 Machines...                                                               True                     16m    See capvultr-quickstart-md-0-b54j9-cw5jh, capvultr-quickstart-md-0-b54j9-nvv2c 
```


## Deleting a workload cluster

You can delete the workload cluster from the management cluster using:

```bash
$ kubectl delete cluster capvultr-quickstart
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
