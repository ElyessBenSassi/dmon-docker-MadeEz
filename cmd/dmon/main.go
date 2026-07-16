package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/compose"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/docker"
	"github.com/ElyessBenSassi/devops-tools/dmon/internal/ui"
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

Run from a directory containing a docker-compose file, or supply -f to
point at a specific compose file.`,
		Version: version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			composeFilePath, err := compose.DetectComposeFile(composeFile, cwd)
			if err != nil {
				return err
			}

			_ = composeFilePath // available for future filtering by project

			dockerClient, err := docker.NewClient()
			if err != nil {
				return fmt.Errorf("connecting to Docker: %w", err)
			}
			defer dockerClient.Close()

			model := ui.NewModel(dockerClient)
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
