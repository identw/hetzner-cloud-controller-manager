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
	"fmt"
	"os"
	"strconv"
	"strings"
	"regexp"
	"net"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

func getServerByName(ctx context.Context, c commonClient, name string) (server *hcloud.Server, err error) {
	// Find exclude servers
	for _, s := range cloudConfig.ExcludeServers {
		if exclude, _ := regexp.MatchString(s, name); exclude {
			return excludeServer, nil
		}
	}
	
	server, _, err = c.Hcloud.Server.GetByName(ctx, name)
	if err != nil {
		return
	}
	
	if server == nil {
		fmt.Fprintf(os.Stderr, "ERROR: Not found serverName: %v, in hcloud\n", name)
		// try hrobot find
		server, err = hrobotGetServerByName(c, name)
		if server == nil {
			fmt.Fprintf(os.Stderr, "ERROR: Not found serverName: %v, in hrobot\n", name)
			err = cloudprovider.InstanceNotFound
			return
		}
		return
	}
	return
}

func getServerByID(ctx context.Context, c commonClient, id int) (server *hcloud.Server, err error) {
	// Find exclude servers
	if id == excludeServer.ID {
		return excludeServer, nil
	}

	server, _, err = c.Hcloud.Server.GetByID(ctx, id)
	if err != nil {
		return
	}
	if server == nil {
		fmt.Fprintf(os.Stderr, "ERROR: Not found serverID: %v, in hcloud\n", id)
		server, err = hrobotGetServerByID(c, id)
		if server == nil {
			fmt.Fprintf(os.Stderr, "ERROR: Not found serverID: %v, in hrobot\n", id)
			err = cloudprovider.InstanceNotFound
			return
		}
	}
	return
}

func providerIDToServerID(providerID string) (id int, err error) {
	if providerID == strconv.Itoa(excludeServer.ID) {
		return excludeServer.ID, nil
	}
	providerPrefix := providerName + "://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		err = fmt.Errorf("ERROR: providerID should start with hetzner://: %s", providerID)
		return
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		err = fmt.Errorf("ERROR: missing server id in providerID: %s", providerID)
		return
	}

	id, err = strconv.Atoi(idString)
	return
}

func hrobotGetServerByName(c commonClient, name string) (*hcloud.Server, error) {
	servers, err := c.Hrobot.ServerGetList()
    if err != nil {
        return nil, err
	}
	for _, s := range servers {
		if s.ServerName == name {
			zone := strings.ToLower(strings.Split(s.Dc, "-")[0])
			ip := net.ParseIP(s.ServerIP)
			server := &hcloud.Server{ 
				ID: s.ServerNumber,
				Name: s.ServerName,
				PublicNet: hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: ip}},
				ServerType: &hcloud.ServerType{Name: s.Product},
				Status: hcloud.ServerStatus("running"),
				Datacenter: &hcloud.Datacenter{ Location: &hcloud.Location{Name: zone}, Name: strings.ToLower(s.Dc) },
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}

func hrobotGetServerByID(c commonClient, id int) (*hcloud.Server, error) {
	servers, err := c.Hrobot.ServerGetList()
    if err != nil {
        return nil, err
	}
	for _, s := range servers {
		if s.ServerNumber == id {
			zone := strings.ToLower(strings.Split(s.Dc, "-")[0])
			ip := net.ParseIP(s.ServerIP)
			server := &hcloud.Server{ 
				ID: s.ServerNumber,
				Name: s.ServerName,
				PublicNet: hcloud.ServerPublicNet{IPv4: hcloud.ServerPublicNetIPv4{IP: ip}},
				ServerType: &hcloud.ServerType{Name: s.Product},
				Status: hcloud.ServerStatus("running"),
				Datacenter: &hcloud.Datacenter{ Location: &hcloud.Location{Name: zone}, Name: strings.ToLower(s.Dc) },
			}
			return server, nil
		}
	}
	// server not found
	return nil, nil
}