# Kubernetes Cloud Controller Manager for Hetzner Cloud and Hetzner Dedicated
This controller is based on [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager), but also support [Hetzner dedicated servers](https://www.hetzner.com/dedicated-rootserver).

## Features
 * adds the following labels to nodes `beta.kubernetes.io/instance-type`, `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`, `node.kubernetes.io/instance-type`, `topology.kubernetes.io/region`, `topology.kubernetes.io/zone`
 * adds the label `node.hetzner.com/type`, which indicates the type of node (cloud or dedicated)
 * copies labels from cloud servers to labels of k8s nodes, see section [Copying Labels from Cloud Nodes](#copying-labels-from-cloud-nodes)
 * sets the external ipv4 address to node status.addresses
 * deletes nodes from Kubernetes that were deleted from the Hetzner Cloud or from Hetzner Robot (panel manager for dedicated servers)
 * exclude the removal of nodes that belong to other providers (kubelet on these nodes should be run WITHOUT the `--cloud-provider=external` option). See section [Exclude nodes](#exclude-nodes)

You can find more information about the cloud controller manager in the [kuberentes documentation](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

**Note:** In the robot panel, dedicated servers, you need to assign a name that matches the name of the node in the k8s cluster. This is necessary so that the controller can detect the server through the API during the initial initialization of the node. After initialization, the controller will use the server ID.

**Note:** Unlike the original [controller]((https://github.com/hetznercloud/hcloud-cloud-controller-manager)), this controller does not support [Hetzner cloud networks](https://community.hetzner.com/tutorials/hcloud-networks-basic), because using them it is impossible to build a network between the cloud and dedicated servers. For a network in your cluster consisting of dedicated and cloud servers, you should  use some kind of **cni** plugin, example [kube-router](https://github.com/cloudnativelabs/kube-router) with option `--enable-overlay`. If you need [Hetzner cloud networks](https://community.hetzner.com/tutorials/hcloud-networks-basic) then you should use the original [controller](https://github.com/hetznercloud/hcloud-cloud-controller-manager) and refuse to use dedicated servers in your cluster.

# Table of Contents
 * [Example](#example)
 * [Version matrix](#version-matrix)
 * [Deployment](#deployment)
   - [Initializing existing nodes in a cluster](#initializing-existing-nodes-in-a-cluster)
   - [Exclude nodes](#exclude-nodes)
 * [Evnironment variables](#evnironment-variables)
 * [Copying Labels from Cloud Nodes](#copying-labels-from-cloud-nodes)
 * [License](#license)

# Example
```bash
NAME               STATUS   ROLES    AGE   VERSION    INSTANCE-TYPE   REGION   ZONE       TYPE
kube-master121-1   Ready    master   54m   v1.16.15   cx31            hel1     hel1-dc2   cloud
kube-worker121-1   Ready    <none>   53m   v1.16.15   cx31            hel1     hel1-dc2   cloud
kube-worker121-2   Ready    <none>   76s   v1.16.15   AX41-NVMe       hel1     hel1-dc4   dedicated

NAME               STATUS   ROLES    AGE    VERSION    INTERNAL-IP   EXTERNAL-IP      OS-IMAGE             KERNEL-VERSION     CONTAINER-RUNTIME
kube-master121-1   Ready    master   54m    v1.16.15   <none>        95.216.202.134   Ubuntu 20.04.2 LTS   5.4.0-65-generic   containerd://1.2.13
kube-worker121-1   Ready    <none>   53m    v1.16.15   <none>        95.217.128.128   Ubuntu 20.04.2 LTS   5.4.0-65-generic   containerd://1.2.13
kube-worker121-2   Ready    <none>   102s   v1.16.15   <none>        135.181.4.26     Ubuntu 20.04.1 LTS   5.4.0-65-generic   containerd://1.2.13
```

Dedicated server:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    io.cilium.network.ipv4-cilium-host: 10.245.2.195
    io.cilium.network.ipv4-health-ip: 10.245.2.15
    io.cilium.network.ipv4-pod-cidr: 10.245.2.0/24
    kubeadm.alpha.kubernetes.io/cri-socket: /run/containerd/containerd.sock
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: "2021-03-08T12:32:24Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: AX41-NVMe # <-- server product
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: hel1 # <-- location
    failure-domain.beta.kubernetes.io/zone: hel1-dc4 # <-- datacenter
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker121-2
    kubernetes.io/os: linux
    node.hetzner.com/type: dedicated # <-- hetzner node type (cloud or dedicated)
  name: kube-worker121-2
  resourceVersion: "3930"
  uid: 19a6c528-ac02-4f42-bb19-ee701f43ca6d
spec:
  podCIDR: 10.245.2.0/24
  podCIDRs:
  - 10.245.2.0/24
  providerID: hetzner://1281541 # <-- Server ID
status:
  addresses:
  - address: kube-worker121-2
    type: Hostname
  - address: 111.233.1.99 # <-- public ipv4
    type: ExternalIP
  allocatable:
    cpu: "12"
    ephemeral-storage: "450673989296"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 65776840Ki
    pods: "110"
  capacity:
    cpu: "12"
    ephemeral-storage: 489012576Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 65879240Ki
    pods: "110"
```

Cloud server:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    io.cilium.network.ipv4-cilium-host: 10.245.1.93
    io.cilium.network.ipv4-health-ip: 10.245.1.141
    io.cilium.network.ipv4-pod-cidr: 10.245.1.0/24
    kubeadm.alpha.kubernetes.io/cri-socket: /run/containerd/containerd.sock
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: "2021-03-08T12:16:59Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: cx31 # <-- server type
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: hel1 # <-- location
    failure-domain.beta.kubernetes.io/zone: hel1-dc2 # <-- datacenter
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker121-1
    kubernetes.io/os: linux
    node.hetzner.com/type: cloud # <-- hetzner node type (cloud or dedicated)
  name: kube-worker121-1
  resourceVersion: "4449"
  uid: f873c208-6403-4ed5-a030-ca92a8a0d48c
spec:
  podCIDR: 10.245.1.0/24
  podCIDRs:
  - 10.245.1.0/24
  providerID: hetzner://10193451 # <-- Server ID
status:
  addresses:
  - address: kube-worker121-1
    type: Hostname
  - address: 95.131.234.167 # <-- public ipv4
    type: ExternalIP
  allocatable:
    cpu: "2"
    ephemeral-storage: "72456848060"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 7856260Ki
    pods: "110"
  capacity:
    cpu: "2"
    ephemeral-storage: 78620712Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 7958660Ki
    pods: "110"
```

# Version matrix
| Kubernetes    | cloud controller | Deployment File |
| ------------- | -----:| ------------------------------------------------------------------------------------------------------:|
| 1.20.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml      |
| 1.19.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.18.x/deploy/deploy.yaml      |
| 1.18.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.18.x/deploy/deploy.yaml      |
| 1.17.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml      |
| 1.16.x          | v0.0.6-k8s-v1.16.x | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.16.x/deploy/deploy.yaml      |

# Deployment
You need to create a token to access the API Hetzner Cloud and API Hetzner Robot. To do this, follow the instructions below:
 * https://docs.hetzner.cloud/#overview-getting-started
 * https://robot.your-server.de/doc/webservice/en.html#preface

 After receiving the token and accesses, create a file with secrets `hetzner-cloud-controller-manager-secret.yaml`):
 ```yaml
 apiVersion: v1
kind: Secret
metadata:
  name: hetzner-cloud-controller-manager
  namespace: kube-system
stringData:
  robot_password: XRmL7hjAMU3RVsXJ4qLpCExiYpcKFJKzMKCiPjzQpJ33RP3b5uHY5DhqhF44YarY #robot password
  robot_user: '#as+BVacIALV' # robot user
  token: pYMfn43zP42ET6N2GtoWX35CUGspyfDA2zbbP57atHKpsFm7YUKbAdcTXFvSyu # hcloud token
 ```

And apply it:
```bash
kubectl apply -f hetzner-cloud-controller-manager-secret.yaml
```
Or do the same with a single line command:
```bash
kubectl create secret generic hetzner-cloud-controller-manager --from-literal=token=pYMfn43zP42ET6N2GtoWX35CUGspyfDA2zbbP57atHKpsFm7YUKbAdcTXFvSyu --from-literal=robot_user='#as+BVacIALV' --from-literal=robot_password=XRmL7hjAMU3RVsXJ4qLpCExiYpcKFJKzMKCiPjzQpJ33RP3b5uHY5DhqhF44YarY
```

Deployment controller:
```bash
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml
```

Now adding new nodes to the cluster, run **kubelet** on them with the parameter: `--cloud-provider=external`. To do this, you can create a file: `/etc/systemd/system/kubelet.service.d/20-external-cloud.conf` with the following contents:
```
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
```
And reload config systemd:
 ```bash
 systemctl daemon-reload
 ```

Next, add the node as usual. For example, if it is **kubeadm**, then:
```bash
kubeadm join kube-api-server:6443 --token token --discovery-token-ca-cert-hash sha256:hash 
```

## Initializing existing nodes in a cluster
If you already had nodes in the cluster before the controller deployment, then you can reinitialize them. To do this, just run **kubelet** on them with the option `--cloud-provider=external` and then manually add **taint** with the key` node.cloudprovider.kubernetes.io/uninitialized` and the effect of `NoSchedule`.


```bash
ssh kube-node-example1
echo '[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
' > /etc/systemd/system/kubelet.service.d/20-external-cloud.conf
systemctl daemon-reload
systemctl restart kubelet
```

Then add **taint** to this node. If there are no **taints** on the node, then do:
```bash
kubectl patch node kube-node-example1 --type='json' -p='[{"op":"add","path":"/spec/taints","value": [{"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized"}]}]'
```
If the node already has some **taints** then:
```bash
kubectl patch node kube-node-example1 --type='json' -p='[{"op":"add","path":"/spec/taints/-","value": {"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized"}}]'
```

The controller will detect this **taint**, initialize the node, and delete **taint**.

## Exclude nodes
If you want to add nodes to your cluster from other cloud/bare-metal providers. Then you may run into a problem - the nodes will be deleted from the cluster. This can be circumvented by excluding this servers using the configuration in JSON format. To do this, you just need to add the **hetzner-cloud-controller-manager** secret to the `cloud_token` key listing the servers that you want to exclude. And add this file to the `--cloud-config` option in the deployment. Regular expressions are supported. For instance:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: hetzner-cloud-controller-manager
  namespace: kube-system
stringData:
  cloud_config: '{"exclude_servers": ["kube-worker102-201","kube-worker102-10.*"]}' # exclude servers
  robot_password: XRmL7hjAMU3RVsXJ4qLpCExiYpcKFJKzMKCiPjzQpJ33RP3b5uHY5DhqhF44YarY  # robot password
  robot_user: '#as+BVacIALV'                                                        # robot user
  token: pYMfn43zP42ET6N2GtoWX35CUGspyfDA2zbbP57atHKpsFm7YUKbAdcTXFvSyu             # hcloud token
```

It is very important to run kubelet on such servers WITHOUT the `--cloud-provider=external` option.

For deployment with exclude servers, a separate file is provided:
```bash
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy-exclude.yaml
```

# Evnironment variables

 * `HCLOUD_TOKEN` (**required**) - Hcloud API access token
 * `HCLOUD_ENDPOINT`(default: `https://api.hetzner.cloud/v1`) - endpoint for Hcloud API
 * `NODE_NAME` (**required**)  - name of the node on which the application is running (spec.nodeName)
 * `HROBOT_USER` (**required**) - user to access the Hrobot API
 * `HROBOT_PASS` (**required**) - password to access the Hrobot API
 * `HROBOT_PERIOD`(default: `180`) - period in seconds with which the Hrobot API will be polled
 * `PROVIDER_NAME` (default: `hetzner`) - the name of the provider to be used in the prefix of the node specification (`spec.providerID`)
 * `NAME_LABEL_TYPE` (default: `node.hetzner.com/type`) - the name of the label in which information about the type of node will be stored (cloud or dedicted)
 * `NAME_CLOUD_NODE` (default: `cloud`) - the name of the cloud node in the meaning of the label `NAME_LABEL_TYPE`
 * `NAME_DEDICATED_NODE` (default: `dedicated`) - the name of the dedicated node in the meaning of the label `NAME_LABEL_TYPE`
 * `ENABLE_SYNC_LABELS` (default: `true`) - enables/disables copying labels from cloud servers

Hrobot get server have limit [200 requests per hour](https://robot.your-server.de/doc/webservice/en.html#get-server). Therefore, the application receives this information with the specified period (`HROBOT_PERIOD`) and saves it in memory. One poll every 180 seconds means 20 queries per hour.

# Copying Labels from Cloud Nodes
Cloud servers can be tagged
```bash
$ hcloud server list -o columns=id,name,labels
NAME                         LABELS
kube-master121-1             myLabel=myValue
kube-worker121-1             myLabel=myValue
```
The controller copies these labels to the labels of the k8s node:
```bash
$ kubectl get node -L myLabel -L node.hetzner.com/type
NAME               STATUS   ROLES                  AGE   VERSION   MYLABEL   TYPE
kube-master121-1   Ready    master                 36m   v1.16.15   myValue   cloud
kube-worker121-1   Ready    <none>                 35m   v1.16.15   myValue   cloud
kube-worker121-2   Ready    <none>                 20m   v1.16.15             dedicated
```
This behavior can be disabled by setting the environment variable `ENABLE_SYNC_LABELS = false`.

Changing a label also changes that label in the cluster. Removing a label from the cloud server will also remove it from the k8s node:
```bash
$ hcloud server remove-label kube-worker121-1 myLabel
$ hcloud server list -o columns=name,labels
NAME                         LABELS   
kube-master121-1             myLabel=myValue
kube-worker121-1             
$ sleep 300
$ kubectl get node -L myLabel -L node.hetzner.com/type
NAME               STATUS   ROLES                  AGE   VERSION   MYLABEL   TYPE
kube-master121-1   Ready    master                 37m   v1.16.15   myValue   cloud
kube-worker121-1   Ready    <none>                 37m   v1.16.15             cloud
kube-worker121-2   Ready    <none>                 21m   v1.16.15             dedicated
```

Synchronization does not occur instantly, but with an interval of 5 minutes. This can be changed via the `--node-status-update-frequency` argument. But be careful, there is a limit on the number of requests in the hetzner API. I would not recommend changing this parameter.

# License

Apache License, Version 2.0
