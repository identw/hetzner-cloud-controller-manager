# Kubernetes Cloud Controller Manager for Hetzner Cloud and Hetzner Dedicated
This controller is based on [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager), but also support [Hetzner dedicated servers](https://www.hetzner.com/dedicated-rootserver).

## Features
 * adds the following labels to nodes `beta.kubernetes.io/instance-type`, `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`, `node.kubernetes.io/instance-type`, `topology.kubernetes.io/region`, `topology.kubernetes.io/zone`
 * sets the external ipv4 address to node status.addresses
 * deletes nodes from Kubernetes that were deleted from the Hetzner Cloud or from Hetzner Robot (panel manager for dedicated servers)
 * exclude the removal of nodes that belong to other providers (kubelet on these nodes should be run WITHOUT the `--cloud-provider=external` option). See section [Exclude nodes](#exclude-nodes)

You can find more information about the cloud controller manager in the [kuberentes documentation](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

**Note:** Unlike the original [controller]((https://github.com/hetznercloud/hcloud-cloud-controller-manager)), this controller does not support [Hetzner cloud networks](https://community.hetzner.com/tutorials/hcloud-networks-basic), because using them it is impossible to build a network between the cloud and dedicated servers. For a network in your cluster consisting of dedicated and cloud servers, you should  use some kind of **cni** plugin, example [kube-router](https://github.com/cloudnativelabs/kube-router) with option `--enable-overlay`. If you need [Hetzner cloud networks](https://community.hetzner.com/tutorials/hcloud-networks-basic) then you should use the original [controller](https://github.com/hetznercloud/hcloud-cloud-controller-manager) and refuse to use dedicated servers in your cluster.


# Example
```bash
$ kubectl get node -L node.kubernetes.io/instance-type -L topology.kubernetes.io/region -L topology.kubernetes.io/zone
NAME                STATUS   ROLES    AGE     VERSION   INSTANCE-TYPE   REGION   ZONE
kube-master103-1    Ready    master   6h37m   v1.19.3   cx31            hel1     hel1-dc2 # <-- cloud server
kube-master103-2    Ready    master   6h37m   v1.19.3   cx31            hel1     hel1-dc2 # <-- cloud server
kube-master103-3    Ready    master   6h37m   v1.19.3   cx31            hel1     hel1-dc2 # <-- cloud server
kube-worker103-1    Ready    <none>   6h37m   v1.19.3   cx31            hel1     hel1-dc2 # <-- cloud server
kube-worker103-10   Ready    <none>   3m59s   v1.19.3   EX42-NVMe       hel1     hel1-dc2 # <-- dedicated server
kube-worker103-2    Ready    <none>   6h37m   v1.19.3   cx31            hel1     hel1-dc2 # <-- cloud server

$ kubectl get node -o wide
NAME                STATUS   ROLES    AGE     VERSION   INTERNAL-IP      EXTERNAL-IP       OS-IMAGE             KERNEL-VERSION       CONTAINER-RUNTIME
kube-master103-1    Ready    master   6h39m   v1.19.3   135.181.40.11    <none>            Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
kube-master103-2    Ready    master   6h38m   v1.19.3   <none>           95.200.111.50     Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
kube-master103-3    Ready    master   6h38m   v1.19.3   <none>           95.198.192.60     Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
kube-worker103-1    Ready    <none>   6h38m   v1.19.3   <none>           135.181.121.132   Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
kube-worker103-10   Ready    <none>   5m24s   v1.19.3   <none>           95.216.231.222    Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
kube-worker103-2    Ready    <none>   6h38m   v1.19.3   <none>           135.181.30.198    Ubuntu 20.04.1 LTS   5.4.0-52-generic     containerd://1.2.13
```

Dedicated server:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: "2020-11-03T12:33:13Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: EX42-NVMe # <-- server product
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: hel1 #  <-- location
    failure-domain.beta.kubernetes.io/zone: hel1-dc2 #  <-- location
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker103-10
    kubernetes.io/os: linux
    node.kubernetes.io/instance-type: EX42-NVMe # <-- server product
    topology.kubernetes.io/region: hel1 #  <-- location
    topology.kubernetes.io/zone: hel1-dc2 #  <-- location
  name: kube-worker103-10
  resourceVersion: "115117"
  selfLink: /api/v1/nodes/kube-worker103-10
  uid: fc0c110f-21bf-4f86-924a-c979d73630af
spec:
  podCIDR: 10.245.14.0/24
  podCIDRs:
  - 10.245.14.0/24
  providerID: hetzner://971213
status:
  addresses:
  - address: kube-worker103-10
    type: Hostname
  - address: 95.216.231.222
    type: ExternalIP
  allocatable:
    cpu: "8"
    ephemeral-storage: "450674933014"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 65645832Ki
    pods: "110"
  capacity:
    cpu: "8"
    ephemeral-storage: 489013600Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 65748232Ki
    pods: "110"
  daemonEndpoints:
    kubeletEndpoint:
      Port: 10250
  nodeInfo:
    architecture: amd64
    bootID: 7b57a478-abc1-4818-9cfe-ee37853c0c5c
    containerRuntimeVersion: containerd://1.2.13
    kernelVersion: 5.4.0-52-generic
    kubeProxyVersion: v1.19.3
    kubeletVersion: v1.19.3
    machineID: b756fa1c38304d40b7a81048551c718a
    operatingSystem: linux
    osImage: Ubuntu 18.04.5 LTS
    systemUUID: 5FF254FD-C436-4866-80A0-06690782E6D9
```

Cloud server:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    kubeadm.alpha.kubernetes.io/cri-socket: /run/containerd/containerd.sock
    node.alpha.kubernetes.io/ttl: "0"
    volumes.kubernetes.io/controller-managed-attach-detach: "true"
  creationTimestamp: "2020-11-03T05:59:44Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: cx31 # <-- Server type
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: hel1 #  <-- location
    failure-domain.beta.kubernetes.io/zone: hel1-dc2 # <-- datacenter
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker103-1
    kubernetes.io/os: linux
    node.kubernetes.io/instance-type: cx31 # <-- Server type
    topology.kubernetes.io/region: hel1 #  <-- location
    topology.kubernetes.io/zone: hel1-dc2 # <-- datacenter
  name: kube-worker103-1
  resourceVersion: "112680"
  selfLink: /api/v1/nodes/kube-worker103-1
  uid: f902a256-dd00-4a59-8254-e172bb33d2ee
spec:
  podCIDR: 10.245.4.0/24
  podCIDRs:
  - 10.245.4.0/24
  providerID: hetzner://8440811
status:
  addresses:
  - address: kube-worker103-1
    type: Hostname
  - address: 135.181.121.132
    type: ExternalIP
  allocatable:
    cpu: "2"
    ephemeral-storage: "72456848060"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 7856256Ki
    pods: "110"
  capacity:
    cpu: "2"
    ephemeral-storage: 78620712Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 7958656Ki
    pods: "110"
  daemonEndpoints:
    kubeletEndpoint:
      Port: 10250
  nodeInfo:
    architecture: amd64
    bootID: 53627e44-4e0e-49b5-b711-bc5ffdcee207
    containerRuntimeVersion: containerd://1.2.13
    kernelVersion: 5.4.0-52-generic
    kubeProxyVersion: v1.19.3
    kubeletVersion: v1.19.3
    machineID: 332bcde329124b7fa5f62c03aef2739d
    operatingSystem: linux
    osImage: Ubuntu 20.04.1 LTS
    systemUUID: 332bcde3-2912-4b7f-a5f6-2c03aef2739d
```

# Version matrix
| Kubernetes    | cloud controller | Deployment File |
| ------------- | -----:| ------------------------------------------------------------------------------------------------------:|
| 1.19          | v0.0.5 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment.yaml      |
| 1.15-1.16     | v0.0.4 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.4/deploy/v0.0.4-deployment.yaml      |


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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment.yaml
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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment-exclude.yaml
```

# Evnironment variables

 * `HCLOUD_TOKEN` (**required**) - Hcloud API access token
 * `HCLOUD_ENDPOINT`(default: `https://api.hetzner.cloud/v1`) - endpoint for Hcloud API
 * `NODE_NAME` (**required**)  - name of the node on which the application is running (spec.nodeName)
 * `HROBOT_USER` (**required**) - user to access the Hrobot API
 * `HROBOT_PASS` (**required**) - password to access the Hrobot API
 * `HROBOT_PERIOD`(default: `180`) - period in seconds with which the Hrobot API will be polled

Hrobot get server have limit [200 requests per hour](https://robot.your-server.de/doc/webservice/en.html#get-server). Therefore, the application receives this information with the specified period (`HROBOT_PERIOD`) and saves it in memory. One poll every 180 seconds means 20 queries per hour.

# License

Apache License, Version 2.0
