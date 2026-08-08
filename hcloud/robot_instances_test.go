package hcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	hrobotmodels "github.com/nl2go/hrobot-go/models"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type mockRobotClient struct {
	servers []hrobotmodels.Server
	err     error
}

func (m *mockRobotClient) SetBaseURL(string)                          {}
func (m *mockRobotClient) SetUserAgent(string)                        {}
func (m *mockRobotClient) GetVersion() string                         { return "test" }
func (m *mockRobotClient) ServerGetList() ([]hrobotmodels.Server, error) {
	return m.servers, m.err
}
func (m *mockRobotClient) ServerGet(string) (*hrobotmodels.Server, error) { return nil, nil }
func (m *mockRobotClient) ServerSetName(string, *hrobotmodels.ServerSetNameInput) (*hrobotmodels.Server, error) {
	return nil, nil
}
func (m *mockRobotClient) ServerReverse(string) (*hrobotmodels.Cancellation, error) { return nil, nil }
func (m *mockRobotClient) KeyGetList() ([]hrobotmodels.Key, error)                   { return nil, nil }
func (m *mockRobotClient) IPGetList() ([]hrobotmodels.IP, error)                     { return nil, nil }
func (m *mockRobotClient) RDnsGetList() ([]hrobotmodels.Rdns, error)                 { return nil, nil }
func (m *mockRobotClient) RDnsGet(string) (*hrobotmodels.Rdns, error)                { return nil, nil }
func (m *mockRobotClient) BootRescueGet(string) (*hrobotmodels.Rescue, error)        { return nil, nil }
func (m *mockRobotClient) BootRescueSet(string, *hrobotmodels.RescueSetInput) (*hrobotmodels.Rescue, error) {
	return nil, nil
}
func (m *mockRobotClient) ResetGet(string) (*hrobotmodels.Reset, error) { return nil, nil }
func (m *mockRobotClient) ResetSet(string, *hrobotmodels.ResetSetInput) (*hrobotmodels.ResetPost, error) {
	return nil, nil
}
func (m *mockRobotClient) FailoverGetList() ([]hrobotmodels.Failover, error)  { return nil, nil }
func (m *mockRobotClient) FailoverGet(string) (*hrobotmodels.Failover, error) { return nil, nil }

func TestMapHrobotServers(t *testing.T) {
	got := mapHrobotServers([]hrobotmodels.Server{{
		ServerNumber: 55,
		ServerName:   "robot-1",
		Product:      "AX41",
		Dc:           "FSN1-DC8",
		ServerIP:     "203.0.113.10",
	}})
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	s := got[0]
	if s.ID != 55 || s.Name != "robot-1" || s.Type != "AX41" || s.Zone != "fsn1" || s.Region != "fsn1-dc8" {
		t.Fatalf("unexpected mapping: %#v", s)
	}
	if !s.IP.Equal(net.ParseIP("203.0.113.10")) {
		t.Fatalf("unexpected ip: %v", s.IP)
	}
}

func TestSyncHrobotCache_WithMockRobotAPI(t *testing.T) {
	orig := hrobotServers
	t.Cleanup(func() { hrobotServers = orig })

	robot := &mockRobotClient{servers: []hrobotmodels.Server{{
		ServerNumber: 77,
		ServerName:   "dedicated-a",
		Product:      "EX44",
		Dc:           "NBG1-DC3",
		ServerIP:     "198.51.100.7",
	}}}
	if err := syncHrobotCache(robot); err != nil {
		t.Fatalf("syncHrobotCache: %v", err)
	}
	if len(hrobotServers) != 1 || hrobotServers[0].ID != 77 || hrobotServers[0].Zone != "nbg1" {
		t.Fatalf("unexpected cache: %#v", hrobotServers)
	}

	robot.err = errors.New("robot down")
	if err := syncHrobotCache(robot); err == nil {
		t.Fatal("expected error")
	}
}

func handleNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(schema.ErrorResponse{
		Error: schema.Error{Code: string(hcloud.ErrorCodeNotFound), Message: "not found"},
	})
}

func TestInstances_RobotFallbackByNameAndID(t *testing.T) {
	origServers := hrobotServers
	origConfig := cloudConfig
	t.Cleanup(func() {
		hrobotServers = origServers
		cloudConfig = origConfig
	})
	cloudConfig = &config{}
	hrobotServers = []HrobotServer{{
		ID:     321,
		Name:   "robot-node",
		Type:   "AX41",
		Zone:   "fsn1",
		Region: "fsn1-dc8",
		IP:     net.ParseIP("203.0.113.21"),
	}}

	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(schema.ServerListResponse{Servers: nil})
	})
	env.Mux.HandleFunc("/servers/321", handleNotFound)

	instances := newInstances(commonClient{Hcloud: env.Client})

	addrs, err := instances.NodeAddresses(context.Background(), "robot-node")
	if err != nil {
		t.Fatalf("NodeAddresses: %v", err)
	}
	if len(addrs) != 2 || addrs[1].Address != "203.0.113.21" {
		t.Fatalf("unexpected addresses: %#v", addrs)
	}

	typ, err := instances.InstanceTypeByProviderID(context.Background(), "hetzner://321")
	if err != nil || typ != "AX41" {
		t.Fatalf("type=%q err=%v", typ, err)
	}

	zone, err := newZones(commonClient{Hcloud: env.Client}, "x").GetZoneByProviderID(context.Background(), "hetzner://321")
	if err != nil {
		t.Fatalf("GetZoneByProviderID: %v", err)
	}
	if zone.Region != "fsn1" || zone.FailureDomain != "fsn1-dc8" {
		t.Fatalf("unexpected zone: %#v", zone)
	}
}

func TestInstances_RobotInstanceTypeNormalized(t *testing.T) {
	origServers := hrobotServers
	origConfig := cloudConfig
	t.Cleanup(func() {
		hrobotServers = origServers
		cloudConfig = origConfig
	})
	cloudConfig = &config{}
	hrobotServers = []HrobotServer{{
		ID:     10423,
		Name:   "kube-worker104-23",
		Type:   "Server Auction",
		Zone:   "fsn1",
		Region: "fsn1-dc14",
		IP:     net.ParseIP("203.0.113.104"),
	}}

	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(schema.ServerListResponse{Servers: nil})
	})
	env.Mux.HandleFunc("/servers/10423", handleNotFound)

	instances := newInstances(commonClient{Hcloud: env.Client})

	typ, err := instances.InstanceType(context.Background(), "kube-worker104-23")
	if err != nil {
		t.Fatalf("InstanceType: %v", err)
	}
	if typ != "Server-Auction" {
		t.Fatalf("got type %q, want Server-Auction", typ)
	}

	typ, err = instances.InstanceTypeByProviderID(context.Background(), "hetzner://10423")
	if err != nil {
		t.Fatalf("InstanceTypeByProviderID: %v", err)
	}
	if typ != "Server-Auction" {
		t.Fatalf("got type %q, want Server-Auction", typ)
	}
}

func TestInstances_ExcludeServerByName(t *testing.T) {
	origConfig := cloudConfig
	t.Cleanup(func() { cloudConfig = origConfig })
	cloudConfig = &config{ExcludeServers: []string{`^skip-.*`}}

	env := newTestEnv()
	defer env.Teardown()
	instances := newInstances(commonClient{Hcloud: env.Client})

	typ, err := instances.InstanceType(context.Background(), "skip-me")
	if err != nil {
		t.Fatalf("InstanceType: %v", err)
	}
	if typ != "exclude" {
		t.Fatalf("got type %q", typ)
	}

	exists, err := instances.InstanceExistsByProviderID(context.Background(), "999999")
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}

func TestInstances_NotFound(t *testing.T) {
	origServers := hrobotServers
	origConfig := cloudConfig
	t.Cleanup(func() {
		hrobotServers = origServers
		cloudConfig = origConfig
	})
	cloudConfig = &config{}
	hrobotServers = nil

	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(schema.ServerListResponse{Servers: nil})
	})

	instances := newInstances(commonClient{Hcloud: env.Client})
	_, err := instances.NodeAddresses(context.Background(), "missing")
	if !errors.Is(err, cloudprovider.InstanceNotFound) {
		t.Fatalf("expected InstanceNotFound, got %v", err)
	}
}

func TestNodeAddresses_WithPrivateNetwork(t *testing.T) {
	t.Setenv(hcloudNetworkENVVar, "1")
	t.Cleanup(func() { _ = os.Unsetenv(hcloudNetworkENVVar) })

	env := newTestEnv()
	defer env.Teardown()
	env.Mux.HandleFunc("/servers/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(schema.ServerGetResponse{
			Server: schema.Server{
				ID:   1,
				Name: "node1",
				PublicNet: schema.ServerPublicNet{
					IPv4: schema.ServerPublicNetIPv4{IP: "203.0.113.1"},
				},
				PrivateNet: []schema.ServerPrivateNet{
					{Network: 1, IP: "10.0.0.5"},
				},
			},
		})
	})
	env.Mux.HandleFunc("/networks/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(schema.NetworkGetResponse{
			Network: schema.Network{ID: 1, Name: "net1", IPRange: "10.0.0.0/16"},
		})
	})

	instances := newInstances(commonClient{Hcloud: env.Client})
	addrs, err := instances.NodeAddressesByProviderID(context.Background(), "hetzner://1")
	if err != nil {
		t.Fatalf("NodeAddressesByProviderID: %v", err)
	}
	if len(addrs) != 3 {
		t.Fatalf("expected 3 addresses, got %#v", addrs)
	}
	if addrs[2].Type != v1.NodeInternalIP || addrs[2].Address != "10.0.0.5" {
		t.Fatalf("unexpected internal address: %#v", addrs[2])
	}
}

func TestFilterNodes_ExcludesConfiguredServers(t *testing.T) {
	origConfig := cloudConfig
	t.Cleanup(func() { cloudConfig = origConfig })
	cloudConfig = &config{ExcludeServers: []string{`master-.*`}}

	nodes := []*v1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "worker-1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "master-1"}},
	}
	got := filterNodes(nodes)
	if len(got) != 1 || got[0].Name != "worker-1" {
		t.Fatalf("unexpected nodes: %#v", got)
	}
}

func TestNodeAddresses_IPHandling(t *testing.T) {
	instances := newInstances(commonClient{})

	t.Run("skips unspecified ipv4 and derives ipv6 host", func(t *testing.T) {
		server := &hcloud.Server{
			Name: "dual",
			PublicNet: hcloud.ServerPublicNet{
				IPv4: hcloud.ServerPublicNetIPv4{}, // unspecified
				IPv6: hcloud.ServerPublicNetIPv6{IP: net.ParseIP("2a01:4f9:c010:c081::")},
			},
		}
		addrs, err := instances.nodeAddresses(context.Background(), server)
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) != 2 {
			t.Fatalf("got %#v", addrs)
		}
		if addrs[0].Type != v1.NodeHostName || addrs[1].Address != "2a01:4f9:c010:c081::1" {
			t.Fatalf("unexpected addresses: %#v", addrs)
		}
	})

	t.Run("nil ipv4 does not become <nil> string", func(t *testing.T) {
		server := &hcloud.Server{Name: "noip"}
		addrs, err := instances.nodeAddresses(context.Background(), server)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range addrs {
			if a.Address == "<nil>" {
				t.Fatalf("found <nil> address: %#v", addrs)
			}
		}
		if len(addrs) != 1 || addrs[0].Type != v1.NodeHostName {
			t.Fatalf("expected only hostname, got %#v", addrs)
		}
	})
}

func TestIPStringHelper(t *testing.T) {
	if got := ipString(nil); got != "" {
		t.Fatalf("nil => %q", got)
	}
	if got := ipString(net.IPv4zero); got != "" {
		t.Fatalf("unspecified => %q", got)
	}
	if got := ipString(net.ParseIP("203.0.113.10")); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
}
