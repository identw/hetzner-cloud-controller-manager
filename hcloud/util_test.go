package hcloud

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

func TestNormalizeK8sLabelValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"Server Auction", "Server-Auction"},
		{"my_value", "my_value"},
		{"MyValue", "MyValue"},
		{"12345", "12345"},
		{"  spaced  ", "spaced"},
		{"!!!", ""},
		{"a.b-c_d", "a.b-c_d"},
		{"foo/bar", "foo-bar"},
		{"--trim--", "trim"},
		{"café", "caf"},
		{"привет", ""},
	}

	for _, tt := range tests {
		got := normalizeK8sLabelValue(tt.in)
		if got != tt.want {
			t.Errorf("normalizeK8sLabelValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if errs := validation.IsValidLabelValue(got); len(errs) != 0 {
			t.Errorf("normalizeK8sLabelValue(%q) = %q is not a valid label value: %v", tt.in, got, errs)
		}
	}
}

func TestNormalizeK8sLabelValueMaxLength(t *testing.T) {
	in := strings.Repeat("a", 80) + " " + strings.Repeat("b", 10)
	got := normalizeK8sLabelValue(in)
	if len(got) > validation.LabelValueMaxLength {
		t.Fatalf("length %d exceeds max %d", len(got), validation.LabelValueMaxLength)
	}
	if errs := validation.IsValidLabelValue(got); len(errs) != 0 {
		t.Fatalf("invalid after truncate: %v", errs)
	}
}

func TestNormalizeK8sLabelKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"valid-key", "valid-key"},
		{"example.com/valid-key", "example.com/valid-key"},
		{"bad key", "bad-key"},
		{"example.com/bad key", "example.com/bad-key"},
		{"!!!", ""},
	}

	for _, tt := range tests {
		got := normalizeK8sLabelKey(tt.in)
		if got != tt.want {
			t.Errorf("normalizeK8sLabelKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if got == "" {
			continue
		}
		if errs := validation.IsQualifiedName(got); len(errs) != 0 {
			t.Errorf("normalizeK8sLabelKey(%q) = %q is not a qualified name: %v", tt.in, got, errs)
		}
	}
}
