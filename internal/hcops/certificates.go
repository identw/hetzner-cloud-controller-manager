package hcops

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// HCloudCertificateClient defines the hcloud-go function related to
// certificate management.
type HCloudCertificateClient interface {
	Get(ctx context.Context, idOrName string) (*hcloud.Certificate, *hcloud.Response, error)
}
