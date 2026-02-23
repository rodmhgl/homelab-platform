package cmd

import (
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Manage and view compliance status",
	Long: `Manage and view platform compliance status including:
  - Compliance score and aggregate metrics
  - Gatekeeper policy violations
  - Trivy vulnerability scans
  - Falco runtime security events

Commands:
  summary      View overall compliance score and metrics
  policies     List active Gatekeeper policies
  violations   View policy violations
  vulns        List vulnerabilities from Trivy scans
  events       View Falco security events`,
}

func init() {
	rootCmd.AddCommand(complianceCmd)
}

// Shared color codes for compliance output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

// severityColor maps severity levels to color codes
func severityColor(severity string) string {
	switch severity {
	case "CRITICAL", "ERROR":
		return colorRed
	case "HIGH", "WARNING":
		return colorYellow
	case "MEDIUM":
		return colorYellow
	case "LOW", "NOTICE", "UNKNOWN":
		return colorGray
	default:
		return colorReset
	}
}

// formatTimestamp parses RFC3339 timestamp and returns human-readable age
func formatTimestamp(ts string) string {
	if ts == "" {
		return "-"
	}

	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}

	return formatAge(parsed)
}

// formatJSONOutput outputs JSON with indentation (consistent with other commands)
func formatJSONOutput(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
