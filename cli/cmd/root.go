package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *Config
)

// Config represents the CLI configuration
type Config struct {
	APIBaseURL string `mapstructure:"api_base_url"`
	AuthToken  string `mapstructure:"auth_token"`
	PortalURL  string `mapstructure:"portal_url"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rdp",
	Short: "RNLabs Developer Platform CLI",
	Long: `rdp is the command-line interface for the RNLabs Internal Developer Platform.

It provides self-service operations for:
  - Scaffolding new applications
  - Managing infrastructure (storage, vaults)
  - Viewing application status and compliance
  - Investigating issues with AI assistance
  - Managing secrets

All operations go through the Platform API, ensuring consistent GitOps workflows.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rdp/config.yaml)")
	rootCmd.PersistentFlags().String("api-url", "", "Platform API base URL (overrides config file)")
	rootCmd.PersistentFlags().String("token", "", "Authentication token (overrides config file)")

	// Bind flags to viper
	viper.BindPFlag("api_base_url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("auth_token", rootCmd.PersistentFlags().Lookup("token"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rdp" (without extension)
		configDir := filepath.Join(home, ".rdp")
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variable overrides (RDP_API_BASE_URL, RDP_AUTH_TOKEN)
	viper.SetEnvPrefix("RDP")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		// Config file found and successfully read
		// (silent success - only report errors)
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// Config file was found but another error was produced
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal config into struct
	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	if cfg == nil {
		cfg = &Config{}
		viper.Unmarshal(cfg)
	}
	return cfg
}

// ValidateConfig checks that required configuration is present
func ValidateConfig() error {
	config := GetConfig()

	if config.APIBaseURL == "" {
		return fmt.Errorf("Platform API URL not configured. Set via:\n" +
			"  - Config file: ~/.rdp/config.yaml\n" +
			"  - Flag: --api-url\n" +
			"  - Environment: RDP_API_BASE_URL")
	}

	if config.AuthToken == "" {
		return fmt.Errorf("Authentication token not configured. Set via:\n" +
			"  - Config file: ~/.rdp/config.yaml\n" +
			"  - Flag: --token\n" +
			"  - Environment: RDP_AUTH_TOKEN")
	}

	return nil
}
