package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var complianceVulnsCmd = &cobra.Command{
	Use:   "vulns",
	Short: "List vulnerabilities from Trivy scans",
	Long: `List vulnerabilities (CVEs) found by Trivy Operator scans.

Shows severity, CVE ID, affected image, package, fixed version, and workload.
Supports filtering by severity level.

Examples:
  rdp compliance vulns
  rdp compliance vulns --severity CRITICAL
  rdp compliance vulns --severity HIGH
  rdp compliance vulns --json`,
	RunE: runComplianceVulns,
	Args: cobra.NoArgs,
}

var (
	vulnsOutputJSON bool
	vulnsSeverity   string
)

func init() {
	complianceCmd.AddCommand(complianceVulnsCmd)
	complianceVulnsCmd.Flags().BoolVarP(&vulnsOutputJSON, "json", "j", false, "Output in JSON format")
	complianceVulnsCmd.Flags().StringVar(&vulnsSeverity, "severity", "", "Filter by severity (CRITICAL|HIGH|MEDIUM|LOW)")
}

// VulnerabilitiesResponse matches API response
type VulnerabilitiesResponse struct {
	Vulnerabilities []VulnerabilityItem `json:"vulnerabilities"`
}

// VulnerabilityItem matches API type
type VulnerabilityItem struct {
	Image           string  `json:"image"`
	Namespace       string  `json:"namespace"`
	Workload        string  `json:"workload"`
	CVEID           string  `json:"cveId"`
	Severity        string  `json:"severity"`
	Score           float64 `json:"score,omitempty"`
	AffectedPackage string  `json:"affectedPackage"`
	FixedVersion    string  `json:"fixedVersion,omitempty"`
	PrimaryLink     string  `json:"primaryLink,omitempty"`
}

func runComplianceVulns(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	// Validate severity filter
	if vulnsSeverity != "" {
		severityUpper := strings.ToUpper(vulnsSeverity)
		validSeverities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}
		valid := false
		for _, s := range validSeverities {
			if severityUpper == s {
				valid = true
				vulnsSeverity = severityUpper
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid severity: %s (must be CRITICAL, HIGH, MEDIUM, or LOW)", vulnsSeverity)
		}
	}

	config := GetConfig()

	// Build request URL with query parameters
	reqURL := config.APIBaseURL + "/api/v1/compliance/vulnerabilities"
	if vulnsSeverity != "" {
		params := url.Values{}
		params.Add("severity", vulnsSeverity)
		reqURL += "?" + params.Encode()
	}

	// Make API request
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", reqURL, nil)
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
	var vulnsResp VulnerabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&vulnsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if vulnsOutputJSON {
		return formatJSONOutput(vulnsResp)
	}

	// Human-readable table format
	displayVulnsTable(vulnsResp.Vulnerabilities)
	return nil
}

func displayVulnsTable(vulns []VulnerabilityItem) {
	if len(vulns) == 0 {
		fmt.Println("No vulnerabilities found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "SEVERITY\tCVE-ID\tIMAGE\tPACKAGE\tFIXED\tWORKLOAD")
	fmt.Fprintln(w, "--------\t------\t-----\t-------\t-----\t--------")

	// Rows
	for _, v := range vulns {
		// Color-code severity
		sevColor := severityColor(v.Severity)
		severityDisplay := fmt.Sprintf("%s%s%s", sevColor, v.Severity, colorReset)

		// Truncate long values
		image := truncateString(v.Image, 30)
		pkg := truncateString(v.AffectedPackage, 20)
		fixed := v.FixedVersion
		if fixed == "" {
			fixed = "-"
		} else {
			fixed = truncateString(fixed, 15)
		}
		workload := truncateString(v.Workload, 25)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			severityDisplay,
			v.CVEID,
			image,
			pkg,
			fixed,
			workload,
		)
	}

	// Summary with severity breakdown
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0
	for _, v := range vulns {
		switch v.Severity {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		case "LOW":
			lowCount++
		}
	}

	fmt.Fprintf(w, "\nTotal: %d vulnerabilities", len(vulns))
	if criticalCount > 0 || highCount > 0 {
		fmt.Fprintf(w, " (%s%d Critical%s, %s%d High%s, %d Medium, %d Low)\n",
			colorRed, criticalCount, colorReset,
			colorYellow, highCount, colorReset,
			mediumCount, lowCount)
	} else {
		fmt.Fprintf(w, " (%d Medium, %d Low)\n", mediumCount, lowCount)
	}
}
