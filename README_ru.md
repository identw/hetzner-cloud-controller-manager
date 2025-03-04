# Kubernetes Cloud Controller Manager для Hetzner Cloud и Hetzner Dedicated
Данный контроллер основан на [hcloud-cloud-controller-manager](https://github.com/hetznercloud/hcloud-cloud-controller-manager) но помимо Hetzner Cloud поддерживает выделенные сервера [Hetzner](https://www.hetzner.com/dedicated-rootserver).

Функции:
 * **instances interface**: добавляет на узлы метки `beta.kubernetes.io/instance-type`, `node.kubernetes.io/instance-type`. Устанавливает внешние ipv4 адреса и удаляет ноды из kubernetes кластера если они были удалены из Hetzner Cloud или из Hetzner Robot
 * **zones interface**: добавялет на узлы метки `failure-domain.beta.kubernetes.io/region`, `failure-domain.beta.kubernetes.io/zone`,`topology.kubernetes.io/region`, `topology.kubernetes.io/zone`
 * **Load Balancers**: Позволяет использовать Hetzner Cloud Load Balancers как для облачных нод так и для dedicated серверов
 * добавляет метку `node.hetzner.com/type`, в которой указан тип ноды (облачная или dedicated)
 * копирует метки из облачных серверов в метки k8s ноды, смотри раздел [Копирование меток с облачных узлов](#копирование-меток-с-облачных-узлов)
 * позволяет исключить удаление нод, которые принадлежат другим провайдерам и хостингам (kubelet на этих нодах следует запускать БЕЗ опции `--cloud-provider=external`). Смотри раздел [Исключение нод](#исключение-нод)

**На заметку:** В robot панели, dedicated серверам, необходимо назначить имя совпадающее с именем ноды в k8s кластере. Это нужно для того чтобы контроллер смог обнаружить сервер через API, при первичной инициализации ноды. После первичной инициализации контроллер будет использовать ID сервера.

**На заметку:** В отличие от оригинального [контроллера](https://github.com/hetznercloud/hcloud-cloud-controller-manager), этот контроллер не поддерживает [облачные сети Hetzner](https://community.hetzner.com/tutorials/hcloud-networks-basic), поскольку нет технической возможности это реализовать. Подробнее смотрите в разделе [почему нет поддержки сетей hetzner cloud](#почему-нет-поддержки-сетей-hetzner-cloud). Для сети в вашем кластере состоящим из облачных и dedicated серверов следует использовать какой-нибудь **cni** плагин, который построет overlay сеть между серверами. Например [kube-router](https://github.com/cloudnativelabs/kube-router) с включенным `--enable-overlay` или [cilium](https://cilium.io/) с `tunnel: vxlan` или `tunnel: geneve`. Если вам требуются [сети Hetzner Cloud](https://community.hetzner.com/tutorials/hcloud-networks-basic), то вам стоит использовать оригинальный [контроллер](https://github.com/hetznercloud/hcloud-cloud-controller-manager) и отказаться от использования dedicated серверов в вашем кластере.

Больше информации о cloud controller manager вы можете найти в [документации kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)


# Содержание
 * [Примеры](#примеры)
 * [Cовместимость версий](#cовместимость-версий)
 * [Деплой](#Деплой)
   - [Инициализация уже существующих нод в кластере](#инициализация-уже-существующих-нод-в-кластере)
   - [Исключение нод](#исключение-нод)
 * [Переменные среды](#переменные-среды)
 * [Копирование меток с облачных узлов](#копирование-меток-с-облачных-узлов)
 * [Балансировщики нагрузки](#балансировщики-нагрузки)
   - [Аннотации](#аннотации)
   - [Примеры](#примеры-аннотаций-для-балансировщика)
 * [Почему нет поддержки сетей hetzner cloud](#почему-нет-поддержки-сетей-hetzner-cloud)
 * [Лицензия](#лицензия)

# Примеры
```bash
$ kubectl get node -L node.kubernetes.io/instance-type -L topology.kubernetes.io/region -L topology.kubernetes.io/zone -L node.hetzner.com/type
NAME               STATUS   ROLES                  AGE     VERSION   INSTANCE-TYPE   REGION   ZONE       TYPE
kube-master121-1   Ready    control-plane,master   25m     v1.20.4   cx31            hel1     hel1-dc2   cloud
kube-worker121-1   Ready    <none>                 24m     v1.20.4   cx31            hel1     hel1-dc2   cloud
kube-worker121-2   Ready    <none>                 9m18s   v1.20.4   AX41-NVMe       hel1     hel1-dc4   dedicated

$ kubectl get node -o wide
NAME               STATUS   ROLES                  AGE     VERSION   INTERNAL-IP   EXTERNAL-IP      OS-IMAGE             KERNEL-VERSION     CONTAINER-RUNTIME
kube-master121-1   Ready    control-plane,master   25m     v1.20.4   <none>        95.131.108.198   Ubuntu 20.04.2 LTS   5.4.0-65-generic   containerd://1.2.13
kube-worker121-1   Ready    <none>                 25m     v1.20.4   <none>        95.131.234.167   Ubuntu 20.04.2 LTS   5.4.0-65-generic   containerd://1.2.13
kube-worker121-2   Ready    <none>                 9m40s   v1.20.4   <none>        111.233.1.99     Ubuntu 20.04.2 LTS   5.4.0-65-generic   containerd://1.2.13
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
    node.kubernetes.io/instance-type: AX41-NVMe # <-- server product
    topology.kubernetes.io/region: hel1 # <-- location
    topology.kubernetes.io/zone: hel1-dc4 # <-- datacenter
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
    node.kubernetes.io/instance-type: cx31 # <-- server type
    topology.kubernetes.io/region: hel1 # <-- location
    topology.kubernetes.io/zone: hel1-dc2 # <-- datacenter
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
| 1.24.x          | v0.0.9 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.9/deploy/deploy.yaml      |
| 1.20.x          | v0.0.8 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy.yaml      |
| 1.19.x          | v0.0.8 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy.yaml      |
| 1.18.x          | v0.0.8 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy.yaml      |
| 1.17.x          | v0.0.8 | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy.yaml      |
| 1.16.x          | v0.0.8-k8s-v1.16.x | https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8-k8s-v1.16.x/deploy/deploy.yaml      |

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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy.yaml
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
kubectl apply -f https://raw.githubusercontent.com/identw/hetzner-cloud-controller-manager/v0.0.8/deploy/deploy-exclude.yaml
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
 * `HCLOUD_LOAD_BALANCERS_ENABLED` - (умолчание: `true`) - отключает/включает поддержку балансировщиков нагрузки
 * `HCLOUD_LOAD_BALANCERS_LOCATION` (умолчания нет, взаимоисключающий с `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`) - location по умолчанию, в котором будут создаваться балансировщики. Например: `fsn1`, `nbg1`, `hel1`
 * `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE` (умолчания нет, взаимоисключающий с `HCLOUD_LOAD_BALANCERS_LOCATION` - network zone по умолчанию. Нарпимер: `eu-central`

 Переменные среды `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`, `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP` из оригинального контроллера не имеют смысла, поскольку данный контроллер не поддерживает [Hetzner Cloud сети](https://community.hetzner.com/tutorials/hcloud-networks-basic). 

Доступные локации и network zone можно узнать с помощью hcloud
```bash
$ hcloud location list
ID   NAME   DESCRIPTION             NETWORK ZONE   COUNTRY   CITY
1    fsn1   Falkenstein DC Park 1   eu-central     DE        Falkenstein
2    nbg1   Nuremberg DC Park 1     eu-central     DE        Nuremberg
3    hel1   Helsinki DC Park 1      eu-central     FI        Helsinki
```

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
kube-master121-1   Ready    control-plane,master   36m   v1.20.4   myValue   cloud
kube-worker121-1   Ready    <none>                 35m   v1.20.4   myValue   cloud
kube-worker121-2   Ready    <none>                 20m   v1.20.4             dedicated
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
kube-master121-1   Ready    control-plane,master   37m   v1.20.4   myValue   cloud
kube-worker121-1   Ready    <none>                 37m   v1.20.4             cloud
kube-worker121-2   Ready    <none>                 21m   v1.20.4             dedicated
```

Синхронизация происходит не моментально, а с интервалом 5 минут. Это можно изменить через аргумент `--node-status-update-frequency`. Но будте осторожны, есть лимит на количество запросов в API hetzner. Я бы не рекомендовал менять этот параметр.

# Балансировщики нагрузки
Контроллер поддерживает [Load Balancers](https://www.hetzner.com/cloud/load-balancer) как для облачных узлов так и для dedicated. Dedicated сервера должны принадлежать владельцу проекта в облаке и должны управляться той же учетной записью.

В targets добавляются все worker узлы кластера. Dedicated сервера имеют тип `ip`, а облачные `server`. 

Например:
```
$ hcloud load-balancer list
ID       NAME                               IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
236648   a3767f900602b4c9093823670db0372c   95.217.174.188   2a01:4f9:c01e:38c::1   lb11   hel1       eu-central


$ hcloud load-balancer describe 236648
ID:				236648
Name:				a3767f900602b4c9093823670db0372c
...
Targets:
  - Type:			server
    Server:
      ID:			10193451
      Name:			kube-worker121-1
    Use Private IP:		no
    Status:
    - Service:			3000
      Status:			healthy
  - Type:			ip
    IP:				111.233.1.99
    Status:
    - Service:			3000
      Status:			healthy
...
```
kube-worker121-1 - облачный сервер, 111.233.1.99 - dedicated сервер (kube-worker121-2).

## Аннотации
На параметры балансировщика вы можете влиять через аннотации к сервису

 * `load-balancer.hetzner.cloud/name` - имя балансировщика, по умолчанию используется случайно сгенерированный id, например `a3767f900602b4c9093823670db0372c`
 * `load-balancer.hetzner.cloud/hostname` - Хостнейм балансировщика, который будет указан в статусе сервиса (service.status.loadBalancer.ingress)
 * `load-balancer.hetzner.cloud/hostname` - 
 * `load-balancer.hetzner.cloud/external-dns-hostname` - Хостнейм балансировщика, который будет указан в статусе сервиса (service.status.loadBalancer.ingress). И добавил две аннотации для external-dns: `external-dns.alpha.kubernetes.io/target: <ipv4-address>,<ipv6-address>` and `external-dns.alpha.kubernetes.io/hostname: the value from load-balancer.hetzner.cloud/external-dns-hostname`. Это удобно для автоматического создания DNS записи (как в aws nlb).
 * `load-balancer.hetzner.cloud/protocol` (умолчание: `tcp`) - протокол, возможные значения: `tcp`, `http`, `https`
 * `load-balancer.hetzner.cloud/algorithm-type` (умолчание: `round_robin`) - алгоритм балансировки, возможные значения: `round_robin`, `least_connections`
 * `load-balancer.hetzner.cloud/type` (умолчание: `lb11`) - тип балансировщика, возможные значения: `lb11`, `lb21`, `lb31`
 * `load-balancer.hetzner.cloud/location` - локация, возможные значения: `fsn1`, `ngb1`, `hel1`. Взаимоисключающая с `load-balancer.hetzner.cloud/network-zone`. Можно задать умолчание с помощью переменной среды `HCLOUD_LOAD_BALANCERS_LOCATION`. Смена локции требует пересоздание службы и смену ип адреса.
 * `load-balancer.hetzner.cloud/network-zone` - зона, возможные значения: `eu-central`. Взаимоисключающая с `load-balancer.hetzner.cloud/location`. Можно задать умолчание с помощью переменной среды `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`
 * `load-balancer.hetzner.cloud/uses-proxyprotocol` (умолчание `false`) - включить proxy protocol. Требует поддержку со стороны приложения
 * `load-balancer.hetzner.cloud/http-sticky-sessions` - включить sticky-sessions с привязкой к куке
 * `load-balancer.hetzner.cloud/http-cookie-name` - имя куки при http/https балансере с включенным sticky-sessions
 * `load-balancer.hetzner.cloud/http-cookie-lifetime` - время жизни куки при http/https балансере с включенным sticky-sessions
 * `load-balancer.hetzner.cloud/http-certificates` - Id сертификатов перечисленных через запятую. Только для https протокола
 * `load-balancer.hetzner.cloud/http-redirect-http` - редирект с http на https. Только для https протокола
 * `load-balancer.hetzner.cloud/health-check-protocol` (умолчание `tcp`) - протокол для проверок сервиса. Возможные значения: `tcp`, `http`, `https`
 * `load-balancer.hetzner.cloud/health-check-port` - порт для проверок сервиса
 * `load-balancer.hetzner.cloud/health-check-interval` - интервал для проверок севриса
 * `load-balancer.hetzner.cloud/health-check-timeout` - таймаут проверки сервиса
 * `load-balancer.hetzner.cloud/health-check-retries` - количетсво попыток проверки, прежде чем считать сервис недоступным
 * `load-balancer.hetzner.cloud/health-check-http-domain` - домен для заголовка `Host` для проверок сервиса
 * `load-balancer.hetzner.cloud/health-check-http-path` - uri для проверок сервиса
 * `load-balancer.hetzner.cloud/health-check-http-validate-certificate` - проверять валидность ssl сертификата при проверках сервиса
 * `load-balancer.hetzner.cloud/http-status-codes` - какие http коды ответов считать успешными при проверках сервиса

 Аннотации из оригинального контроллера `load-balancer.hetzner.cloud/disable-public-network`, `load-balancer.hetzner.cloud/disable-private-ingress`, `load-balancer.hetzner.cloud/use-private-ip` не имеют смысла, поскольку данный контроллер не поддерживает [Hetzner Cloud сети](https://community.hetzner.com/tutorials/hcloud-networks-basic).

## Примеры аннотаций для балансировщика
Создаем балансировщик в локации `hel1`:
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/location: hel1
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```

```bash
$ hcloud loab-balancer list
ID       NAME                               IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
237283   a0e106866840c401ca5eff56ccb06130   95.217.172.232   2a01:4f9:c01e:424::1   lb11   hel1       eu-central

$ kubectl get svc nginx
NAME    TYPE           CLUSTER-IP       EXTERNAL-IP                           PORT(S)          AGE
nginx   LoadBalancer   10.109.210.214   2a01:4f9:c01e:424::1,95.217.172.232   3000:30905/TCP   30m
```

Для смены локации необходимо пересоздать сервис, что приведет к смене ип адреса балансировщика. Чтобы не указывать каждый раз локацию в аннотации вы можете сделать какую-то локацию по умолчанию передав контроллеру переменную среды `HCLOUD_LOAD_BALANCERS_LOCATION`. 

Если локация не имеет значения, вы можете указать зону:
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/network-zone: eu-central
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```

```bash
$ hcloud load-balancer list
ID       NAME                               IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
237284   a526bb12dc33143d69b084c3e2d2e58b   95.217.172.232   2a01:4f9:c01e:424::1   lb11   hel1       eu-central
```

Для удобства вы можете назначить имя балансировщику, которое будет отображаться в API и веб-инетрфейсе. Это не требует пересоздания сервиса и балансировщика.
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/name: nginx-example
    load-balancer.hetzner.cloud/location: hel1
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```

```bash
$ hcloud load-balancer list
ID       NAME            IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
237284   nginx-example   95.217.172.232   2a01:4f9:c01e:424::1   lb11   hel1       eu-central
```

Вместо ип адреса в статусе службы k8s, вы можете указать домен:
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/name: nginx-example
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/hostname: example.com
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```
```bash
$ hcloud load-balancer list
ID       NAME            IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
237284   nginx-example   95.217.172.232   2a01:4f9:c01e:424::1   lb11   hel1       eu-central

$ kubectl get svc nginx
NAME    TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
nginx   LoadBalancer   10.109.210.214   example.com   3000:30905/TCP   26m

$ kubectl get svc nginx -o yaml
apiVersion: v1
kind: Service
...
status:
  loadBalancer:
    ingress:
    - hostname: example.com
```

Если вам не хватает возможностей текущего балансировщика, вы можете взять подороже. Это также не требует пересоздания службы.
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/name: nginx-example
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/type: lb21
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
  ```

```bash
$ hcloud load-balancer list
ID       NAME            IPV4             IPV6                   TYPE   LOCATION   NETWORK ZONE
237284   nginx-example   95.217.172.232   2a01:4f9:c01e:424::1   lb21   hel1       eu-central
```

Меняем алгоритм распределения запросов на `least_connections`
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/name: nginx-example
    load-balancer.hetzner.cloud/location: hel1
    load-balancer.hetzner.cloud/algorithm-type: least_connections
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```


```bash
$ hcloud load-balancer describe 237284
ID:				237284
Name:				nginx-example
Public Net:
  Enabled:			yes
  IPv4:				95.217.172.232
  IPv6:				2a01:4f9:c01e:424::1
Private Net:
    No Private Network
Algorithm:			least_connections
Load Balancer Type:		lb11 (ID: 1)
...
```

Меняем протокол на http:
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    load-balancer.hetzner.cloud/name: nginx-example
    load-balancer.hetzner.cloud/protocol: http
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: http
  selector:
    app: nginx
  type: LoadBalancer
```

```bash
$ hcloud load-balancer describe 237284
ID:				237284
Name:				nginx-example
Public Net:
  Enabled:			yes
  IPv4:				95.217.172.232
  IPv6:				2a01:4f9:c01e:424::1
Services:
  - Protocol:			http
    Listen Port:		3000
 ...
```

# Почему нет поддержки сетей hetzner cloud
К сожалению, реализовать это технически невозможно, так как есть ограничения в возможностях маршрутизации в облаке hetzner.

Например:  
Облачная подсеть: 10.240.0.0/12, подсеть для облачных узлов: 10.240.0.0/24, подсеть для dedicated узлов vswitch: 10.240.1.0/24, сеть подов кластера k8s: 10.245.0.0/16  
```
kube-master121-1 (cloud node, hel-dc2) - public IP: 95.216.201.207, private IP: 10.240.0.2, pod network: 10.245.0.0/24
kube-worker121-1 (cloud node, hel-dc2) - public IP: 95.216.209.218, private IP: 10.240.0.3,  pod network: 10.245.1.0/24
kube-worker121-2 (cloud node, hel-dc2) - public IP: 135.181.41.158, private IP: 10.240.0.4, pod network: 10.245.2.0/24
kube-worker121-10 (dedicated node, hel-dc4) public IP: 135.181.96.131, private IP: 10.240.1.2, pod network: 10.245.3.0/24
```

Vlan сеть на kube-worker121-10:
```
3: enp34s0.4002@enp34s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc noqueue state UP group default qlen 1000
    link/ether 2c:f0:6d:05:d1:ae brd ff:ff:ff:ff:ff:ff
    inet 10.240.1.2/24 scope global enp34s0.4002
       valid_lft forever preferred_lft forever
    inet6 fe80::2ef0:5dff:fe0d:dcae/64 scope link 
       valid_lft forever preferred_lft forever
```
![image](https://user-images.githubusercontent.com/22338503/111874592-ce69e900-89b7-11eb-8ab8-288840d83f5c.png)
![image](https://user-images.githubusercontent.com/22338503/111874636-e7729a00-89b7-11eb-9686-6fe438633c11.png)


Конечно, это дает возможность объеденить в одну сеть облачные и dedicated сервера. Но этого не достаточно для организации pod-to-pod сети. Так как для этого нужно иметь возможность создавать маршруты как для облачных узлов так и для dedicated серверов.

Например, для ноды kube-master121-1 должны быть такие маршруты:
```
10.245.1.0/24 via 10.240.0.3 (cloud node kube-worker121-1) - может бы создан через API
10.245.2.0/24 via 10.240.0.4 (cloud node kube-worker121-2) - может бы создан через API
10.245.3.0/24 via 10.240.1.2 (dedicated node kube-worker121-10) - такой маршрут невозможно создать
```
![image](https://user-images.githubusercontent.com/22338503/111875069-a67b8500-89b9-11eb-9b5e-9d85fdbe38ee.png)


Для dedicated серверов нельзя настраивать маршрты:
![image](https://user-images.githubusercontent.com/22338503/111875090-c9a63480-89b9-11eb-8aa2-b349fadaa857.png)
Маршруты могут быть созданы только для облачных нод:
![image](https://user-images.githubusercontent.com/22338503/111875100-dfb3f500-89b9-11eb-8fee-e2b4654d2d4e.png)

Для dedicated серверов невозможно создать маршруты для сетей pod'ов, поэтому котроллер не поддерживает эту возможность.

Кроме того, вряд ли вам захочется подключать облачные сервера к выделенным через vswitch, поскольку по сравнению с публичной сетью, время задержки значительно ухудшается:  


ping из облачноой ноды (kube-master121-1) на dedicated ноду (kube-worker121-10) через публичную сеть:
```
$ ping 135.181.96.131
PING 135.181.96.131 (135.181.96.131) 56(84) bytes of data.
64 bytes from 135.181.96.131: icmp_seq=1 ttl=59 time=0.442 ms
64 bytes from 135.181.96.131: icmp_seq=2 ttl=59 time=0.372 ms
64 bytes from 135.181.96.131: icmp_seq=3 ttl=59 time=0.460 ms
64 bytes from 135.181.96.131: icmp_seq=4 ttl=59 time=0.539 ms
```
ping через vswitch:
```
$ ping 10.240.1.2
PING 10.240.1.2 (10.240.1.2) 56(84) bytes of data.
64 bytes from 10.240.1.2: icmp_seq=1 ttl=63 time=47.4 ms
64 bytes from 10.240.1.2: icmp_seq=2 ttl=63 time=47.0 ms
64 bytes from 10.240.1.2: icmp_seq=3 ttl=63 time=46.9 ms
64 bytes from 10.240.1.2: icmp_seq=4 ttl=63 time=46.9 ms
```
~0.5ms через публичную сеть против ~46.5ms через vswitch =(.

# Лицензия

Apache License, Version 2.0
