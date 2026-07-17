package ui

import (
	"testing"

	"github.com/ElyessBenSassi/devops-tools/dmon/internal/compose"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
)

func TestBelongsToProject(t *testing.T) {
	proj := &compose.Project{
		File: "/home/me/app/docker-compose.yml",
		Dir:  "/home/me/app",
		Name: "app",
	}
	m := Model{project: proj}

	tests := []struct {
		name string
		c    docker.ContainerInfo
		want bool
	}{
		{
			name: "matches by working dir",
			c:    docker.ContainerInfo{ComposeWorkingDir: "/home/me/app"},
			want: true,
		},
		{
			name: "matches by config file",
			c:    docker.ContainerInfo{ComposeConfigs: []string{"/home/me/app/docker-compose.yml", "/home/me/app/override.yml"}},
			want: true,
		},
		{
			name: "matches with unclean path",
			c:    docker.ContainerInfo{ComposeWorkingDir: "/home/me/app/"},
			want: true,
		},
		{
			name: "different project",
			c:    docker.ContainerInfo{ComposeWorkingDir: "/home/me/other", ComposeConfigs: []string{"/home/me/other/docker-compose.yml"}},
			want: false,
		},
		{
			name: "standalone container",
			c:    docker.ContainerInfo{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.belongsToProject(tt.c); got != tt.want {
				t.Errorf("belongsToProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBelongsToProjectNilProject(t *testing.T) {
	m := Model{project: nil}
	if m.belongsToProject(docker.ContainerInfo{ComposeWorkingDir: "/anything"}) {
		t.Error("belongsToProject should be false when no project is scoped")
	}
}

func TestMarkProjectOrdersAndTags(t *testing.T) {
	proj := &compose.Project{File: "/app/compose.yml", Dir: "/app", Name: "app"}
	m := Model{project: proj}

	in := []docker.ContainerInfo{
		{Name: "zeta-standalone"},
		{Name: "app-web-1", ComposeWorkingDir: "/app", ComposeService: "web"},
		{Name: "alpha-standalone"},
		{Name: "app-db-1", ComposeWorkingDir: "/app", ComposeService: "db"},
	}

	got := m.markProject(in)

	// Project members first, ordered by service name (db before web).
	wantOrder := []string{"app-db-1", "app-web-1", "alpha-standalone", "zeta-standalone"}
	if len(got) != len(wantOrder) {
		t.Fatalf("got %d rows, want %d", len(got), len(wantOrder))
	}
	for i, name := range wantOrder {
		if got[i].Name != name {
			t.Errorf("row %d = %q, want %q", i, got[i].Name, name)
		}
	}

	// First two are project members; last two are not.
	for i, ctr := range got {
		wantIn := i < 2
		if ctr.InProject != wantIn {
			t.Errorf("row %d (%s) InProject = %v, want %v", i, ctr.Name, ctr.InProject, wantIn)
		}
	}
}
