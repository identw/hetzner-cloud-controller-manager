package annotation

import (
	"errors"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestName_AnnotateAndRead(t *testing.T) {
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{}}

	if err := LBID.AnnotateService(svc, int64(42)); err != nil {
		t.Fatalf("annotate int64: %v", err)
	}
	if got := svc.Annotations[string(LBID)]; got != "42" {
		t.Fatalf("got %q", got)
	}

	if err := LBHostname.AnnotateService(svc, "lb.example.com"); err != nil {
		t.Fatalf("annotate string: %v", err)
	}
	v, ok := LBHostname.StringFromService(svc)
	if !ok || v != "lb.example.com" {
		t.Fatalf("got %q ok=%v", v, ok)
	}

	if err := LBDisablePublicNetwork.AnnotateService(svc, true); err != nil {
		t.Fatalf("annotate bool: %v", err)
	}
	b, err := LBDisablePublicNetwork.BoolFromService(svc)
	if err != nil || !b {
		t.Fatalf("bool=%v err=%v", b, err)
	}

	certs := []*hcloud.Certificate{{ID: 7}, {Name: "cert-a"}}
	if err := LBSvcHTTPCertificates.AnnotateService(svc, certs); err != nil {
		t.Fatalf("annotate certs: %v", err)
	}
	got, err := LBSvcHTTPCertificates.CertificatesFromService(svc)
	if err != nil {
		t.Fatalf("CertificatesFromService: %v", err)
	}
	if len(got) != 2 || got[0].ID != 7 || got[1].Name != "cert-a" {
		t.Fatalf("unexpected certs: %#v", got)
	}
}

func TestName_MissingAnnotation(t *testing.T) {
	svc := &v1.Service{}
	if _, ok := LBLocation.StringFromService(svc); ok {
		t.Fatal("expected missing")
	}
	_, err := LBDisablePrivateIngress.BoolFromService(svc)
	if !errors.Is(err, ErrNotSet) {
		t.Fatalf("expected ErrNotSet, got %v", err)
	}
}

func TestValidateAlgorithmAndProtocol(t *testing.T) {
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
		string(LBAlgorithmType): "round_robin",
		string(LBSvcProtocol):   "https",
	}}}

	alg, err := LBAlgorithmType.LBAlgorithmTypeFromService(svc)
	if err != nil || alg != hcloud.LoadBalancerAlgorithmTypeRoundRobin {
		t.Fatalf("alg=%v err=%v", alg, err)
	}
	proto, err := LBSvcProtocol.LBSvcProtocolFromService(svc)
	if err != nil || proto != hcloud.LoadBalancerServiceProtocolHTTPS {
		t.Fatalf("proto=%v err=%v", proto, err)
	}

	svc.Annotations[string(LBAlgorithmType)] = "nope"
	if _, err := LBAlgorithmType.LBAlgorithmTypeFromService(svc); err == nil {
		t.Fatal("expected invalid algorithm error")
	}
}
