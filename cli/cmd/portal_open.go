package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	portalURLFlag string
	printOnly     bool
)

var portalOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the Portal UI in your default browser",
	Long: `Open the Platform Portal web interface in your default browser.

The Portal URL is determined in the following order of precedence:
  1. --url flag (highest priority)
  2. portal_url in config file (~/.rdp/config.yaml)
  3. Derived from api_base_url (replace 'api.' with 'portal.')
  4. Default: http://portal.rdp.azurelaboratory.com

Examples:
  # Open production Portal
  rdp portal open

  # Open local dev Portal (port-forward)
  rdp portal open --url http://localhost:8080

  # Print URL without opening browser
  rdp portal open --print`,
	RunE: runPortalOpen,
}

func init() {
	portalCmd.AddCommand(portalOpenCmd)

	portalOpenCmd.Flags().StringVar(&portalURLFlag, "url", "", "Override Portal URL")
	portalOpenCmd.Flags().BoolVar(&printOnly, "print", false, "Print URL only, don't open browser")
}

func runPortalOpen(cmd *cobra.Command, args []string) error {
	// Determine Portal URL
	portalURL := getPortalURL()

	// Validate URL format
	if _, err := url.Parse(portalURL); err != nil {
		return fmt.Errorf("invalid Portal URL: %v", err)
	}

	// Print mode - just output URL and exit
	if printOnly {
		fmt.Println(portalURL)
		return nil
	}

	// Attempt to open browser
	if err := openBrowser(portalURL); err != nil {
		// Fallback: print URL with instructions
		fmt.Println("Unable to open browser automatically.")
		fmt.Printf("Open this URL in your browser: %s\n", portalURL)
		return nil // Not a fatal error - user can manually open
	}

	fmt.Printf("Opening Portal UI in browser: %s\n", portalURL)
	return nil
}

// getPortalURL determines the Portal URL using precedence rules
func getPortalURL() string {
	// 1. Flag override (highest priority)
	if portalURLFlag != "" {
		return portalURLFlag
	}

	// 2. Config file portal_url
	if viper.IsSet("portal_url") {
		if portalURL := viper.GetString("portal_url"); portalURL != "" {
			return portalURL
		}
	}

	// 3. Derive from api_base_url
	if viper.IsSet("api_base_url") {
		if apiURL := viper.GetString("api_base_url"); apiURL != "" {
			// Try multiple replacement patterns to derive portal URL from API URL
			portalURL := apiURL

			// Pattern 1: 'api.' prefix (e.g., api.rdp.com -> portal.rdp.com)
			if strings.Contains(portalURL, "api.") {
				portalURL = strings.Replace(portalURL, "api.", "portal.", 1)
				return portalURL
			}

			// Pattern 2: 'api-' prefix (e.g., api-rdp.com -> portal-rdp.com)
			if strings.Contains(portalURL, "api-") {
				portalURL = strings.Replace(portalURL, "api-", "portal-", 1)
				return portalURL
			}

			// Pattern 3: '-api.' or '-api-' infix (e.g., platform-api.rdp.com -> platform-portal.rdp.com)
			if strings.Contains(portalURL, "-api.") {
				portalURL = strings.Replace(portalURL, "-api.", "-portal.", 1)
				return portalURL
			}
			if strings.Contains(portalURL, "-api-") {
				portalURL = strings.Replace(portalURL, "-api-", "-portal-", 1)
				return portalURL
			}

			// Pattern 4: 'platform' anywhere in URL -> replace with 'portal'
			// This handles cases like platform.rdp.com or my-platform.com
			if strings.Contains(portalURL, "platform") {
				portalURL = strings.Replace(portalURL, "platform", "portal", 1)
				return portalURL
			}

			// If no pattern matches, fall through to default
		}
	}

	// 4. Hardcoded default
	return "http://portal.rdp.azurelaboratory.com"
}

// openBrowser attempts to open the URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		// Try xdg-open (standard on most Linux desktops and WSL with xdg-utils)
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		// macOS
		cmd = exec.Command("open", url)
	case "windows":
		// Windows
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
