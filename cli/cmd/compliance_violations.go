package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var complianceViolationsCmd = &cobra.Command{
	Use:   "violations",
	Short: "View policy violations",
	Long: `View Gatekeeper policy violations with detailed information.

Shows constraint name, resource kind, resource path, namespace, and violation message.
Supports filtering by namespace.

Examples:
  rdp compliance violations
  rdp compliance violations --namespace platform
  rdp compliance violations --json`,
	RunE: runComplianceViolations,
	Args: cobra.NoArgs,
}

var (
	violationsOutputJSON bool
	violationsNamespace  string
)

func init() {
	complianceCmd.AddCommand(complianceViolationsCmd)
	complianceViolationsCmd.Flags().BoolVarP(&violationsOutputJSON, "json", "j", false, "Output in JSON format")
	complianceViolationsCmd.Flags().StringVarP(&violationsNamespace, "namespace", "n", "", "Filter by namespace")
}

// ViolationsResponse matches API response
type ViolationsResponse struct {
	Violations []Violation `json:"violations"`
}

// Violation matches API type
type Violation struct {
	ConstraintName string `json:"constraintName"`
	ConstraintKind string `json:"constraintKind"`
	Resource       string `json:"resource"`
	Namespace      string `json:"namespace"`
	Message        string `json:"message"`
	Timestamp      string `json:"timestamp,omitempty"`
}

func runComplianceViolations(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()

	// Build request URL with query parameters
	reqURL := config.APIBaseURL + "/api/v1/compliance/violations"
	if violationsNamespace != "" {
		params := url.Values{}
		params.Add("namespace", violationsNamespace)
		reqURL += "?" + params.Encode()
	}

	// Make API request
	client := &http.Client{Timeout: 10 * time.Second}
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
	var violationsResp ViolationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&violationsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if violationsOutputJSON {
		return formatJSONOutput(violationsResp)
	}

	// Human-readable table format
	displayViolationsTable(violationsResp.Violations)
	return nil
}

func displayViolationsTable(violations []Violation) {
	if len(violations) == 0 {
		fmt.Println("No violations found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "CONSTRAINT\tKIND\tRESOURCE\tNAMESPACE\tMESSAGE")
	fmt.Fprintln(w, "----------\t----\t--------\t---------\t-------")

	// Rows
	for _, v := range violations {
		// Truncate long values
		constraint := truncateString(v.ConstraintName, 20)
		kind := truncateString(v.ConstraintKind, 15)
		resource := truncateString(v.Resource, 30)
		namespace := v.Namespace
		if namespace == "" {
			namespace = "-"
		}
		message := truncateString(v.Message, 50)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			constraint,
			kind,
			resource,
			namespace,
			message,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d violations\n", len(violations))
}
