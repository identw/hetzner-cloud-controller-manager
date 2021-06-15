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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/identw/hetzner-cloud-controller-manager/internal/hcops"
	hrobot "github.com/nl2go/hrobot-go"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	hrobotUserENVVar                         = "HROBOT_USER"
	hrobotPassENVVar                         = "HROBOT_PASS"
	hrobotPeriodENVVar                       = "HROBOT_PERIOD"
	hcloudTokenENVVar                        = "HCLOUD_TOKEN"
	hcloudEndpointENVVar                     = "HCLOUD_ENDPOINT"
	hcloudNetworkENVVar                      = "HCLOUD_NETWORK"
	nodeNameENVVar                           = "NODE_NAME"
	providerNameENVVar                       = "PROVIDER_NAME"
	nameLabelTypeENVVar                      = "NAME_LABEL_TYPE"
	nameCloudNodeENVVar                      = "NAME_CLOUD_NODE"
	nameDedicatedNodeENVVar                  = "NAME_DEDICATED_NODE"
	enableSyncLabelsENVVar                   = "ENABLE_SYNC_LABELS"
	hcloudLoadBalancersEnabledENVVar         = "HCLOUD_LOAD_BALANCERS_ENABLED"
	hcloudLoadBalancersLocation              = "HCLOUD_LOAD_BALANCERS_LOCATION"
	hcloudLoadBalancersNetworkZone           = "HCLOUD_LOAD_BALANCERS_NETWORK_ZONE"
	hcloudLoadBalancersDisablePrivateIngress = "HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS"
	hcloudLoadBalancersUsePrivateIP          = "HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP"
	providerVersion                          = "v0.0.8"
)

var (
	hrobotPeriod     = 180
	enableSyncLabels = true
)

type commonClient struct {
	Hrobot    hrobot.RobotClient
	Hcloud    *hcloud.Client
	K8sClient *kubernetes.Clientset
}

type cloud struct {
	client       commonClient
	instances    cloudprovider.Instances
	zones        cloudprovider.Zones
	routes       cloudprovider.Routes
	loadBalancer *loadBalancers
	network      int
}

type config struct {
	ExcludeServers []string `json:"exclude_servers"`
}

type HrobotServer struct {
	ID     int
	Name   string
	Type   string
	Zone   string
	Region string
	IP     net.IP
}

var hrobotServers []HrobotServer

func readHrobotServers(hrobot hrobot.RobotClient) {
	go func() {
		for {
			servers, err := hrobot.ServerGetList()
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: get servers from hrobot: %v\n", err)
				time.Sleep(time.Duration(hrobotPeriod) * time.Second)
				continue
			}
			var hservers []HrobotServer
			for _, s := range servers {
				zone := strings.ToLower(strings.Split(s.Dc, "-")[0])
				server := HrobotServer{
					ID:     s.ServerNumber,
					Name:   s.ServerName,
					Type:   s.Product,
					Zone:   zone,
					Region: strings.ToLower(s.Dc),
					IP:     net.ParseIP(s.ServerIP),
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
)

func newCloud(configFile io.Reader) (cloudprovider.Interface, error) {
	const op = "hcloud/newCloud"

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

	if s := os.Getenv(providerNameENVVar); s != "" {
		hcops.ProviderName = s
	}
	if s := os.Getenv(nameLabelTypeENVVar); s != "" {
		hcops.NameLabelType = s
	}
	if s := os.Getenv(nameCloudNodeENVVar); s != "" {
		hcops.NameCloudNode = s
	}
	if s := os.Getenv(nameDedicatedNodeENVVar); s != "" {
		hcops.NameDedicatedNode = s
	}
	if s := os.Getenv(enableSyncLabelsENVVar); s == "false" {
		enableSyncLabels = false
	}

	var client commonClient
	client.Hcloud = hcloud.NewClient(opts...)
	client.Hrobot = hrobot.NewBasicAuthClient(user, pass)
	readHrobotServers(client.Hrobot)

	// k8s read config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Could not read k8s config: %s", err.Error())
	}
	client.K8sClient, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("k8s config problem: %s", err.Error())
	}

	// Load Balancer
	lbOpsDefaults, lbDisablePrivateIngress, err := loadBalancerDefaultsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lbOps := &hcops.LoadBalancerOps{
		LBClient:      &client.Hcloud.LoadBalancer,
		CertClient:    &client.Hcloud.Certificate,
		ActionClient:  &client.Hcloud.Action,
		NetworkClient: &client.Hcloud.Network,
		NetworkID:     0,
		Defaults:      lbOpsDefaults,
	}

	loadBalancers := newLoadBalancers(lbOps, &client.Hcloud.Action, lbDisablePrivateIngress)
	if os.Getenv(hcloudLoadBalancersEnabledENVVar) == "false" {
		loadBalancers = nil
	}

	fmt.Printf("Hetzner Cloud k8s cloud controller %s started\n", providerVersion)

	return &cloud{
		client:       client,
		zones:        newZones(client, nodeName),
		instances:    newInstances(client),
		loadBalancer: loadBalancers,
		routes:       nil,
		network:      0,
	}, nil
}

func (c *cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

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
	if c.loadBalancer == nil {
		return nil, false
	}
	return c.loadBalancer, true
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (c *cloud) ProviderName() string {
	return hcops.ProviderName
}

func (c *cloud) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nil, nil
}

func (c *cloud) HasClusterID() bool {
	return false
}

func loadBalancerDefaultsFromEnv() (hcops.LoadBalancerDefaults, bool, error) {
	defaults := hcops.LoadBalancerDefaults{
		Location:    os.Getenv(hcloudLoadBalancersLocation),
		NetworkZone: os.Getenv(hcloudLoadBalancersNetworkZone),
	}

	if defaults.Location != "" && defaults.NetworkZone != "" {
		return defaults, false, errors.New(
			"HCLOUD_LOAD_BALANCERS_LOCATION/HCLOUD_LOAD_BALANCERS_NETWORK_ZONE: Only one of these can be set")
	}

	disablePrivateIngress, err := getEnvBool(hcloudLoadBalancersDisablePrivateIngress)
	disablePrivateIngress = true
	err = nil
	if err != nil {
		return defaults, false, err
	}

	defaults.UsePrivateIP, err = getEnvBool(hcloudLoadBalancersUsePrivateIP)
	defaults.UsePrivateIP = false
	err = nil
	if err != nil {
		return defaults, false, err
	}

	return defaults, disablePrivateIngress, nil
}

// getEnvBool returns the boolean parsed from the environment variable with the given key and a potential error
// parsing the var. Returns false if the env var is unset.
func getEnvBool(key string) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %v", key, err)
	}

	return b, nil
}

func init() {
	cloudprovider.RegisterCloudProvider(hcops.ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	})
}
