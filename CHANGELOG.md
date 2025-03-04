Changelog
=========
v0.0.11
------
* support kube-api v1.32.* (client-go v0.32.2, cloud-provider v0.32.2)
* add support annotation `load-balancer.hetzner.cloud/external-dns-hostname` - specifies the hostname of the Load Balancer. This will be used as service.status.loadBalancer.ingress address instead of the Load Balancer IP addresses if specified. And it add two annotations for external-dns: `external-dns.alpha.kubernetes.io/target: <ipv4-address>,<ipv6-address>` and `external-dns.alpha.kubernetes.io/hostname: the value from load-balancer.hetzner.cloud/external-dns-hostname`. This is useful and convenient for automatically create DNS record (like aws nlb).

v0.0.10
------
* kube-api v1.24.8, cluent-go v0.24.8, support kubernetes v1.24.x

v0.0.9
------
 * Add labels `instance.hetzner.cloud/provided-by` and `instance.hetzner.cloud/is-root-server` for nodes

v0.0.8
------
 * Fixed a bug where a dedicated server could be removed from the cluster if it was unavailable https://robot-ws.your-server.de/server

v0.0.7
------
 * Add support LoadBalancer

v0.0.6
------
 * Synchronization of labels from cloud servers to k8s labels of work nodes. Removing the label from the cloud server also removes it on the k8s worker node 
 * Adding a label (default: `node.hetzner.com/type`) to separate cloud and dedicated servers
 * env `PROVIDER_NAME` for change provderID prefix
 * update k8s libraries to 1.19.8
 * update github workflows. Also push images to ghcr.io
 
v0.0.5
------
 * Support kubernetes v1.19.x
 * update k8s libraries to 1.19.3

v0.0.4
------
 * Fix problem with requests rate limit for Hrobot API (200 requests per hour)
 * Servers from hrobot api are now cached in memory and updated with the period `HROBOT_PERIOD` seconds

v0.0.3 
------
 * add capability: exclude the removal of nodes that belong to other providers

v0.0.2
------
* Fix bug: invalid memory address or nil pointer dereference if server not found in hrobot

v0.0.1
------
* Initial
