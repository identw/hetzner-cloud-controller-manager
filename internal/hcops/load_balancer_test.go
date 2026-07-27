package hcops

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadBalancerOps_GetByK8SServiceUID(t *testing.T) {
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{UID: "uid-1", Name: "svc"}}

	t.Run("found", func(t *testing.T) {
		lbClient := &mockLBClient{
			allWithOptsFn: func(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error) {
				if opts.LabelSelector != LabelServiceUID+"=uid-1" {
					t.Fatalf("unexpected selector: %s", opts.LabelSelector)
				}
				return []*hcloud.LoadBalancer{{ID: 10, Name: "lb"}}, nil
			},
		}
		ops := &LoadBalancerOps{LBClient: lbClient}
		lb, err := ops.GetByK8SServiceUID(context.Background(), svc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lb.ID != 10 {
			t.Fatalf("got id=%d", lb.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ops := &LoadBalancerOps{LBClient: &mockLBClient{}}
		_, err := ops.GetByK8SServiceUID(context.Background(), svc)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("non unique", func(t *testing.T) {
		ops := &LoadBalancerOps{LBClient: &mockLBClient{
			allWithOptsFn: func(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error) {
				return []*hcloud.LoadBalancer{{ID: 1}, {ID: 2}}, nil
			},
		}}
		_, err := ops.GetByK8SServiceUID(context.Background(), svc)
		if !errors.Is(err, ErrNonUniqueResult) {
			t.Fatalf("expected ErrNonUniqueResult, got %v", err)
		}
	})
}

func TestLoadBalancerOps_GetByNameAndID(t *testing.T) {
	ops := &LoadBalancerOps{LBClient: &mockLBClient{
		getByNameFn: func(ctx context.Context, name string) (*hcloud.LoadBalancer, *hcloud.Response, error) {
			if name == "missing" {
				return nil, nil, nil
			}
			return &hcloud.LoadBalancer{ID: 5, Name: name}, nil, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*hcloud.LoadBalancer, *hcloud.Response, error) {
			if id == 404 {
				return nil, nil, nil
			}
			return &hcloud.LoadBalancer{ID: id}, nil, nil
		},
	}}

	lb, err := ops.GetByName(context.Background(), "lb-a")
	if err != nil || lb.ID != 5 {
		t.Fatalf("GetByName: lb=%v err=%v", lb, err)
	}
	_, err = ops.GetByName(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	lb, err = ops.GetByID(context.Background(), 9)
	if err != nil || lb.ID != 9 {
		t.Fatalf("GetByID: lb=%v err=%v", lb, err)
	}
	_, err = ops.GetByID(context.Background(), 404)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLoadBalancerOps_Create(t *testing.T) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:  "svc-uid",
			Name: "svc",
			Annotations: map[string]string{
				"load-balancer.hetzner.cloud/location": "fsn1",
			},
		},
	}

	var createdOpts hcloud.LoadBalancerCreateOpts
	lbClient := &mockLBClient{
		createFn: func(ctx context.Context, opts hcloud.LoadBalancerCreateOpts) (hcloud.LoadBalancerCreateResult, *hcloud.Response, error) {
			createdOpts = opts
			return hcloud.LoadBalancerCreateResult{
				LoadBalancer: &hcloud.LoadBalancer{ID: 100, Name: opts.Name},
				Action:       &hcloud.Action{ID: 1},
			}, nil, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*hcloud.LoadBalancer, *hcloud.Response, error) {
			return &hcloud.LoadBalancer{ID: id, Name: "a3a2b3c"}, nil, nil
		},
	}

	ops := &LoadBalancerOps{
		LBClient:     lbClient,
		ActionClient: &mockActionClient{},
		Defaults:     LoadBalancerDefaults{Location: "nbg1"},
	}

	lb, err := ops.Create(context.Background(), "a3a2b3c", svc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if lb.ID != 100 {
		t.Fatalf("got id=%d", lb.ID)
	}
	if createdOpts.Location == nil || createdOpts.Location.Name != "fsn1" {
		t.Fatalf("expected location from annotation, got %#v", createdOpts.Location)
	}
	if createdOpts.Labels[LabelServiceUID] != "svc-uid" {
		t.Fatalf("missing service uid label: %#v", createdOpts.Labels)
	}
}

func TestLoadBalancerOps_Create_RequiresLocationOrZone(t *testing.T) {
	ops := &LoadBalancerOps{
		LBClient:     &mockLBClient{},
		ActionClient: &mockActionClient{},
	}
	_, err := ops.Create(context.Background(), "lb", &v1.Service{ObjectMeta: metav1.ObjectMeta{UID: "u"}})
	if err == nil {
		t.Fatal("expected error when location/network zone missing")
	}
}

func TestLoadBalancerOps_Delete(t *testing.T) {
	deleted := false
	ops := &LoadBalancerOps{LBClient: &mockLBClient{
		deleteFn: func(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error) {
			deleted = true
			return nil, nil
		},
	}}
	if err := ops.Delete(context.Background(), &hcloud.LoadBalancer{ID: 1}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete to be called")
	}

	ops = &LoadBalancerOps{LBClient: &mockLBClient{
		deleteFn: func(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error) {
			return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound}
		},
	}}
	if err := ops.Delete(context.Background(), &hcloud.LoadBalancer{ID: 1}); err != nil {
		t.Fatalf("not found should be ignored, got %v", err)
	}
}

func TestLoadBalancerOps_ReconcileHCLBTargets(t *testing.T) {
	origProvider := ProviderName
	t.Cleanup(func() { ProviderName = origProvider })
	ProviderName = "hetzner"

	lbClient := &mockLBClient{}
	ops := &LoadBalancerOps{
		LBClient:     lbClient,
		ActionClient: &mockActionClient{},
	}

	lb := &hcloud.LoadBalancer{
		ID: 1,
		Targets: []hcloud.LoadBalancerTarget{
			{
				Type: hcloud.LoadBalancerTargetTypeServer,
				Server: &hcloud.LoadBalancerTargetServer{
					Server: &hcloud.Server{ID: 99},
				},
			},
		},
	}
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "svc"}}
	nodes := []*v1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cloud-node",
				Labels: map[string]string{
					NameLabelType: NameCloudNode,
				},
			},
			Spec: v1.NodeSpec{ProviderID: "hetzner://10"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "robot-node",
				Labels: map[string]string{
					NameLabelType: NameDedicatedNode,
				},
			},
			Spec: v1.NodeSpec{ProviderID: "hetzner://20"},
			Status: v1.NodeStatus{
				Addresses: []v1.NodeAddress{
					{Type: v1.NodeExternalIP, Address: "203.0.113.20"},
				},
			},
		},
	}

	changed, err := ops.ReconcileHCLBTargets(context.Background(), lb, svc, nodes)
	if err != nil {
		t.Fatalf("ReconcileHCLBTargets: %v", err)
	}
	if !changed {
		t.Fatal("expected changes")
	}
	if len(lbClient.removedServerIDs) != 1 || lbClient.removedServerIDs[0] != 99 {
		t.Fatalf("expected stale target 99 removed, got %#v", lbClient.removedServerIDs)
	}
	if len(lbClient.addedServerTargets) != 1 || lbClient.addedServerTargets[0].Server.ID != 10 {
		t.Fatalf("expected cloud target 10, got %#v", lbClient.addedServerTargets)
	}
	if len(lbClient.addedIPTargets) != 1 || lbClient.addedIPTargets[0].IP.String() != "203.0.113.20" {
		t.Fatalf("expected robot IP target, got %#v", lbClient.addedIPTargets)
	}
}

func TestLoadBalancerOps_ReconcileHCLBTargets_NoChange(t *testing.T) {
	origProvider := ProviderName
	t.Cleanup(func() { ProviderName = origProvider })
	ProviderName = "hetzner"

	lbClient := &mockLBClient{}
	ops := &LoadBalancerOps{
		LBClient:     lbClient,
		ActionClient: &mockActionClient{},
	}

	lb := &hcloud.LoadBalancer{
		ID: 1,
		Targets: []hcloud.LoadBalancerTarget{
			{
				Type:         hcloud.LoadBalancerTargetTypeServer,
				UsePrivateIP: false,
				Server: &hcloud.LoadBalancerTargetServer{
					Server: &hcloud.Server{ID: 10},
				},
			},
		},
	}
	nodes := []*v1.Node{{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cloud-node",
			Labels: map[string]string{NameLabelType: NameCloudNode},
		},
		Spec: v1.NodeSpec{ProviderID: "hetzner://10"},
	}}

	changed, err := ops.ReconcileHCLBTargets(context.Background(), lb, &v1.Service{}, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected no changes")
	}
	if len(lbClient.addedServerTargets) != 0 || len(lbClient.removedServerIDs) != 0 {
		t.Fatalf("unexpected mutations: added=%#v removed=%#v", lbClient.addedServerTargets, lbClient.removedServerIDs)
	}
}

func TestLbAttached(t *testing.T) {
	lb := &hcloud.LoadBalancer{
		PrivateNet: []hcloud.LoadBalancerPrivateNet{
			{Network: &hcloud.Network{ID: 5}, IP: net.ParseIP("10.0.0.2")},
		},
	}
	if !lbAttached(lb, 5) {
		t.Fatal("expected attached")
	}
	if lbAttached(lb, 6) {
		t.Fatal("expected not attached")
	}
}
