package cmd

import (
	"github.com/spf13/cobra"
)

var portalCmd = &cobra.Command{
	Use:   "portal",
	Short: "Portal UI operations",
	Long: `Open and interact with the Platform Portal web interface.

The Portal UI provides a visual dashboard for:
  - Application status and health
  - Infrastructure resources (Claims)
  - Compliance score and violations
  - Vulnerability feed (CVE scanning)
  - Security events (Falco alerts)

Access the Portal at: portal.rdp.azurelaboratory.com`,
}

func init() {
	rootCmd.AddCommand(portalCmd)
}
