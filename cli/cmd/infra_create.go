package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rodmhgl/homelab-platform/cli/internal/tui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create infrastructure resources",
	Long:  "Create Crossplane Claims for infrastructure resources via interactive forms",
}

var createStorageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Create a StorageBucket Claim",
	Long: `Interactively create an Azure Storage Account via Crossplane StorageBucket Claim.

This wizard will guide you through:
- Naming the storage bucket
- Selecting namespace, location, tier, and redundancy
- Configuring versioning
- Specifying the GitHub repository for GitOps

The Platform API will commit the Claim YAML to your repository,
and Argo CD will sync it to the cluster within 60 seconds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate config
		if err := ValidateConfig(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		config := GetConfig()

		// Launch TUI
		model := tui.NewStorageModel(config.APIBaseURL, config.AuthToken)
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		// Check if user quit vs succeeded
		if m, ok := finalModel.(tui.StorageModel); ok {
			if m.State() == "success" {
				return nil
			}
		}

		return nil // User canceled
	},
}

var createVaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Create a Vault Claim",
	Long: `Interactively create an Azure Key Vault via Crossplane Vault Claim.

This wizard will guide you through:
- Naming the vault
- Selecting namespace, location, and SKU
- Configuring soft delete retention days
- Specifying the GitHub repository for GitOps

The Platform API will commit the Claim YAML to your repository,
and Argo CD will sync it to the cluster within 60 seconds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate config
		if err := ValidateConfig(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		config := GetConfig()

		// Launch TUI
		model := tui.NewVaultModel(config.APIBaseURL, config.AuthToken)
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		// Check if user quit vs succeeded
		if m, ok := finalModel.(tui.VaultModel); ok {
			if m.State() == "success" {
				return nil
			}
		}

		return nil // User canceled
	},
}

func init() {
	infraCmd.AddCommand(createCmd)
	createCmd.AddCommand(createStorageCmd)
	createCmd.AddCommand(createVaultCmd)
}
