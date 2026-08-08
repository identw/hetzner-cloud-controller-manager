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
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/identw/hetzner-cloud-controller-manager/internal/hcops"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

func getServerByName(ctx context.Context, c commonClient, name string) (server *hcloud.Server, err error) {
	// Find exclude servers
	if cloudConfig != nil {
		for _, s := range cloudConfig.ExcludeServers {
			if exclude, _ := regexp.MatchString(s, name); exclude {
				return hcops.ExcludeServer, nil
			}
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

func getServerByID(ctx context.Context, c commonClient, id int64) (server *hcloud.Server, err error) {
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
				ID:         int64(s.ID),
				Name:       s.Name,
				PublicNet:  hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: s.IP}},
				ServerType: &hcloud.ServerType{Name: s.Type},
				Status:     hcloud.ServerStatus("running"),
				Location:   &hcloud.Location{Name: s.Zone},
				Labels:     map[string]string{hcops.RobotDatacenterLabel: s.Region},
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}

func hrobotGetServerByID(id int64) (*hcloud.Server, error) {
	for _, s := range hrobotServers {
		if int64(s.ID) == id {
			server := &hcloud.Server{
				ID:         int64(s.ID),
				Name:       s.Name,
				PublicNet:  hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: s.IP}},
				ServerType: &hcloud.ServerType{Name: s.Type},
				Status:     hcloud.ServerStatus("running"),
				Location:   &hcloud.Location{Name: s.Zone},
				Labels:     map[string]string{hcops.RobotDatacenterLabel: s.Region},
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}

// isK8sLabelChar reports whether r is allowed in a Kubernetes label name/value
// (alphanumeric ASCII, '-', '_' or '.').
func isK8sLabelChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.'
}

// normalizeK8sLabelPart replaces characters that are not valid in Kubernetes
// label names/values with '-', trims leading/trailing separators and enforces
// the maximum length.
func normalizeK8sLabelPart(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isK8sLabelChar(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}

	out := strings.Trim(b.String(), "-_.")
	if len(out) > validation.LabelValueMaxLength {
		out = out[:validation.LabelValueMaxLength]
		out = strings.TrimRight(out, "-_.")
	}
	return out
}

// normalizeK8sLabelValue makes a string a valid Kubernetes label value.
// Invalid characters are replaced with '-', then leading/trailing separators
// are trimmed. Empty result is valid (Kubernetes allows empty label values).
func normalizeK8sLabelValue(value string) string {
	return normalizeK8sLabelPart(value)
}

// normalizeK8sLabelKey makes a string a valid Kubernetes label key (qualified name).
// If the key already validates, it is returned unchanged. Otherwise invalid
// characters in the name part are replaced similarly to values.
func normalizeK8sLabelKey(key string) string {
	if len(validation.IsQualifiedName(key)) == 0 {
		return key
	}

	prefix, name := "", key
	if i := strings.LastIndex(key, "/"); i >= 0 {
		prefix, name = key[:i], key[i+1:]
	}

	name = normalizeK8sLabelPart(name)
	if name == "" {
		return ""
	}

	if prefix != "" {
		if len(validation.IsDNS1123Subdomain(prefix)) != 0 {
			return ""
		}
		return prefix + "/" + name
	}
	return name
}

// Sync Labels from cloud node to k8s node
func syncLabels(k8sClient *kubernetes.Clientset, server *hcloud.Server) {
	if !enableSyncLabels || k8sClient == nil {
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

		// Normalize Hetzner labels to valid Kubernetes label keys/values.
		normalized := make(map[string]string, len(server.Labels))
		for k, v := range server.Labels {
			nk := normalizeK8sLabelKey(k)
			if nk == "" {
				klog.Warningf("skipping invalid label key %q from server %s", k, server.Name)
				continue
			}
			nv := normalizeK8sLabelValue(v)
			if nk != k || nv != v {
				klog.Infof("normalized label %q=%q -> %q=%q for server %s", k, v, nk, nv, server.Name)
			}
			normalized[nk] = nv
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
				if _, ok := normalized[k]; !ok {
					changed = true
					delete(node.ObjectMeta.Labels, k)
				}
			}
		}
		sl, _ := json.Marshal(normalized)
		node.ObjectMeta.Annotations[annotation] = string(sl)
		// sync labels
		for k, v := range normalized {
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
	if k8sClient == nil {
		return
	}
	node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		for k, v := range hcops.TypeLabels[typeNode] {
			if _, ok := node.ObjectMeta.Labels[k]; !ok {
				node.ObjectMeta.Labels[k] = v
			}
			if node.ObjectMeta.Labels[k] != v {
				node.ObjectMeta.Labels[k] = v
			}
		}

		k8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	}
}
