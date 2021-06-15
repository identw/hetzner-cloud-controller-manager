/*
Copyright 2018 Hetzner Cloud GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/identw/hetzner-cloud-controller-manager/internal/hcops"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

func getServerByName(ctx context.Context, c commonClient, name string) (server *hcloud.Server, err error) {
	// Find exclude servers
	for _, s := range cloudConfig.ExcludeServers {
		if exclude, _ := regexp.MatchString(s, name); exclude {
			return hcops.ExcludeServer, nil
		}
	}

	server, _, err = c.Hcloud.Server.GetByName(ctx, name)
	if err != nil {
		return
	}

	if server != nil {
		syncLabels(c.K8sClient, server)
		addTypeLabel(c.K8sClient, server.Name, hcops.NameCloudNode)
	}

	if server == nil {
		// try hrobot find
		server, err = hrobotGetServerByName(name)
		if server == nil {
			fmt.Fprintf(os.Stderr, "ERROR: Not found serverName: %v, in hcloud and hrobot\n", name)
			err = cloudprovider.InstanceNotFound
			return
		}
		addTypeLabel(c.K8sClient, server.Name, hcops.NameDedicatedNode)
		return
	}
	return
}

func getServerByID(ctx context.Context, c commonClient, id int) (server *hcloud.Server, err error) {
	// Find exclude servers
	if id == hcops.ExcludeServer.ID {
		return hcops.ExcludeServer, nil
	}

	server, _, err = c.Hcloud.Server.GetByID(ctx, id)
	if err != nil {
		return
	}

	if server != nil {
		syncLabels(c.K8sClient, server)
		addTypeLabel(c.K8sClient, server.Name, hcops.NameCloudNode)
	}
	if server == nil {
		server, err = hrobotGetServerByID(id)
		if server == nil {
			fmt.Fprintf(os.Stderr, "ERROR: Not found serverID: %v, in hcloud and hrobot\n", id)
			err = cloudprovider.InstanceNotFound
			return
		}
		addTypeLabel(c.K8sClient, server.Name, hcops.NameDedicatedNode)
	}
	return
}

func hrobotGetServerByName(name string) (*hcloud.Server, error) {
	for _, s := range hrobotServers {
		if s.Name == name {
			server := &hcloud.Server{
				ID:         s.ID,
				Name:       s.Name,
				PublicNet:  hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: s.IP}},
				ServerType: &hcloud.ServerType{Name: s.Type},
				Status:     hcloud.ServerStatus("running"),
				Datacenter: &hcloud.Datacenter{Location: &hcloud.Location{Name: s.Zone}, Name: s.Region},
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}

func hrobotGetServerByID(id int) (*hcloud.Server, error) {
	for _, s := range hrobotServers {
		if s.ID == id {
			server := &hcloud.Server{
				ID:         s.ID,
				Name:       s.Name,
				PublicNet:  hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: s.IP}},
				ServerType: &hcloud.ServerType{Name: s.Type},
				Status:     hcloud.ServerStatus("running"),
				Datacenter: &hcloud.Datacenter{Location: &hcloud.Location{Name: s.Zone}, Name: s.Region},
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}

// Sync Labels from cloud node to k8s node
func syncLabels(k8sClient *kubernetes.Clientset, server *hcloud.Server) {
	if !enableSyncLabels {
		return
	}
	node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), server.Name, metav1.GetOptions{})
	if err == nil {
		// Annotation in which the labels applied from the last time are stored
		const annotation = "ccm.hetzner.com/last-applied-labels"
		// flag exist annotation
		ccma := false
		// flag changed
		changed := false
		if _, ok := node.ObjectMeta.Annotations[annotation]; ok {
			ccma = true
		}
		// If the annotation exists, then we look for labels that have been removed from the server and
		// remove them from the k8s node
		if ccma {
			// Previous labels from annotations
			var pl map[string]string
			if err := json.Unmarshal([]byte(node.ObjectMeta.Annotations[annotation]), &pl); err != nil {
				klog.Errorf("Unmarshal error annotatios: %s, error: %s", annotation, err)
			}
			for k := range pl {
				if _, ok := server.Labels[k]; !ok {
					changed = true
					delete(node.ObjectMeta.Labels, k)
				}
			}
		}
		sl, _ := json.Marshal(server.Labels)
		node.ObjectMeta.Annotations[annotation] = string(sl)
		// sync labels
		for k, v := range server.Labels {
			if _, ok := node.ObjectMeta.Labels[k]; !ok {
				changed = true
				node.ObjectMeta.Labels[k] = v
			}
			if node.ObjectMeta.Labels[k] != v {
				changed = true
				node.ObjectMeta.Labels[k] = v
			}
		}

		if changed {
			k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		}
	}
}

func addTypeLabel(k8sClient *kubernetes.Clientset, name string, typeNode string) {
	node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		if _, ok := node.ObjectMeta.Labels[hcops.NameLabelType]; !ok {
			node.ObjectMeta.Labels[hcops.NameLabelType] = typeNode
		}
		if node.ObjectMeta.Labels[hcops.NameLabelType] != typeNode {
			node.ObjectMeta.Labels[hcops.NameLabelType] = typeNode
		}
		k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	}
}
