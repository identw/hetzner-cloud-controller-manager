Changelog
=========

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
