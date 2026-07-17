package ui

import (
	"strings"
	"testing"

	"github.com/ElyessBenSassi/devops-tools/dmon/internal/compose"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleContainers() []docker.ContainerInfo {
	return []docker.ContainerInfo{
		{Name: "plantuml", State: "running", Image: "plantuml/plantuml-server", Ports: "8080->8080/tcp"},
		{Name: "app-backend-1", State: "running", Health: "healthy",
			ComposeProject: "app", ComposeService: "backend", ComposeWorkingDir: "/app",
			ComposeConfigs: []string{"/app/docker-compose.yml"}},
		{Name: "app-db-1", State: "running", Health: "starting",
			ComposeProject: "app", ComposeService: "db", ComposeWorkingDir: "/app"},
	}
}

// loadedModel returns a Model sized and populated with sampleContainers.
func loadedModel(project *compose.Project) Model {
	m := NewModel(nil, project)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = mm.(Model)
	mm, _ = m.Update(containersLoadedMsg{containers: sampleContainers()})
	return mm.(Model)
}

func TestViewLoadingState(t *testing.T) {
	// Sized but no container fetch has returned yet.
	m := NewModel(nil, nil)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = mm.(Model)

	view := m.View()
	if !strings.Contains(view, "Loading containers…") {
		t.Error("view should show a loading indicator before the first fetch returns")
	}
	if strings.Contains(view, "No containers found.") {
		t.Error("view must not claim zero containers while still loading")
	}

	// After a fetch returns empty, it should say so rather than loading.
	mm, _ = m.Update(containersLoadedMsg{containers: nil})
	m = mm.(Model)
	after := m.View()
	if !strings.Contains(after, "No containers found.") {
		t.Error("view should report no containers once a fetch has returned empty")
	}
	if strings.Contains(after, "Loading containers…") {
		t.Error("loading indicator must disappear after the first fetch")
	}
}

func TestViewWithProject(t *testing.T) {
	m := loadedModel(&compose.Project{File: "/app/docker-compose.yml", Dir: "/app", Name: "app"})
	view := m.View()
	for _, want := range []string{"compose · app", "other containers", "u compose up", "plantuml"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestViewNoProject(t *testing.T) {
	m := loadedModel(nil)
	view := m.View()
	if !strings.Contains(view, "no compose file detected") {
		t.Error("view should note that no compose file was detected")
	}
	if strings.Contains(view, "u compose up") {
		t.Error("compose up hint must be hidden when no project is detected")
	}
	if strings.Contains(view, "other containers") {
		t.Error("no section headers should appear without a project")
	}
}

func TestViewProfilePrompt(t *testing.T) {
	m := loadedModel(&compose.Project{File: "/app/docker-compose.yml", Dir: "/app", Name: "app"})
	// Press 'u', then type "gpu".
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = mm.(Model)
	for _, r := range "gpu" {
		mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mm.(Model)
	}
	if m.mode != modeProfilePrompt {
		t.Fatalf("expected modeProfilePrompt, got %v", m.mode)
	}
	if !strings.Contains(m.View(), "Profile name (blank = default services): gpu") {
		t.Error("prompt should echo the typed profile name")
	}
	// Esc cancels back to the list without running anything.
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.mode != modeList {
		t.Error("esc should return to list mode")
	}
}

func TestComposeUpUnavailableWhenNoProject(t *testing.T) {
	m := loadedModel(nil)
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = mm.(Model)
	if m.mode != modeList {
		t.Error("'u' must not open a prompt when no project is detected")
	}
	if cmd != nil {
		t.Error("'u' should be a no-op command when no project is detected")
	}
}
