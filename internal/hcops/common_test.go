package hcops

import (
	"testing"
)

func TestProviderIDToServerID(t *testing.T) {
	origProvider := ProviderName
	t.Cleanup(func() { ProviderName = origProvider })
	ProviderName = "hetzner"

	tests := []struct {
		name    string
		input   string
		wantID  int64
		wantErr bool
	}{
		{
			name:   "valid cloud provider id",
			input:  "hetzner://42",
			wantID: 42,
		},
		{
			name:   "exclude sentinel as bare id",
			input:  "999999",
			wantID: ExcludeServer.ID,
		},
		{
			name:    "wrong prefix",
			input:   "hcloud://1",
			wantErr: true,
		},
		{
			name:    "missing id",
			input:   "hetzner://",
			wantErr: true,
		},
		{
			name:    "non numeric id",
			input:   "hetzner://abc",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ProviderIDToServerID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got id=%d", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("got id=%d, want %d", id, tt.wantID)
			}
		})
	}
}

func TestProviderIDToServerID_CustomProviderName(t *testing.T) {
	origProvider := ProviderName
	t.Cleanup(func() { ProviderName = origProvider })
	ProviderName = "hcloud"

	id, err := ProviderIDToServerID("hcloud://7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("got id=%d, want 7", id)
	}

	if _, err = ProviderIDToServerID("hetzner://7"); err == nil {
		t.Fatal("expected error for wrong prefix")
	}
}
