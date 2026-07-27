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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/identw/hetzner-cloud-controller-manager/internal/hcops"
)

type testEnv struct {
	Server *httptest.Server
	Mux    *http.ServeMux
	Client *hcloud.Client
}

func (env *testEnv) Teardown() {
	env.Server.Close()
	env.Server = nil
	env.Mux = nil
	env.Client = nil
}

func newTestEnv() testEnv {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithToken("token"),
		hcloud.WithPollOpts(hcloud.PollOpts{
			BackoffFunc: hcloud.ConstantBackoff(0),
		}),
		hcloud.WithRetryOpts(hcloud.RetryOpts{
			BackoffFunc: hcloud.ConstantBackoff(0),
			MaxRetries:  5,
		}),
	)
	return testEnv{
		Server: server,
		Mux:    mux,
		Client: client,
	}
}

func TestCloudInterfaces(t *testing.T) {
	c := &cloud{
		instances:    newInstances(commonClient{}),
		zones:        newZones(commonClient{}, "node"),
		loadBalancer: newLoadBalancers(&mockLoadBalancerOps{}, nil, true, commonClient{}),
	}

	if _, ok := c.Instances(); !ok {
		t.Error("Instances should be supported")
	}
	if _, ok := c.Zones(); !ok {
		t.Error("Zones should be supported")
	}
	if _, ok := c.LoadBalancer(); !ok {
		t.Error("LoadBalancer should be supported when configured")
	}
	if _, ok := c.Clusters(); ok {
		t.Error("Clusters should not be supported")
	}
	if _, ok := c.Routes(); ok {
		t.Error("Routes should not be supported")
	}
	if c.HasClusterID() {
		t.Error("HasClusterID should be false")
	}
	if c.ProviderName() != hcops.ProviderName {
		t.Errorf("ProviderName = %q", c.ProviderName())
	}
}

func TestCloudLoadBalancerDisabled(t *testing.T) {
	c := &cloud{}
	if _, ok := c.LoadBalancer(); ok {
		t.Error("LoadBalancer should be unsupported when nil")
	}
}

func TestNewCloud_RequiresEnv(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "")
	t.Setenv("NODE_NAME", "")
	t.Setenv("HROBOT_USER", "")
	t.Setenv("HROBOT_PASS", "")

	_, err := newCloud(nil)
	if err == nil {
		t.Fatal("expected error when required env is missing")
	}
}

func TestNewCloud_RequiresRobotCredentials(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "token")
	t.Setenv("NODE_NAME", "node")
	_ = os.Unsetenv("HROBOT_USER")
	_ = os.Unsetenv("HROBOT_PASS")

	_, err := newCloud(nil)
	if err == nil {
		t.Fatal("expected error when robot credentials missing")
	}
}
