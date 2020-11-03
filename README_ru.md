# Kubernetes Cloud Controller Manager для Hetzner Cloud и Hetzner Dedicated
Данный контроллер основан на [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager) но помимо Hetzner Cloud поддерживает выделенные сервера [Hetzner](https://www.hetzner.com/dedicated-rootserver).

Функции:
 * добавялет метки `beta.kubernetes.io/instance-type`, `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`, `node.kubernetes.io/instance-type`, `topology.kubernetes.io/region`, `topology.kubernetes.io/zone`
 * устанавливет внешний ip в status.addresses
 * удаляет ноды из kubernetes кластера если они были удалены из Hetzner Cloud или из Hetzner Robot
 * позволяет исключить удаление нод, которые принадлежат другим провайдерам (kubelet на этих нодах следует запускать БЕЗ опции `--cloud-provider=external`). Смотри раздел [Исключение нод](#исключение-нод)

Больше информации о cloud controller manager вы можете найти в [документации kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)

**На заметку:** в отличие от оригинального [контроллера](https://github.com/hetznercloud/hcloud-cloud-controller-manager), этот контроллер не поддерживает [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), поскольку невозможно с помощью них настроить сеть между dedicated и облачным серверами. Для сети в вашем кластере состоящим из облачных и dedicated серверов следует использовать какой-нибудь **cni** плагин, например [kube-router](https://github.com/cloudnativelabs/kube-router) с включенным `--enable-overlay`. Если вам требуются [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), то вам стоит использовать оригинальный [контроллер](https://github.com/hetznercloud/hcloud-cloud-controller-manager) и отказаться от использования dedicated серверов в вашем кластере.

# Примеры
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

# Cовместимость версий
| Kubernetes    | cloud controller | Deployment File |
| ------------- | -----:| ------------------------------------------------------------------------------------------------------:|
| 1.19          | v0.0.5 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment.yaml      |
| 1.15-1.16     | v0.0.4 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.4/deploy/v0.0.4-deployment.yaml      |


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
Или сделайте тоже самое однострочной командой:
```bash
kubectl create secret generic hetzner-cloud-controller-manager --from-literal=token=pYMfn43zP42ET6N2GtoWX35CUGspyfDA2zbbP57atHKpsFm7YUKbAdcTXFvSyu --from-literal=robot_user='#as+BVacIALV' --from-literal=robot_password=XRmL7hjAMU3RVsXJ4qLpCExiYpcKFJKzMKCiPjzQpJ33RP3b5uHY5DhqhF44YarY
```

Деплой контроллера:
```bash
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment.yaml
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

Затем добавьте **taint** на эту ноду. Если на ноде нету никаких **taints**, то выполните:
```bash
kubectl patch node kube-node-example1 --type='json' -p='[{"op":"add","path":"/spec/taints","value": [{"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized"}]}]'
```
Если на ноде уже есть какие-то **taints** то:
```bash
kubectl patch node kube-node-example1 --type='json' -p='[{"op":"add","path":"/spec/taints/-","value": {"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized"}}]'
```

Контроллер обнаружит этот **taint**, иницализирует ноду и удалит **taint**.

## Исключение нод
Если вы хотите добавлять узлы в ваш кластер из других облачных/bare-metal провайдеров. То вы можете столкнутся с проблемой - узлы будут удалятся из кластера. Это можно обойти исключив данные сервера, с помощью конфигурации в JSON формате. Для этого вам достаточно добавить в секрет **hetzner-cloud-controller-manager** ключ `cloud_token` с перечислением серверов, которые требуется исключить. И добавить этот файл в опцию `--cloud-config` в деплойменте. Поддерживаются регулярные выражения. Например:

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

Очень важно на таких серверах запускать kubelet БЕЗ опции `--cloud-provider=external`.

Для деплоя с исключением предусмотрен отдельный файл: 
```bash
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.5/deploy/v0.0.5-deployment-exclude.yaml
```

# Переменные среды

 * `HCLOUD_TOKEN` (**обязательный**) - токен для доступа к Hcloud API
 * `HCLOUD_ENDPOINT`(умолчание: `https://api.hetzner.cloud/v1`) - endpoint для Hcloud API
 * `NODE_NAME` (**обязательный**)  - имя ноды, на которой запущен под (spec.nodeName)
 * `HROBOT_USER` (**обязательный**) - пользователь для доступа к Hrobot API
 * `HROBOT_PASS` (**обязательный**) - пароль для доступа к Hrobot API
 * `HROBOT_PERIOD`(умолчание: `180`) - период в секундах с которым будет опрашиваться Hrobot API

Запросы на Hrobot имеют лимит [200 запросов в час](https://robot.your-server.de/doc/webservice/en.html#get-server). Поэтому приложение опрашивает его с указанным периодом `HROBOT_PERIOD`, а результаты хранит в памяти. Опрос раз в 180 секунд это 20 запросов в час. Данный параметр можете подобрать под ваши нужды.

# Лицензия

Apache License, Version 2.0