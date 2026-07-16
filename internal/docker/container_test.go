package docker

import (
	"testing"
)

func TestParseHealthFromStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"healthy", "Up 2 hours (healthy)", "healthy"},
		{"unhealthy", "Up 3 minutes (unhealthy)", "unhealthy"},
		{"starting", "Up 1 minute (health: starting)", "starting"},
		{"no healthcheck", "Up 5 hours", ""},
		{"exited", "Exited (1) 2 minutes ago", ""},
		{"empty", "", ""},
		{"healthy uppercase", "Up 1 hour (Healthy)", "healthy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHealthFromStatus(tt.status)
			if got != tt.want {
				t.Errorf("parseHealthFromStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	mb := uint64(1 << 20)
	gb := uint64(1 << 30)

	tests := []struct {
		name  string
		input uint64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"exactly 1 MB", mb, "1.0 MB"},
		{"123.4 MB", uint64(123.4 * float64(mb)), "123.4 MB"},
		{"exactly 1 GB", gb, "1.0 GB"},
		{"2.5 GB", uint64(2.5 * float64(gb)), "2.5 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.input)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortenImage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain image", "nginx:latest", "nginx:latest"},
		{"with digest", "nginx@sha256:abc123def456", "nginx"},
		{"registry with tag and digest", "registry.example.com/myapp:v1.0@sha256:deadbeef", "registry.example.com/myapp:v1.0"},
		{"no tag no digest", "ubuntu", "ubuntu"},
		{"registry with tag only", "ghcr.io/org/image:v2", "ghcr.io/org/image:v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenImage(tt.input)
			if got != tt.want {
				t.Errorf("shortenImage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
