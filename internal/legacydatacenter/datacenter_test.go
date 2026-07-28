package legacydatacenter

import "testing"

func TestNameFromLocation(t *testing.T) {
	tests := []struct {
		location string
		want     string
	}{
		{location: "nbg1", want: "nbg1-dc3"},
		{location: "hel1", want: "hel1-dc2"},
		{location: "fsn1", want: "fsn1-dc14"},
		{location: "ash", want: "ash-dc1"},
		{location: "hil", want: "hil-dc1"},
		{location: "sin", want: "sin-dc1"},
		{location: "exclude", want: "exclude"},
		{location: "newloc", want: "newloc"},
	}

	for _, tt := range tests {
		t.Run(tt.location, func(t *testing.T) {
			if got := NameFromLocation(tt.location); got != tt.want {
				t.Fatalf("NameFromLocation(%q) = %q, want %q", tt.location, got, tt.want)
			}
		})
	}
}
