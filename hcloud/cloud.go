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
	"fmt"
	"io"
	"os"
	"encoding/json"
	"io/ioutil"
	"time"
	"strings"
	"strconv"
	"net"

	"github.com/hetznercloud/hcloud-go/hcloud"
	hrobot "github.com/nl2go/hrobot-go"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	hrobotUserENVVar     = "HROBOT_USER"
	hrobotPassENVVar     = "HROBOT_PASS"
	hrobotPeriodENVVar   = "HROBOT_PERIOD"
	hcloudTokenENVVar    = "HCLOUD_TOKEN"
	hcloudEndpointENVVar = "HCLOUD_ENDPOINT"
	hcloudNetworkENVVar  = "HCLOUD_NETWORK"
	nodeNameENVVar       = "NODE_NAME"
	providerName         = "hetzner"
	providerVersion      = "v0.0.4"
)

var (
	hrobotPeriod = 180
)

type commonClient struct {
	Hrobot hrobot.RobotClient
	Hcloud *hcloud.Client
}

type cloud struct {
	client    commonClient
	instances cloudprovider.Instances
	zones     cloudprovider.Zones
	routes    cloudprovider.Routes
	network   int
}

type config struct {
	ExcludeServers []string                     `json:"exclude_servers"`
}

type HrobotServer struct {
	ID int
	Name string
	Type string
	Zone string
	Region string
	IP net.IP
}

var hrobotServers []HrobotServer

func readHrobotServers(hrobot hrobot.RobotClient) {
	go func() {
			for {
					servers, err := hrobot.ServerGetList()
					if err != nil {
						fmt.Fprintf(os.Stderr, "ERROR: get servers from hrobot: %v\n", err)
					}
					var hservers []HrobotServer
					for _, s := range servers {
						zone := strings.ToLower(strings.Split(s.Dc, "-")[0])
						server := HrobotServer{
							ID: s.ServerNumber,
							Name: s.ServerName,
							Type: s.Product,
							Zone: zone,
							Region: strings.ToLower(s.Dc),
							IP: net.ParseIP(s.ServerIP),
						}
						hservers = append(hservers, server)
					}
					hrobotServers = hservers
					time.Sleep(time.Duration(hrobotPeriod) * time.Second)
			}
	}()
}

var (
	cloudConfig *config
	excludeServer = &hcloud.Server{ 
		ID: 999999,
		ServerType: &hcloud.ServerType{Name: "exclude"},
		Status: hcloud.ServerStatus("running"),
		Datacenter: &hcloud.Datacenter{
			Location: &hcloud.Location{
				Name: "exclude",
			}, 
			Name: "exclude",
		},
	}
)

func newCloud(configFile io.Reader) (cloudprovider.Interface, error) {
	cfg := &config{}
	if configFile != nil {
        body, err := ioutil.ReadAll(configFile)
        if err != nil {
            return nil, err
        }
        err = json.Unmarshal(body, cfg)
        if err != nil {
            return nil, err
        }
	}

	cloudConfig = cfg
	if len(cloudConfig.ExcludeServers) == 0 {
		cloudConfig.ExcludeServers = make([]string, 0, 0)
	}

	token := os.Getenv(hcloudTokenENVVar)
	if token == "" {
		return nil, fmt.Errorf("environment variable %q is required", hcloudTokenENVVar)
	}
	nodeName := os.Getenv(nodeNameENVVar)
	if nodeName == "" {
		return nil, fmt.Errorf("environment variable %q is required", nodeNameENVVar)
	}

	// network := os.Getenv(hcloudNetworkENVVar)

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-cloud-controller", providerVersion),
	}
	if endpoint := os.Getenv(hcloudEndpointENVVar); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}

	// hetzner robot get auth from env
	user := os.Getenv(hrobotUserENVVar)
	if user == "" {
		return nil, fmt.Errorf("environment variable %q is required", hrobotUserENVVar)
	}
	pass := os.Getenv(hrobotPassENVVar)
	if pass == "" {
		return nil, fmt.Errorf("environment variable %q is required", hrobotPassENVVar)
	}
	period := os.Getenv(hrobotPeriodENVVar)
	if period == "" {
		hrobotPeriod = 180
	} else {
		hrobotPeriod, _ = strconv.Atoi(period)
	}

	var client commonClient
	client.Hcloud = hcloud.NewClient(opts...)
	client.Hrobot = hrobot.NewBasicAuthClient(user, pass)
	readHrobotServers(client.Hrobot)

	return &cloud{
		client:    client,
		zones:     newZones(client, nodeName),
		instances: newInstances(client),
		routes:    nil,
		network:   0,
	}, nil
}

func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return c.instances, true
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return nil, false
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return c.zones, true
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (c *cloud) ProviderName() string {
	return providerName
}

func (c *cloud) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nil, nil
}

func (c *cloud) HasClusterID() bool {
	return false
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	})
}
