package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ElyessBenSassi/devops-tools/dmon/internal/compose"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg is sent on each timer tick to trigger a refresh.
type tickMsg time.Time

// containersLoadedMsg carries freshly loaded container info.
type containersLoadedMsg struct {
	containers []docker.ContainerInfo
	err        error
}

// restartDoneMsg is sent when a restart command completes.
type restartDoneMsg struct {
	containerName string
	err           error
}

// execDoneMsg is sent when an exec session exits.
type execDoneMsg struct{ err error }

// composeUpDoneMsg is sent when a `docker compose up` session exits.
type composeUpDoneMsg struct{ err error }

// inputMode tracks whether the TUI is showing the container list or a prompt.
type inputMode int

const (
	modeList inputMode = iota
	modeProfilePrompt
)

// Model is the bubbletea application model.
type Model struct {
	containers   []docker.ContainerInfo
	cursor       int
	err          error
	status       string
	width        int
	height       int
	dockerClient *docker.Client
	lastRefresh  time.Time

	project *compose.Project // detected compose project, or nil (run-from-anywhere)

	loaded      bool      // true once the first container fetch has returned
	mode        inputMode // list vs. prompt
	profileText string    // buffer for the profile-name prompt
}

// NewModel creates a Model backed by the given Docker client, scoped to project
// (which may be nil when dmon is run outside a compose directory).
func NewModel(dockerClient *docker.Client, project *compose.Project) Model {
	return Model{
		dockerClient: dockerClient,
		project:      project,
	}
}

// Init starts the first refresh and sets up the periodic tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadContainers(m.dockerClient),
		tickEvery(),
	)
}

// tickEvery creates a Cmd that fires after 5 seconds.
func tickEvery() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadContainers returns a Cmd that fetches container info in the background.
func loadContainers(c *docker.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		containers, err := c.ListContainers(ctx)
		return containersLoadedMsg{containers: containers, err: err}
	}
}

// restartContainer returns a Cmd that restarts a container by ID.
func restartContainer(c *docker.Client, id, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := c.RestartContainer(ctx, id)
		return restartDoneMsg{containerName: name, err: err}
	}
}

// composeUp returns a Cmd that runs `docker compose up -d` for the detected
// project, optionally scoped to a profile.
func composeUp(file, profile string) tea.Cmd {
	args := []string{"compose", "-f", file}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	args = append(args, "up", "-d")
	cmd := exec.Command("docker", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return composeUpDoneMsg{err: err}
	})
}

// markProject tags each container that belongs to the detected project and
// orders the list so project members come first (by service name), followed by
// everything else (by name). Ordering is stable across refreshes.
func (m Model) markProject(containers []docker.ContainerInfo) []docker.ContainerInfo {
	for i := range containers {
		containers[i].InProject = m.belongsToProject(containers[i])
	}
	sort.SliceStable(containers, func(i, j int) bool {
		a, b := containers[i], containers[j]
		if a.InProject != b.InProject {
			return a.InProject // project members first
		}
		if a.InProject {
			return a.ComposeService < b.ComposeService
		}
		return a.Name < b.Name
	})
	return containers
}

// belongsToProject reports whether c is part of the compose project dmon is
// scoped to, matching on the working-dir or config-file labels Compose sets.
func (m Model) belongsToProject(c docker.ContainerInfo) bool {
	if m.project == nil {
		return false
	}
	if c.ComposeWorkingDir != "" && samePath(c.ComposeWorkingDir, m.project.Dir) {
		return true
	}
	for _, cfg := range c.ComposeConfigs {
		if samePath(cfg, m.project.File) {
			return true
		}
	}
	return false
}

// samePath compares two filesystem paths after cleaning them.
func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadContainers(m.dockerClient), tickEvery())

	case containersLoadedMsg:
		m.loaded = true
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.containers = m.markProject(msg.containers)
			m.lastRefresh = time.Now()
			m.status = "Last refresh: " + m.lastRefresh.Format("15:04:05")
			// Keep cursor in bounds
			if m.cursor >= len(m.containers) && len(m.containers) > 0 {
				m.cursor = len(m.containers) - 1
			}
		}
		return m, nil

	case restartDoneMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("restart %s: %w", msg.containerName, msg.err)
			m.status = ""
		} else {
			m.err = nil
			m.status = fmt.Sprintf("Restarted %s", msg.containerName)
		}
		return m, loadContainers(m.dockerClient)

	case execDoneMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("exec: %w", msg.err)
		}
		return m, loadContainers(m.dockerClient)

	case composeUpDoneMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("compose up: %w", msg.err)
			m.status = ""
		} else {
			m.err = nil
			m.status = "Compose up complete"
		}
		return m, loadContainers(m.dockerClient)

	case tea.KeyMsg:
		if m.mode == modeProfilePrompt {
			return m.updatePrompt(msg)
		}
		return m.updateList(msg)
	}

	return m, nil
}

// updatePrompt handles keystrokes while the profile-name prompt is active.
func (m Model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeList
		m.profileText = ""
		m.status = "Compose up cancelled"
		return m, nil

	case "enter":
		profile := strings.TrimSpace(m.profileText)
		m.mode = modeList
		m.profileText = ""
		if profile == "" {
			m.status = "Running compose up (default profile)..."
		} else {
			m.status = fmt.Sprintf("Running compose up (profile %q)...", profile)
		}
		m.err = nil
		return m, composeUp(m.project.File, profile)

	case "backspace":
		if r := []rune(m.profileText); len(r) > 0 {
			m.profileText = string(r[:len(r)-1])
		}
		return m, nil

	default:
		// Accept printable single-rune keystrokes into the buffer.
		if len(msg.Runes) == 1 {
			m.profileText += string(msg.Runes)
		}
		return m, nil
	}
}

// updateList handles keystrokes while the container list is active.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.containers)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "R":
		m.status = "Refreshing..."
		return m, loadContainers(m.dockerClient)

	case "r":
		if len(m.containers) > 0 {
			ctr := m.containers[m.cursor]
			m.status = fmt.Sprintf("Restarting %s...", ctr.Name)
			m.err = nil
			return m, restartContainer(m.dockerClient, ctr.ID, ctr.Name)
		}

	case "l", "L":
		if len(m.containers) > 0 {
			ctr := m.containers[m.cursor]
			cmd := exec.Command("sh", "-c",
				fmt.Sprintf("docker logs --tail 100 %s 2>&1 | less", ctr.ID))
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				if err != nil {
					return containersLoadedMsg{err: fmt.Errorf("logs: %w", err)}
				}
				// Reload on return instead of blanking the list.
				return execDoneMsg{}
			})
		}

	case "e":
		if len(m.containers) > 0 {
			ctr := m.containers[m.cursor]
			if ctr.State != "running" {
				m.status = fmt.Sprintf("%s is not running", ctr.Name)
				return m, nil
			}
			// try bash first, fall back to sh
			cmd := exec.Command("sh", "-c",
				fmt.Sprintf("docker exec -it %s bash 2>/dev/null || docker exec -it %s sh", ctr.ID, ctr.ID))
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return execDoneMsg{err: err}
			})
		}

	case "u":
		// compose up is only available when a compose project is detected.
		if m.project == nil {
			m.status = "No compose file detected — 'u' unavailable here"
			return m, nil
		}
		m.mode = modeProfilePrompt
		m.profileText = ""
		m.err = nil
		return m, nil
	}

	return m, nil
}

// View renders the entire TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Title line: compose project scope, or run-from-anywhere.
	if m.project != nil {
		sb.WriteString(projectStyle.Render(fmt.Sprintf("compose · %s", m.project.Name)))
		sb.WriteString(pathStyle.Render("  " + m.project.File))
	} else {
		sb.WriteString(projectStyle.Render("all containers"))
		sb.WriteString(pathStyle.Render("  (no compose file detected)"))
	}
	sb.WriteString("\n\n")

	// Table header
	header := formatRow(
		"CONTAINER", "STATE", "HEALTH", "CPU%", "MEMORY", "PORTS", "IMAGE",
	)
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", tableWidth()))
	sb.WriteString("\n")

	m.writeRows(&sb)

	if len(m.containers) == 0 {
		if m.loaded {
			sb.WriteString("  No containers found.\n")
		} else {
			sb.WriteString(statusStyle.Render("  Loading containers…"))
			sb.WriteString("\n")
		}
	}

	// Prompt or status bar
	sb.WriteString("\n")
	switch {
	case m.mode == modeProfilePrompt:
		sb.WriteString(promptStyle.Render("Profile name (blank = default services): " + m.profileText + "▏"))
	case m.err != nil:
		sb.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	default:
		sb.WriteString(statusStyle.Render(m.status))
	}

	sb.WriteString("\n")
	sb.WriteString(statusStyle.Render(m.helpLine()))

	return borderStyle.Render(sb.String())
}

// writeRows renders the container rows, inserting a section header before the
// project group and before the "other" group when a project is detected.
func (m Model) writeRows(sb *strings.Builder) {
	sectionShown := map[string]bool{}
	for i, ctr := range m.containers {
		if m.project != nil {
			key := "other"
			label := "other containers"
			if ctr.InProject {
				key = "project"
				label = fmt.Sprintf("compose · %s", m.project.Name)
			}
			if !sectionShown[key] {
				sectionShown[key] = true
				sb.WriteString(sectionStyle.Render("  " + label))
				sb.WriteString("\n")
			}
		}

		cpuStr := fmt.Sprintf("%.1f%%", ctr.CPU)
		row := formatRow(
			truncate(ctr.Name, colWidthName),
			truncate(ctr.State, colWidthState),
			truncate(ctr.Health, colWidthHealth),
			truncate(cpuStr, colWidthCPU),
			truncate(ctr.Memory, colWidthMemory),
			truncate(ctr.Ports, colWidthPorts),
			truncate(ctr.Image, colWidthImage),
		)

		sb.WriteString(m.styleRow(i, ctr, row))
		sb.WriteString("\n")
	}
}

// styleRow applies selection, project, and health styling to a rendered row.
func (m Model) styleRow(i int, ctr docker.ContainerInfo, row string) string {
	if i == m.cursor {
		return selectedRowStyle.Render("> " + row)
	}
	// Project members get an accent gutter; others a blank one.
	gutter := "  "
	if ctr.InProject {
		gutter = projectGutterStyle.Render("▎") + " "
		if ctr.Health != "" {
			return gutter + healthStyle(ctr.Health).Render(row)
		}
		return gutter + projectRowStyle.Render(row)
	}
	if ctr.Health != "" {
		return healthStyle(ctr.Health).Render(gutter + row)
	}
	return normalRowStyle.Render(gutter + row)
}

// helpLine returns the key-hint line, including compose up only when available.
func (m Model) helpLine() string {
	up := ""
	if m.project != nil {
		up = "u compose up  "
	}
	return "j/k move  r restart  l logs  e exec  " + up + "R refresh  q quit"
}

// tableWidth returns the total character width of the table.
func tableWidth() int {
	// 2 spaces per separator between 7 columns = 6 separators × 2 = 12 extra chars
	return colWidthName + colWidthState + colWidthHealth + colWidthCPU + colWidthMemory + colWidthPorts + colWidthImage + 12
}

// formatRow pads each cell to its column width and joins with "  " separators.
func formatRow(name, state, health, cpu, memory, ports, image string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		pad(name, colWidthName),
		"  ",
		pad(state, colWidthState),
		"  ",
		pad(health, colWidthHealth),
		"  ",
		pad(cpu, colWidthCPU),
		"  ",
		pad(memory, colWidthMemory),
		"  ",
		pad(ports, colWidthPorts),
		"  ",
		pad(image, colWidthImage),
	)
}

// pad right-pads s to width with spaces, truncating if longer.
func pad(s string, width int) string {
	s = truncate(s, width)
	if len(s) < width {
		s += strings.Repeat(" ", width-len(s))
	}
	return s
}

// truncate cuts s to at most width runes (appending "…" if cut).
func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
