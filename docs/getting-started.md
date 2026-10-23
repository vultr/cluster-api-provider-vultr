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
 export VULTR_API_KEY=yourapikey
```

## Create SSH-Key
```
 vultr-cli ssh create --name="cluster-api-key" --key="ssh-rsa AAAAB3NzaC1yc...."

```



# Building Vultr Images with Image Builder


**Clone the Image Builder repository:**
```bash
git clone https://github.com/kubernetes-sigs/image-builder.git
```
Navigate to the capi images directory:

     cd image-builder/images/capi

Build Vultr image:

     make build-vultr-ubuntu-2204


Available make Commands for Vultr
```
make help | grep vultr

deps-vultr                           Installs/checks dependencies for Vultr builds
build-vultr-ubuntu-2204              Builds Ubuntu 22.04 Vultr Snapshot
validate-vultr-ubuntu-2204           Validates Ubuntu 22.04 Vultr Snapshot Packer config
```

Verify that the image is available in your account and remember the corresponding image ID:

   vultr-cli snapshot list


## Initialize the management cluster

Most templates are ClusterClass based and ship Cilium through the Cluster API Helm addon
provider, so enable the topology feature gate and install the addon provider along with
CAPVULTR:

```bash
export CLUSTER_TOPOLOGY=true
clusterctl init --infrastructure vultr --addon helm
```

The output will be similar to this:

```bash
Fetching providers
Installing cert-manager Version="v1.19.1"
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v1.13.6" TargetNamespace="capi-system"
Installing Provider="bootstrap-kubeadm" Version="v1.13.6" TargetNamespace="capi-kubeadm-bootstrap-system"
Installing Provider="control-plane-kubeadm" Version="v1.13.6" TargetNamespace="capi-kubeadm-control-plane-system"
Installing Provider="infrastructure-vultr" Version="v0.6.0" TargetNamespace="capvultr-system"
Installing Provider="addon-helm" Version="v0.6.4" TargetNamespace="caaph-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl generate cluster [name] --kubernetes-version [version] | kubectl apply -f -

```

CAPVULTR needs your Vultr API key in the management cluster. `clusterctl` reads it from the
environment or from `~/.config/cluster-api/clusterctl.yaml`, so export it before `clusterctl init`:

```bash
export VULTR_API_KEY=<your api key>
```

## Cluster templates

`clusterctl generate cluster --flavor <flavor>` picks a template. All of them except
`cluster-template.yaml` and `bare-metal-standalone` are ClusterClass based, so the management
cluster needs the topology feature gate from the previous step.

| Flavor | Contents |
|---|---|
| *(none)* | plain template: cloud compute control plane and workers, no ClusterClass, no addons |
| `clusterclass-kubeadm` | the same cluster as a ClusterClass topology |
| `cilium` | adds Cilium through the Helm addon provider |
| `vultr-ccm` | adds the Vultr cloud controller manager |
| `vultr-csi` | adds the Vultr CSI driver |
| `full` | Cilium, CCM and CSI together |
| `bare-metal` | cloud compute control plane, bare metal workers, Cilium and CCM |
| `bare-metal-cp` | the same with the control plane on bare metal as well |
| `bare-metal-standalone` | bare metal workers without ClusterClass |

Bare metal has its own guide: [docs/bare-metal.md](bare-metal.md).

The ClusterClass itself is published separately, as `clusterclass-vultr.yaml`,
`clusterclass-vultr-bare-metal.yaml` and `clusterclass-vultr-bare-metal-cp.yaml`. `clusterctl`
adds the right one to the generated output when it is not installed on the management cluster
yet, so you do not normally apply it by hand.

Working from a source checkout rather than a release, build the artifacts and apply the class
yourself, because `clusterctl --from` reads a file and never fetches a missing class:

```bash
make generate-release
clusterctl generate cluster ${CLUSTER_NAME} --from out/clusterclass-vultr.yaml | kubectl apply -f -
clusterctl generate cluster ${CLUSTER_NAME} --from out/cluster-template-full.yaml | kubectl apply -f -
```

## Creating a workload cluster

Set the variables the template needs. `scripts/capvultr-config-example` lists all of them with
placeholders; copy it, fill it in and source it.

```bash
 export CLUSTER_NAME=<clustername>
 export KUBERNETES_VERSION=v1.34.3
 export CONTROL_PLANE_MACHINE_COUNT=1
 export CONTROL_PLANE_PLAN_ID=<plan_id>
 export WORKER_MACHINE_COUNT=1
 export WORKER_PLAN_ID=<plan_id>
 export MACHINE_IMAGE=<snapshot_id> # created in the step above.
 export REGION=<region>
 export SSH_KEY_ID=<sshKey_id>
 export VPC_ID=""   # leave empty for public IPs only; set to put every node in the VPC
```

```
cp scripts/capvultr-config-example my-cluster.env
# fill in the placeholders, then
source my-cluster.env
```

The plan variables have no default, so `clusterctl` stops and names them if any is unset.

Create the workload cluster on the management cluster:

```
clusterctl generate cluster capvultr-quickstart --flavor full > cluster.yaml
```

Apply the template

```bash
kubectl apply -f cluster.yaml

clusterclass.cluster.x-k8s.io/vultr created
kubeadmcontrolplanetemplate.controlplane.cluster.x-k8s.io/vultr-control-plane created
kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/vultr-default-worker created
vultrclustertemplate.infrastructure.cluster.x-k8s.io/vultr-infrastructure created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/vultr-control-plane-machine created
vultrmachinetemplate.infrastructure.cluster.x-k8s.io/vultr-default-worker-machine created
configmap/vultr-capvultr-quickstart-ccm-manifests created
configmap/vultr-capvultr-quickstart-csi-manifests created
secret/vultr-capvultr-quickstart-ccm-secret created
secret/vultr-capvultr-quickstart-csi-secret created
helmchartproxy.addons.cluster.x-k8s.io/capvultr-quickstart-cilium created
clusterresourceset.addons.cluster.x-k8s.io/capvultr-quickstart-ccm-manifests created
clusterresourceset.addons.cluster.x-k8s.io/capvultr-quickstart-ccm-secret created
clusterresourceset.addons.cluster.x-k8s.io/capvultr-quickstart-csi-manifests created
clusterresourceset.addons.cluster.x-k8s.io/capvultr-quickstart-csi-secret created
cluster.cluster.x-k8s.io/capvultr-quickstart created

```

The `ClusterClass` and the `*Template` objects are cluster independent, so a second cluster from
the same flavor only adds its own `Cluster` and addons. The `Cluster` is what the topology
controller expands into the control plane, machine deployments and infrastructure objects.

You can see the workload cluster resources by using:

```bash
kubectl get cluster-api
```

> Note: The control planes won’t be ready until the CNI and the Vultr Cloud Controller Manager
> are installed. The flavors listed above install both automatically; with the plain template
> you install them yourself, as described further down.

To verify that the first control plane is up, use:

```bash
kubectl get kubeadmcontrolplane

NAME                              CLUSTER             INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION
capvultr-quickstart-control-plane   capvultr-quickstart   true                                 1                  1         1             20m   v1.28.9

```

After the first control plane node has the `initialized` status, you can retrieve the workload cluster's Kubeconfig:

```bash
clusterctl get kubeconfig capvultr-quickstart > capvultr-quickstart.kubeconfig
```

You can verify what kubernetes nodes exist in the workload cluster by using:

```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl get node
NAME                                    STATUS     ROLES           AGE   VERSION
capvultr-quickstart-control-plane-jsvrz   NotReady   control-plane   20m   v1.28.9
capvultr-quickstart-md-0-b54j9-2szdn      NotReady   <none>          14m   v1.28.9
capvultr-quickstart-md-0-b54j9-vb5tz      NotReady   <none>          14m   v1.28.9
```

## Installing the CNI, CCM and CSI by hand

The `cilium`, `vultr-ccm`, `vultr-csi`, `full` and the two bare metal flavors install these for
you, so skip this section unless you used the plain template or want to bring your own.

### Deploy CNI

Cilium is used here as an example but you can bring your own CNI.

https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#cilium-quick-installation



### Deploy Vultr CCM and CSI

#### Create Vultr secret
```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl create secret generic vultr-ccm --namespace kube-system --from-literal api-key=$VULTR_API_KEY
```

#### Deploy Vultr Cloud Controller Manager
```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl apply -f https://raw.githubusercontent.com/vultr/vultr-cloud-controller-manager/master/docs/releases/latest.yml

```

#### Deploy Vultr CSI

```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl create secret generic vultr-csi --namespace kube-system --from-literal api-key=$VULTR_API_KEY
```

```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl apply -f https://raw.githubusercontent.com/vultr/vultr-csi/refs/heads/master/docs/releases/latest.yml

```

After the [CNI](https://github.com/containernetworking/cni) and the [CCM](https://github.com/vultr/vultr-cloud-controller-manager) have deployed your workload cluster nodes should be in the `ready` state. You can verify this by using:

```bash
KUBECONFIG=capvultr-quickstart.kubeconfig kubectl get node

NAME                                    STATUS   ROLES           AGE     VERSION
capvultr-quickstart-control-plane-jsvrz   Ready    control-plane   51m     v1.28.9
capvultr-quickstart-md-0-b54j9-cw5jh      Ready    <none>          106s    v1.28.9
capvultr-quickstart-md-0-b54j9-nvv2c      Ready    <none>          8m17s   v1.28.9

```

On the Mangement Cluster you should see the following:

```
k get cluster-api     
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
kubectl delete cluster capvultr-quickstart
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
