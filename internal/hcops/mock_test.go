package hcops

import (
	"context"
	"net"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type mockActionClient struct {
	watchErr error
}

func (m *mockActionClient) WatchProgress(ctx context.Context, a *hcloud.Action) (<-chan int, <-chan error) {
	progress := make(chan int)
	errc := make(chan error, 1)
	close(progress)
	errc <- m.watchErr
	close(errc)
	return progress, errc
}

type mockNetworkClient struct {
	network *hcloud.Network
	err     error
}

func (m *mockNetworkClient) GetByID(ctx context.Context, id int64) (*hcloud.Network, *hcloud.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.network != nil && m.network.ID != id {
		return nil, nil, nil
	}
	return m.network, nil, nil
}

type mockCertificateClient struct {
	certs map[string]*hcloud.Certificate
	err   error
}

func (m *mockCertificateClient) Get(ctx context.Context, idOrName string) (*hcloud.Certificate, *hcloud.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.certs == nil {
		return nil, nil, nil
	}
	return m.certs[idOrName], nil, nil
}

// mockLBClient implements HCloudLoadBalancerClient with overridable hooks.
type mockLBClient struct {
	getByIDFn            func(ctx context.Context, id int64) (*hcloud.LoadBalancer, *hcloud.Response, error)
	getByNameFn          func(ctx context.Context, name string) (*hcloud.LoadBalancer, *hcloud.Response, error)
	createFn             func(ctx context.Context, opts hcloud.LoadBalancerCreateOpts) (hcloud.LoadBalancerCreateResult, *hcloud.Response, error)
	updateFn             func(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerUpdateOpts) (*hcloud.LoadBalancer, *hcloud.Response, error)
	deleteFn             func(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error)
	allWithOptsFn        func(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error)
	addServerTargetFn    func(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts) (*hcloud.Action, *hcloud.Response, error)
	addIPTargetFn        func(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddIPTargetOpts) (*hcloud.Action, *hcloud.Response, error)
	removeServerTargetFn func(ctx context.Context, lb *hcloud.LoadBalancer, server *hcloud.Server) (*hcloud.Action, *hcloud.Response, error)
	removeIPTargetFn     func(ctx context.Context, lb *hcloud.LoadBalancer, ip net.IP) (*hcloud.Action, *hcloud.Response, error)

	addedServerTargets []hcloud.LoadBalancerAddServerTargetOpts
	addedIPTargets     []hcloud.LoadBalancerAddIPTargetOpts
	removedServerIDs   []int64
	removedIPs         []string
}

func (m *mockLBClient) GetByID(ctx context.Context, id int64) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil, nil
}

func (m *mockLBClient) GetByName(ctx context.Context, name string) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return nil, nil, nil
}

func (m *mockLBClient) Create(ctx context.Context, opts hcloud.LoadBalancerCreateOpts) (hcloud.LoadBalancerCreateResult, *hcloud.Response, error) {
	if m.createFn != nil {
		return m.createFn(ctx, opts)
	}
	return hcloud.LoadBalancerCreateResult{}, nil, nil
}

func (m *mockLBClient) Update(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerUpdateOpts) (*hcloud.LoadBalancer, *hcloud.Response, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, lb, opts)
	}
	return lb, nil, nil
}

func (m *mockLBClient) Delete(ctx context.Context, lb *hcloud.LoadBalancer) (*hcloud.Response, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, lb)
	}
	return nil, nil
}

func (m *mockLBClient) AddService(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServiceOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) UpdateService(ctx context.Context, lb *hcloud.LoadBalancer, listenPort int, opts hcloud.LoadBalancerUpdateServiceOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) DeleteService(ctx context.Context, lb *hcloud.LoadBalancer, listenPort int) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) ChangeAlgorithm(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeAlgorithmOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) ChangeType(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerChangeTypeOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) AddServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts) (*hcloud.Action, *hcloud.Response, error) {
	m.addedServerTargets = append(m.addedServerTargets, opts)
	if m.addServerTargetFn != nil {
		return m.addServerTargetFn(ctx, lb, opts)
	}
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) AddIPTarget(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddIPTargetOpts) (*hcloud.Action, *hcloud.Response, error) {
	m.addedIPTargets = append(m.addedIPTargets, opts)
	if m.addIPTargetFn != nil {
		return m.addIPTargetFn(ctx, lb, opts)
	}
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) RemoveServerTarget(ctx context.Context, lb *hcloud.LoadBalancer, server *hcloud.Server) (*hcloud.Action, *hcloud.Response, error) {
	if server != nil {
		m.removedServerIDs = append(m.removedServerIDs, server.ID)
	}
	if m.removeServerTargetFn != nil {
		return m.removeServerTargetFn(ctx, lb, server)
	}
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) RemoveIPTarget(ctx context.Context, lb *hcloud.LoadBalancer, ip net.IP) (*hcloud.Action, *hcloud.Response, error) {
	m.removedIPs = append(m.removedIPs, ip.String())
	if m.removeIPTargetFn != nil {
		return m.removeIPTargetFn(ctx, lb, ip)
	}
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) AttachToNetwork(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAttachToNetworkOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) DetachFromNetwork(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerDetachFromNetworkOpts) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) EnablePublicInterface(ctx context.Context, loadBalancer *hcloud.LoadBalancer) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) DisablePublicInterface(ctx context.Context, loadBalancer *hcloud.LoadBalancer) (*hcloud.Action, *hcloud.Response, error) {
	return &hcloud.Action{ID: 1}, nil, nil
}

func (m *mockLBClient) AllWithOpts(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error) {
	if m.allWithOptsFn != nil {
		return m.allWithOptsFn(ctx, opts)
	}
	return nil, nil
}
