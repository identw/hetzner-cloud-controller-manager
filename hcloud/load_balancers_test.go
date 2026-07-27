package hcloud

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/identw/hetzner-cloud-controller-manager/internal/annotation"
	"github.com/identw/hetzner-cloud-controller-manager/internal/hcops"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockLoadBalancerOps struct {
	getByUIDFn            func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error)
	getByNameFn           func(ctx context.Context, name string) (*hcloud.LoadBalancer, error)
	getByIDFn             func(ctx context.Context, id int64) (*hcloud.LoadBalancer, error)
	createFn              func(ctx context.Context, lbName string, service *v1.Service) (*hcloud.LoadBalancer, error)
	deleteFn              func(ctx context.Context, lb *hcloud.LoadBalancer) error
	reconcileHCLBFn       func(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)
	reconcileTargetsFn    func(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service, nodes []*v1.Node) (bool, error)
	reconcileServicesFn   func(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error)

	createdName string
	deletedID   int64
}

func (m *mockLoadBalancerOps) GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return nil, hcops.ErrNotFound
}

func (m *mockLoadBalancerOps) GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, hcops.ErrNotFound
}

func (m *mockLoadBalancerOps) GetByK8SServiceUID(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
	if m.getByUIDFn != nil {
		return m.getByUIDFn(ctx, svc)
	}
	return nil, hcops.ErrNotFound
}

func (m *mockLoadBalancerOps) Create(ctx context.Context, lbName string, service *v1.Service) (*hcloud.LoadBalancer, error) {
	m.createdName = lbName
	if m.createFn != nil {
		return m.createFn(ctx, lbName, service)
	}
	return &hcloud.LoadBalancer{
		ID:               1,
		Name:             lbName,
		LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
		Algorithm:        hcloud.LoadBalancerAlgorithm{Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin},
		Location:         &hcloud.Location{Name: "fsn1", NetworkZone: hcloud.NetworkZoneEUCentral},
		PublicNet: hcloud.LoadBalancerPublicNet{
			IPv4: hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("203.0.113.10")},
			IPv6: hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("2001:db8::1")},
		},
	}, nil
}

func (m *mockLoadBalancerOps) Delete(ctx context.Context, lb *hcloud.LoadBalancer) error {
	m.deletedID = lb.ID
	if m.deleteFn != nil {
		return m.deleteFn(ctx, lb)
	}
	return nil
}

func (m *mockLoadBalancerOps) ReconcileHCLB(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	if m.reconcileHCLBFn != nil {
		return m.reconcileHCLBFn(ctx, lb, svc)
	}
	return false, nil
}

func (m *mockLoadBalancerOps) ReconcileHCLBTargets(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service, nodes []*v1.Node) (bool, error) {
	if m.reconcileTargetsFn != nil {
		return m.reconcileTargetsFn(ctx, lb, svc, nodes)
	}
	return false, nil
}

func (m *mockLoadBalancerOps) ReconcileHCLBServices(ctx context.Context, lb *hcloud.LoadBalancer, svc *v1.Service) (bool, error) {
	if m.reconcileServicesFn != nil {
		return m.reconcileServicesFn(ctx, lb, svc)
	}
	return false, nil
}

func sampleLB() *hcloud.LoadBalancer {
	return &hcloud.LoadBalancer{
		ID:               42,
		Name:             "lb-existing",
		LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"},
		Algorithm:        hcloud.LoadBalancerAlgorithm{Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin},
		Location:         &hcloud.Location{Name: "fsn1", NetworkZone: hcloud.NetworkZoneEUCentral},
		PublicNet: hcloud.LoadBalancerPublicNet{
			IPv4: hcloud.LoadBalancerPublicNetIPv4{IP: net.ParseIP("203.0.113.42")},
			IPv6: hcloud.LoadBalancerPublicNetIPv6{IP: net.ParseIP("2001:db8::42")},
		},
	}
}

func TestLoadBalancers_GetLoadBalancer(t *testing.T) {
	ops := &mockLoadBalancerOps{
		getByUIDFn: func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
			return sampleLB(), nil
		},
	}
	l := newLoadBalancers(ops, nil, true, commonClient{})
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "svc", UID: "u1"}}

	status, exists, err := l.GetLoadBalancer(context.Background(), "c", svc)
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	if len(status.Ingress) != 2 || status.Ingress[0].IP != "203.0.113.42" {
		t.Fatalf("unexpected status: %#v", status)
	}

	svc.Annotations = map[string]string{string(annotation.LBHostname): "lb.example.com"}
	status, exists, err = l.GetLoadBalancer(context.Background(), "c", svc)
	if err != nil || !exists || status.Ingress[0].Hostname != "lb.example.com" {
		t.Fatalf("hostname status=%#v exists=%v err=%v", status, exists, err)
	}
}

func TestLoadBalancers_EnsureLoadBalancer_CreatesWhenMissing(t *testing.T) {
	origConfig := cloudConfig
	t.Cleanup(func() { cloudConfig = origConfig })
	cloudConfig = &config{}

	ops := &mockLoadBalancerOps{}
	l := newLoadBalancers(ops, nil, true, commonClient{})
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc", UID: "uid-1", Namespace: "default"},
		Spec:       v1.ServiceSpec{Ports: []v1.ServicePort{{Port: 80}}},
	}

	status, err := l.EnsureLoadBalancer(context.Background(), "cluster", svc, nil)
	if err != nil {
		t.Fatalf("EnsureLoadBalancer: %v", err)
	}
	if ops.createdName == "" {
		t.Fatal("expected Create to be called")
	}
	if len(status.Ingress) < 1 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestLoadBalancers_EnsureLoadBalancerDeleted(t *testing.T) {
	ops := &mockLoadBalancerOps{
		getByUIDFn: func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
			return &hcloud.LoadBalancer{ID: 9, Protection: hcloud.LoadBalancerProtection{Delete: false}}, nil
		},
	}
	l := newLoadBalancers(ops, nil, true, commonClient{})
	if err := l.EnsureLoadBalancerDeleted(context.Background(), "c", &v1.Service{}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if ops.deletedID != 9 {
		t.Fatalf("deletedID=%d", ops.deletedID)
	}

	ops = &mockLoadBalancerOps{
		getByUIDFn: func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
			return &hcloud.LoadBalancer{ID: 9, Protection: hcloud.LoadBalancerProtection{Delete: true}}, nil
		},
	}
	l = newLoadBalancers(ops, nil, true, commonClient{})
	if err := l.EnsureLoadBalancerDeleted(context.Background(), "c", &v1.Service{}); err != nil {
		t.Fatalf("protected delete: %v", err)
	}
	if ops.deletedID != 0 {
		t.Fatal("protected LB must not be deleted")
	}

	ops = &mockLoadBalancerOps{
		getByUIDFn: func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
			return nil, hcops.ErrNotFound
		},
	}
	l = newLoadBalancers(ops, nil, true, commonClient{})
	if err := l.EnsureLoadBalancerDeleted(context.Background(), "c", &v1.Service{}); err != nil {
		t.Fatalf("missing lb: %v", err)
	}
}

func TestLoadBalancers_UpdateLoadBalancer_MissingIsNoop(t *testing.T) {
	origConfig := cloudConfig
	t.Cleanup(func() { cloudConfig = origConfig })
	cloudConfig = &config{}

	ops := &mockLoadBalancerOps{}
	l := newLoadBalancers(ops, nil, true, commonClient{})
	err := l.UpdateLoadBalancer(context.Background(), "c", &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "svc"}}, nil)
	if err != nil {
		t.Fatalf("UpdateLoadBalancer: %v", err)
	}
}

func TestLoadBalancers_GetLoadBalancer_NotFound(t *testing.T) {
	l := newLoadBalancers(&mockLoadBalancerOps{}, nil, true, commonClient{})
	_, exists, err := l.GetLoadBalancer(context.Background(), "c", &v1.Service{})
	if err != nil || exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}

func TestLoadBalancers_GetLoadBalancerName(t *testing.T) {
	l := newLoadBalancers(&mockLoadBalancerOps{}, nil, true, commonClient{})
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:        "svc",
		UID:         "abcdef12-3456-7890-abcd-ef1234567890",
		Annotations: map[string]string{string(annotation.LBName): "custom-lb"},
	}}
	if got := l.GetLoadBalancerName(context.Background(), "c", svc); got != "custom-lb" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureLoadBalancer_PropagatesOpsError(t *testing.T) {
	origConfig := cloudConfig
	t.Cleanup(func() { cloudConfig = origConfig })
	cloudConfig = &config{}

	ops := &mockLoadBalancerOps{
		getByUIDFn: func(ctx context.Context, svc *v1.Service) (*hcloud.LoadBalancer, error) {
			return nil, errors.New("api boom")
		},
	}
	l := newLoadBalancers(ops, nil, true, commonClient{})
	_, err := l.EnsureLoadBalancer(context.Background(), "c", &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "svc"}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
