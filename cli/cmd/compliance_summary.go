package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var complianceSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "View overall compliance score and metrics",
	Long: `View overall compliance score and aggregate metrics including:
  - Compliance score (0-100)
  - Total policy violations
  - Total vulnerabilities by severity
  - Total security events

Examples:
  rdp compliance summary
  rdp compliance summary --json`,
	RunE: runComplianceSummary,
	Args: cobra.NoArgs,
}

var summaryOutputJSON bool

func init() {
	complianceCmd.AddCommand(complianceSummaryCmd)
	complianceSummaryCmd.Flags().BoolVarP(&summaryOutputJSON, "json", "j", false, "Output in JSON format")
}

// ComplianceSummaryResponse matches API response exactly
type ComplianceSummaryResponse struct {
	ComplianceScore           float64        `json:"complianceScore"`
	TotalViolations           int            `json:"totalViolations"`
	TotalVulnerabilities      int            `json:"totalVulnerabilities"`
	ViolationsBySeverity      map[string]int `json:"violationsBySeverity"`
	VulnerabilitiesBySeverity map[string]int `json:"vulnerabilitiesBySeverity"`
}

func runComplianceSummary(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()

	// Make API request
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL+"/api/v1/compliance/summary", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.AuthToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var summary ComplianceSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if summaryOutputJSON {
		return formatJSONOutput(summary)
	}

	// Human-readable format
	displayComplianceSummary(summary)
	return nil
}

func displayComplianceSummary(summary ComplianceSummaryResponse) {
	fmt.Println("┌─ Compliance Summary ──────────────────────────────────────┐")
	fmt.Println("│                                                            │")

	// Compliance score with color coding
	scoreIcon := formatComplianceScoreIcon(summary.ComplianceScore)
	scoreColor := getComplianceScoreColor(summary.ComplianceScore)
	fmt.Printf("│  Compliance Score:  %s%s%.0f%s                                 \n",
		scoreColor, scoreIcon, summary.ComplianceScore, colorReset)
	fmt.Println("│                                                            │")

	// Metrics
	fmt.Printf("│  Policy Violations: %d                                   \n", summary.TotalViolations)

	// Vulnerabilities with severity breakdown
	criticalCVEs := summary.VulnerabilitiesBySeverity["CRITICAL"]
	highCVEs := summary.VulnerabilitiesBySeverity["HIGH"]
	mediumCVEs := summary.VulnerabilitiesBySeverity["MEDIUM"]
	lowCVEs := summary.VulnerabilitiesBySeverity["LOW"]

	if criticalCVEs > 0 || highCVEs > 0 {
		fmt.Printf("│  Vulnerabilities:   %d (%s%d Critical%s, %s%d High%s)            \n",
			summary.TotalVulnerabilities,
			colorRed, criticalCVEs, colorReset,
			colorYellow, highCVEs, colorReset)
	} else if mediumCVEs > 0 {
		fmt.Printf("│  Vulnerabilities:   %d (%s%d Medium%s, %d Low)              \n",
			summary.TotalVulnerabilities,
			colorYellow, mediumCVEs, colorReset,
			lowCVEs)
	} else {
		fmt.Printf("│  Vulnerabilities:   %d                                   \n", summary.TotalVulnerabilities)
	}

	fmt.Println("│                                                            │")
	fmt.Println("└────────────────────────────────────────────────────────────┘")
}

func formatComplianceScoreIcon(score float64) string {
	if score >= 90 {
		return "✓ "
	} else if score >= 70 {
		return "⚠ "
	}
	return "✗ "
}

func getComplianceScoreColor(score float64) string {
	if score >= 90 {
		return colorGreen
	} else if score >= 70 {
		return colorYellow
	}
	return colorRed
}
