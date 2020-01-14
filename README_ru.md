# Kubernetes Cloud Controller Manager для Hetzner Cloud и Hetzner Dedicated
Данный контроллер основан на [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager) но помимо Hetzner Cloud поддерживает выделенные сервера [Hetzner](https://www.hetzner.com/dedicated-rootserver).

Он добавляет на ноды метки: `beta.kubernetes.io/instance-type`, `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`, устанавливет внешний ip в статус, а также удаляет ноды из **kubernetes** кластера, если они были удалены из Hetzner Cloud или из Hetzner Robot.

Больше информации о cloud controller manager вы можете найти в [документации kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)

**На заметку:** в отличие от оригинального [контроллера](https://github.com/hetznercloud/hcloud-cloud-controller-manager), этот контроллер не поддерживает [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), поскольку невозможно с помощью них настроить сеть между dedicated и облачным серверами. Для сети в вашем кластере состоящим из облачных и dedicated серверов следует использовать какой-нибудь **cni** плагин, например [kube-router](https://github.com/cloudnativelabs/kube-router) с включенным `--enable-overlay`. Если вам требуются [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), то вам стоит использовать оригинальный [контроллер](https://github.com/hetznercloud/hcloud-cloud-controller-manager) и отказаться от использования dedicated серверов в вашем кластере.

# Примеры
```bash
$ kubectl get node -L beta.kubernetes.io/instance-type -L failure-domain.beta.kubernetes.io/region -L failure-domain.beta.kubernetes.io/zone
NAME               STATUS   ROLES    AGE     VERSION   INSTANCE-TYPE   REGION   ZONE
kube-master102-1   Ready    master   9d      v1.15.3   cx21            nbg1     nbg1-dc3 # <-- cloud server
kube-worker102-1   Ready    <none>   3m21s   v1.15.3   cx31            nbg1     nbg1-dc3 # <-- cloud server
kube-worker102-3   Ready    <none>   3m37s   v1.15.3   cx31            nbg1     nbg1-dc3 # <-- cloud server
kube-worker102-4   Ready    <none>   2d18h   v1.15.3   EX41S-SSD       fsn1     fsn1-dc8 # <-- dedicated server

$ kubectl get node -o wide
NAME               STATUS   ROLES    AGE     VERSION   INTERNAL-IP   EXTERNAL-IP      OS-IMAGE             KERNEL-VERSION      CONTAINER-RUNTIME
kube-master102-1   Ready    master   9d      v1.15.3   <none>        78.47.171.273    Ubuntu 18.04.3 LTS   4.18.0-25-generic   docker://18.9.8
kube-worker102-1   Ready    <none>   24m     v1.15.3   <none>        78.47.111.15     Ubuntu 18.04.3 LTS   4.15.0-72-generic   docker://18.9.8
kube-worker102-3   Ready    <none>   24m     v1.15.3   <none>        78.47.156.13     Ubuntu 18.04.3 LTS   4.15.0-72-generic   docker://18.9.8
kube-worker102-4   Ready    <none>   2d18h   v1.15.3   <none>        138.205.17.11    Ubuntu 18.04.3 LTS   4.18.0-25-generic   docker://18.9.8
```

Dedicated сервер:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
  creationTimestamp: "2020-01-10T12:38:09Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: EX41S-SSD # <-- server product
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: fsn1   #  <-- location
    failure-domain.beta.kubernetes.io/zone: fsn1-dc8 #  <-- datacenter
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker102-4
    kubernetes.io/os: linux
  name: kube-worker102-4
  resourceVersion: "1044876"
  selfLink: /api/v1/nodes/kube-worker102-4
  uid: 9e6c1873-cd43-482d-90a8-43d676dcd1fa
spec:
  podCIDR: 10.244.54.0/24
  providerID: hetzner://902711 # <-- Server ID
status:
  addresses:
  - address: kube-worker102-4
    type: Hostname
  - address: 138.205.17.11 # <-- Public ipv4
    type: ExternalIP
  allocatable:
    cpu: "8"
    ephemeral-storage: "218529260797"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 65637400Ki
    pods: "110"

```
Облачный сервер:
```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    kubeadm.alpha.kubernetes.io/cri-socket: /var/run/dockershim.sock
    node.alpha.kubernetes.io/ttl: "0"
  creationTimestamp: "2020-01-13T06:49:28Z"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: cx31 # <-- Server type
    beta.kubernetes.io/os: linux
    failure-domain.beta.kubernetes.io/region: nbg1    #  <-- location
    failure-domain.beta.kubernetes.io/zone: nbg1-dc3  # <-- datacenter
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kube-worker102-3
    kubernetes.io/os: linux
  name: kube-worker102-3
  resourceVersion: "1045728"
  selfLink: /api/v1/nodes/kube-worker102-3
  uid: e626e314-7c28-4f54-86cd-6c0a10493a44
spec:
  podCIDR: 10.244.1.0/24
  providerID: hetzner://4017715 # <-- Server ID
status:
  addresses:
  - address: kube-worker102-3
    type: Hostname
  - address: 78.47.156.13  # <-- Public ipv4
    type: ExternalIP
  allocatable:
    cpu: "2"
    ephemeral-storage: "72538243772"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 7871308Ki
    pods: "110"
```

# Деплой
Вам нужно создать токен для доступа к API Hetzner Cloud и к API Hetzner Robot. Для этого следуйте следующим инструкциям:
 * https://docs.hetzner.cloud/#overview-getting-started
 * https://robot.your-server.de/doc/webservice/en.html#preface

Получив токен и доступы, создайте файл с секретами (`hetzner-cloud-controller-manager-secret.yaml`):
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
И примените его:
```bash
kubectl apply -f hetzner-cloud-controller-manager-secret.yaml
```
Или сделайте тоже самое через интерактивную команду создания секрета в kubectl:
```bash
kubectl create secret generic hetzner-cloud-controller-manager --from-literal=token=pYMfn43zP42ET6N2GtoWX35CUGspyfDA2zbbP57atHKpsFm7YUKbAdcTXFvSyu --from-literal=robot_user='#as+BVacIALV' --from-literal=robot_password=XRmL7hjAMU3RVsXJ4qLpCExiYpcKFJKzMKCiPjzQpJ33RP3b5uHY5DhqhF44YarY
```

Деплой контроллера:
```bash
kubectl apply -f deploy/v0.0.1-deployment.yaml
```

Теперь добавляя новые узлы в кластер, запускайте на них **kubelet** c параметром: `--cloud-provider=external`. Для этого вы можете создать файл: `/etc/systemd/system/kubelet.service.d/20-external-cloud.conf` со следующим содержимым:

```
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
```

 И перегрузите systemctl:
 ```bash
 systemctl daemon-reload
 ```

Далее добавляете ноду как обычно. Напирмер если это **kubeadm**, то:
```bash
kubeadm join kube-api-server:6443 --token token  --discovery-token-ca-cert-hash sha256:hash 
```

## Инициализация уже существующих нод в кластере
Если у вас до деплоя контроллера уже были ноды в кластере. То вы можете их переинициализировать. Для этого достаточно запустить на них **kubelet** c `--cloud-provider=external` и затем вручную добавить **taint** с ключем `node.cloudprovider.kubernetes.io/uninitialized` и эффектом `NoSchedule`.


```bash
ssh kube-node-example1
echo '[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
' > /etc/systemd/system/kubelet.service.d/20-external-cloud.conf
systemctl daemon-reload
systemctl restart kubelet
```

Затем добавьте **taint** на эту ноду
```bash
kubectl patch node kube-node-example1 --type='json' -p='[{"op":"add","path":"/spec/taints/-","value": {"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized"}}]'
```

Контроллер обнаружит этот **taint**, иницализирует ноду и удалит **taint**.


# Лицензия

Apache License, Version 2.0