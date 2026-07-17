package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// ContainerInfo holds the display data for a single container.
type ContainerInfo struct {
	ID     string  // 12-char short ID
	Name   string  // trimmed (no leading /)
	State  string  // running, exited, etc.
	Health string  // healthy, unhealthy, starting, or ""
	CPU    float64 // percent
	Memory string  // "123.4 MB / 2.0 GB"
	Ports  string  // "8080->80/tcp, 443->443/tcp"
	Image  string

	// Compose metadata, read from the container's labels ("" / nil when the
	// container is not managed by Docker Compose).
	ComposeProject    string   // com.docker.compose.project
	ComposeService    string   // com.docker.compose.service
	ComposeWorkingDir string   // com.docker.compose.project.working_dir
	ComposeConfigs    []string // com.docker.compose.project.config_files, split

	// InProject is set by the UI once it knows which compose project dmon is
	// scoped to; the docker layer always leaves it false.
	InProject bool
}

// statsJSON mirrors the Docker stats API response (snake_case JSON).
type statsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
}

// ListContainers returns info for all containers visible to Docker.
func (c *Client) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	ctrs, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.Args{},
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	infos := make([]ContainerInfo, 0, len(ctrs))
	for _, ctr := range ctrs {
		name := ""
		if len(ctr.Names) > 0 {
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}

		health := parseHealthFromStatus(ctr.Status)

		cpu, mem := fetchStats(ctx, c, ctr.ID, string(ctr.State))

		infos = append(infos, ContainerInfo{
			ID:                ctr.ID[:12],
			Name:              name,
			State:             string(ctr.State),
			Health:            health,
			CPU:               cpu,
			Memory:            mem,
			Ports:             formatPorts(ctr.Ports),
			Image:             shortenImage(ctr.Image),
			ComposeProject:    ctr.Labels["com.docker.compose.project"],
			ComposeService:    ctr.Labels["com.docker.compose.service"],
			ComposeWorkingDir: ctr.Labels["com.docker.compose.project.working_dir"],
			ComposeConfigs:    splitConfigFiles(ctr.Labels["com.docker.compose.project.config_files"]),
		})
	}
	return infos, nil
}

// fetchStats calls ContainerStats (non-streaming) and returns CPU% and memory string.
// Returns zero values on error so the list still shows the container.
func fetchStats(ctx context.Context, c *Client, id, state string) (float64, string) {
	if state != "running" {
		return 0, ""
	}

	resp, err := c.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, ""
	}

	var s statsJSON
	if err := json.Unmarshal(body, &s); err != nil {
		return 0, ""
	}

	// CPU calculation
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemCPUUsage) - float64(s.PreCPUStats.SystemCPUUsage)
	numCPUs := s.CPUStats.OnlineCPUs
	if numCPUs == 0 {
		numCPUs = len(s.CPUStats.CPUUsage.PercpuUsage)
	}
	if numCPUs == 0 {
		numCPUs = 1
	}

	var cpuPercent float64
	if sysDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * float64(numCPUs) * 100.0
	}

	// Memory calculation — subtract cache from usage when available
	cache := s.MemoryStats.Stats["cache"]
	usedMem := s.MemoryStats.Usage
	if usedMem > cache {
		usedMem -= cache
	}
	memStr := ""
	if s.MemoryStats.Limit > 0 {
		memStr = fmt.Sprintf("%s / %s", formatBytes(usedMem), formatBytes(s.MemoryStats.Limit))
	}

	return cpuPercent, memStr
}

// RestartContainer restarts the container with a 10-second stop timeout.
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	timeout := int(10 * time.Second / time.Second) // 10 seconds
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
}

// splitConfigFiles splits the comma-separated config_files label into a cleaned
// list of paths, dropping empty entries. Returns nil when the label is empty.
func splitConfigFiles(label string) []string {
	if label == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(label, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseHealthFromStatus extracts a health keyword from the Docker status string.
// Examples: "Up 2 hours (healthy)" → "healthy"
//
//	"Up 3 minutes (unhealthy)" → "unhealthy"
//	"Up 1 minute (health: starting)" → "starting"
//	"Up 5 hours" → ""
func parseHealthFromStatus(status string) string {
	lower := strings.ToLower(status)
	switch {
	case strings.Contains(lower, "(unhealthy)"):
		return "unhealthy"
	case strings.Contains(lower, "(healthy)"):
		return "healthy"
	case strings.Contains(lower, "health: starting"):
		return "starting"
	default:
		return ""
	}
}

// formatBytes converts a byte count to a human-readable string (MB or GB).
func formatBytes(b uint64) string {
	const mb = 1 << 20
	const gb = 1 << 30
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// shortenImage removes the sha256 digest suffix if present.
func shortenImage(image string) string {
	if idx := strings.Index(image, "@sha256:"); idx != -1 {
		image = image[:idx]
	}
	return image
}

// formatPorts converts the Docker port list into a compact, sorted string.
// Example: "8080->80/tcp, 443->443/tcp"
func formatPorts(ports []types.Port) string {
	seen := make(map[string]struct{})
	var parts []string
	for _, p := range ports {
		var s string
		if p.PublicPort != 0 {
			s = fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
		} else {
			s = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			parts = append(parts, s)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
