package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Column widths
	colWidthName   = 30
	colWidthState  = 10
	colWidthHealth = 10
	colWidthCPU    = 8
	colWidthMemory = 22
	colWidthPorts  = 22
	colWidthImage  = 40

	// Colors
	colorHeader    = lipgloss.Color("240") // dark gray
	colorSelected  = lipgloss.Color("33")  // blue
	colorError     = lipgloss.Color("196") // red
	colorStatus    = lipgloss.Color("243") // mid gray
	colorHealthy   = lipgloss.Color("46")  // green
	colorUnhealthy = lipgloss.Color("196") // red
	colorStarting  = lipgloss.Color("220") // yellow
	colorProject   = lipgloss.Color("39")  // cyan — compose project accent
	colorPath      = lipgloss.Color("240") // dark gray — file path
	colorSection   = lipgloss.Color("245") // light gray — section labels
	colorPrompt    = lipgloss.Color("214") // orange — active prompt

	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHeader).
			PaddingBottom(0)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSelected)

	normalRowStyle = lipgloss.NewStyle()

	statusStyle = lipgloss.NewStyle().
			Foreground(colorStatus).
			PaddingTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			PaddingTop(1)

	healthyStyle = lipgloss.NewStyle().
			Foreground(colorHealthy)

	unhealthyStyle = lipgloss.NewStyle().
			Foreground(colorUnhealthy)

	startingStyle = lipgloss.NewStyle().
			Foreground(colorStarting)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238"))

	// projectStyle renders the compose project name in the title bar.
	projectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorProject)

	// pathStyle renders the compose file path / "no compose file" hint.
	pathStyle = lipgloss.NewStyle().
			Foreground(colorPath)

	// sectionStyle renders the "compose · <project>" / "other containers" labels.
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSection)

	// projectGutterStyle renders the accent bar on project-member rows.
	projectGutterStyle = lipgloss.NewStyle().
				Foreground(colorProject)

	// projectRowStyle renders a project-member row that has no health colour.
	projectRowStyle = lipgloss.NewStyle().
			Foreground(colorProject)

	// promptStyle renders the active profile-name input line.
	promptStyle = lipgloss.NewStyle().
			Foreground(colorPrompt)
)

// healthStyle returns the appropriate style for a health string.
func healthStyle(health string) lipgloss.Style {
	switch health {
	case "healthy":
		return healthyStyle
	case "unhealthy":
		return unhealthyStyle
	case "starting":
		return startingStyle
	default:
		return normalRowStyle
	}
}
