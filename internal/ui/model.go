package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
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
}

// NewModel creates a Model backed by the given Docker client.
func NewModel(dockerClient *docker.Client) Model {
	return Model{
		dockerClient: dockerClient,
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
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.containers = msg.containers
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

	case tea.KeyMsg:
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
					return containersLoadedMsg{}
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
		}
	}

	return m, nil
}

// View renders the entire TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Table header
	header := formatRow(
		"CONTAINER", "STATE", "HEALTH", "CPU%", "MEMORY", "PORTS", "IMAGE",
	)
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", tableWidth()))
	sb.WriteString("\n")

	// Container rows
	for i, ctr := range m.containers {
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

		var rowStyled string
		if i == m.cursor {
			rowStyled = selectedRowStyle.Render("> " + row)
		} else {
			// Apply health colour to the row when not selected
			rowStyled = normalRowStyle.Render("  " + row)
			if ctr.Health != "" {
				rowStyled = healthStyle(ctr.Health).Render("  " + row)
			}
		}
		sb.WriteString(rowStyled)
		sb.WriteString("\n")
	}

	if len(m.containers) == 0 {
		sb.WriteString("  No containers found.\n")
	}

	// Status bar
	sb.WriteString("\n")
	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: " + m.err.Error()))
	} else {
		sb.WriteString(statusStyle.Render(m.status))
	}

	sb.WriteString("\n")
	sb.WriteString(statusStyle.Render("j/k move  r restart  l logs  e exec  R refresh  q quit"))

	return borderStyle.Render(sb.String())
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
