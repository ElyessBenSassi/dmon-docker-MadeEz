package main

import (
	"fmt"
	"os"

	"github.com/ElyessBenSassi/devops-tools/dmon/internal/compose"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var composeFile string

	root := &cobra.Command{
		Use:   "dmon",
		Short: "Terminal UI for monitoring Docker containers",
		Long: `dmon — a terminal UI for monitoring Docker containers.

Run it from anywhere to monitor every container on the host. When run from a
directory containing a compose file (or pointed at one with -f), the services
of that project are grouped and highlighted, and 'u' brings them up.`,
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			// Detection is non-fatal: a nil project means "run from anywhere"
			// and show all containers. Only an explicit -f that is missing errors.
			project, err := compose.Detect(composeFile, cwd)
			if err != nil {
				return err
			}

			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("connecting to Docker: %w", err)
			}
			defer dockerClient.Close()

			model := ui.NewModel(dockerClient, project)
			p := tea.NewProgram(
				model,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running TUI: %w", err)
			}
			return nil
		},
	}

	root.Flags().StringVarP(&composeFile, "file", "f", "", "Path to docker compose file (default: auto-detect)")

	return root
}
