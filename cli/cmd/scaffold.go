package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rodmhgl/homelab-platform/cli/internal/tui"
	"github.com/spf13/cobra"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Scaffold new services from templates",
	Long:  "Create new services from Copier templates with GitOps onboarding",
}

var scaffoldCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service from template",
	Long: `Interactively create a new service from a Copier template.

This wizard will guide you through:
- Selecting a template (go-service, python-service)
- Configuring project metadata (name, description, ports)
- Enabling optional features (gRPC, database, storage, Key Vault)
- Specifying the GitHub repository for source code

The Platform API will:
1. Execute the Copier template
2. Create a GitHub repository
3. Push scaffolded code
4. Add apps/{name}/config.json to platform repo
5. Argo CD auto-discovers and syncs the application

Your service will be live in the cluster within 60 seconds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate config
		if err := ValidateConfig(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		config := GetConfig()

		// Launch TUI
		model := tui.NewScaffoldModel(config.APIBaseURL, config.AuthToken)
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		// Check if user quit vs succeeded
		if m, ok := finalModel.(tui.ScaffoldModel); ok {
			if m.State() == "success" {
				return nil
			}
		}

		return nil // User canceled
	},
}

func init() {
	rootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.AddCommand(scaffoldCreateCmd)
}
