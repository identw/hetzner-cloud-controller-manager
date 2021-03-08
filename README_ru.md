# Kubernetes Cloud Controller Manager для Hetzner Cloud и Hetzner Dedicated
Данный контроллер основан на [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager) но помимо Hetzner Cloud поддерживает выделенные сервера [Hetzner](https://www.hetzner.com/dedicated-rootserver).

Функции:
 * добавялет метки `beta.kubernetes.io/instance-type`, `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`, `node.kubernetes.io/instance-type`, `topology.kubernetes.io/region`, `topology.kubernetes.io/zone`
 * добавляет метку `node.hetzner.com/type`, в которой указан тип ноды (облачная или dedicated)
 * копирует метки из облачных серверов в метки k8s ноды, смотри раздел [Копирование меток с облачных узлов](#копирование-меток-с-облачных-узлов)
 * устанавливет внешний ip в status.addresses
 * удаляет ноды из kubernetes кластера если они были удалены из Hetzner Cloud или из Hetzner Robot
 * позволяет исключить удаление нод, которые принадлежат другим провайдерам (kubelet на этих нодах следует запускать БЕЗ опции `--cloud-provider=external`). Смотри раздел [Исключение нод](#исключение-нод)

Больше информации о cloud controller manager вы можете найти в [документации kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)

**На заметку:** В robot панели, dedicated серверам, необходимо назначить имя совпадающее с именем ноды в k8s кластере. Это нужно для того чтобы контроллер смог обнаружить сервер через API, при первичной инициализации ноды. После первичной инициализации контроллер будет использовать ID сервера.

**На заметку:** В отличие от оригинального [контроллера](https://github.com/hetznercloud/hcloud-cloud-controller-manager), этот контроллер не поддерживает [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), поскольку невозможно с помощью них настроить сеть между dedicated и облачным серверами. Для сети в вашем кластере состоящим из облачных и dedicated серверов следует использовать какой-нибудь **cni** плагин, например [kube-router](https://github.com/cloudnativelabs/kube-router) с включенным `--enable-overlay`. Если вам требуются [облачные сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), то вам стоит использовать оригинальный [контроллер](https://github.com/hetznercloud/hcloud-cloud-controller-manager) и отказаться от использования dedicated серверов в вашем кластере.

# Содержимое
 * [Примеры](#примеры)
 * [Cовместимость версий](#cовместимость-версий)
 * [Деплой](#Деплой)
   - [Инициализация уже существующих нод в кластере](#инициализация-уже-существующих-нод-в-кластере)
   - [Исключение нод](#исключение-нод)
 * [Переменные среды](#переменные-среды)
 * [Копирование меток с облачных узлов](#копирование-меток-с-облачных-узлов)
 * [Лицензия](#лицензия)

# Примеры
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

# Cовместимость версий
| Kubernetes    | cloud controller | Deployment File |
| ------------- | -----:| ------------------------------------------------------------------------------------------------------:|
| 1.20.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml      |
| 1.19.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.18.x/deploy/deploy.yaml      |
| 1.18.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.18.x/deploy/deploy.yaml      |
| 1.17.x          | v0.0.6 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml      |
| 1.16.x          | v0.0.6-k8s-v1.16.x | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6-k8s-v1.16.x/deploy/deploy.yaml      |

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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy.yaml
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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.6/deploy/deploy-exclude.yaml
```

# Переменные среды

 * `HCLOUD_TOKEN` (**обязательный**) - токен для доступа к Hcloud API
 * `HCLOUD_ENDPOINT`(умолчание: `https://api.hetzner.cloud/v1`) - endpoint для Hcloud API
 * `NODE_NAME` (**обязательный**)  - имя ноды, на которой запущен под (spec.nodeName)
 * `HROBOT_USER` (**обязательный**) - пользователь для доступа к Hrobot API
 * `HROBOT_PASS` (**обязательный**) - пароль для доступа к Hrobot API
 * `HROBOT_PERIOD`(умолчание: `180`) - период в секундах с которым будет опрашиваться Hrobot API
 * `PROVIDER_NAME` (умолчание: `hetzner`) - название провайдера, которое будет использовано в префиксе спецификации ноды (`spec.providerID`)
 * `NAME_LABEL_TYPE` (умолчание: `node.hetzner.com/type`) - название лейбла в котором будет хранится информация о типе ноды (облачная или dedicted)
 * `NAME_CLOUD_NODE` (умолчание: `cloud`) - название облачной ноды в значении лейбла `NAME_LABEL_TYPE`
 * `NAME_DEDICATED_NODE` (умолчание: `dedicated`) - название dedicated ноды в значении лейбла `NAME_LABEL_TYPE`
 * `ENABLE_SYNC_LABELS` (умолчание: `true`) - включает/выключает копирование меток из облачных серверов


Запросы на Hrobot имеют лимит [200 запросов в час](https://robot.your-server.de/doc/webservice/en.html#get-server). Поэтому приложение опрашивает его с указанным периодом `HROBOT_PERIOD`, а результаты хранит в памяти. Опрос раз в 180 секунд это 20 запросов в час. Данный параметр можете подобрать под ваши нужды.

# Копирование меток с облачных узлов
На облачных серверах можно устанавливать метки
```bash
$ hcloud server list -o columns=id,name,labels
NAME                         LABELS
kube-master121-1             myLabel=myValue
kube-worker121-1             myLabel=myValue
```
Контроллер копирует эти метки в метки k8s ноды:
```bash
$ kubectl get node -L myLabel -L node.hetzner.com/type
NAME               STATUS   ROLES                  AGE   VERSION   MYLABEL   TYPE
kube-master121-1   Ready    master                 36m   v1.16.15   myValue   cloud
kube-worker121-1   Ready    <none>                 35m   v1.16.15   myValue   cloud
kube-worker121-2   Ready    <none>                 20m   v1.16.15             dedicated
```
Это поведение можно отключить задав переменную среды `ENABLE_SYNC_LABELS=false`.

Изменение метки, также приводит к изменению этой метки в кластере.  Удаление метки с облачного сервера, также удалит ее с ноды k8s:
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

Синхронизация происходит не моментально, а с интервалом 5 минут. Это можно изменить через аргумент `--node-status-update-frequency`. Но будте осторожны, есть лимит на количество запросов в API hetzner. Я бы не рекомендовал менять этот параметр.

# Лицензия

Apache License, Version 2.0