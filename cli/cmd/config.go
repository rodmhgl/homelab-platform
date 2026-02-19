package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and modify the rdp CLI configuration file.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Create a new configuration file at ~/.rdp/config.yaml with default values.

You will be prompted to provide:
  - Platform API URL (e.g., https://api.platform.rnlabs.local)
  - Authentication token`,
	RunE: runConfigInit,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display current configuration",
	Long:  `Show the current configuration values from all sources (file, env, flags).`,
	RunE:  runConfigView,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in the config file.

Available keys:
  - api_base_url: Platform API base URL
  - auth_token: Authentication token`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)

	// Flags for config init
	configInitCmd.Flags().String("api-url", "", "Platform API URL")
	configInitCmd.Flags().String("token", "", "Authentication token")
	configInitCmd.Flags().Bool("force", false, "Overwrite existing config file")
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".rdp")
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	force, _ := cmd.Flags().GetBool("force")
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("config file already exists at %s\nUse --force to overwrite", configPath)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get values from flags or prompt
	apiURL, _ := cmd.Flags().GetString("api-url")
	token, _ := cmd.Flags().GetString("token")

	if apiURL == "" {
		fmt.Print("Platform API URL: ")
		fmt.Scanln(&apiURL)
	}

	if token == "" {
		fmt.Print("Authentication token: ")
		fmt.Scanln(&token)
	}

	// Create config content
	configContent := fmt.Sprintf(`# RNLabs Developer Platform CLI Configuration
# This file is automatically managed by 'rdp config' commands

# Platform API base URL (e.g., https://api.platform.rnlabs.local)
api_base_url: %s

# Authentication token for Platform API
auth_token: %s
`, apiURL, token)

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Configuration initialized at %s\n", configPath)
	return nil
}

func runConfigView(cmd *cobra.Command, args []string) error {
	config := GetConfig()

	fmt.Println("Current Configuration:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("API Base URL: %s\n", maskEmpty(config.APIBaseURL))
	fmt.Printf("Auth Token:   %s\n", maskToken(config.AuthToken))
	fmt.Println("─────────────────────────────────────────")

	if viper.ConfigFileUsed() != "" {
		fmt.Printf("Config file:  %s\n", viper.ConfigFileUsed())
	} else {
		fmt.Println("Config file:  (not found)")
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate key
	validKeys := map[string]bool{
		"api_base_url": true,
		"auth_token":   true,
	}

	if !validKeys[key] {
		return fmt.Errorf("invalid config key: %s\nValid keys: api_base_url, auth_token", key)
	}

	// Set the value
	viper.Set(key, value)

	// Determine config file path
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(home, ".rdp")
		configFile = filepath.Join(configDir, "config.yaml")

		// Create directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write the config file
	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Set %s in %s\n", key, configFile)
	return nil
}

// maskToken returns a masked version of the token for display
func maskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// maskEmpty returns "(not set)" for empty strings
func maskEmpty(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}
