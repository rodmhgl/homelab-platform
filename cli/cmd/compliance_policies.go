package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var compliancePoliciesCmd = &cobra.Command{
	Use:   "policies",
	Short: "List active Gatekeeper policies",
	Long: `List active Gatekeeper ConstraintTemplates and their configurations.

Shows policy name, kind, description, and scope (cluster-wide or namespaced).

Examples:
  rdp compliance policies
  rdp compliance policies --json`,
	RunE: runCompliancePolicies,
	Args: cobra.NoArgs,
}

var policiesOutputJSON bool

func init() {
	complianceCmd.AddCommand(compliancePoliciesCmd)
	compliancePoliciesCmd.Flags().BoolVarP(&policiesOutputJSON, "json", "j", false, "Output in JSON format")
}

// PoliciesResponse matches API response
type PoliciesResponse struct {
	Policies []Policy `json:"policies"`
}

// Policy matches API type
type Policy struct {
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Description string                 `json:"description"`
	Scope       []string               `json:"scope"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

func runCompliancePolicies(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()

	// Make API request
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL+"/api/v1/compliance/policies", nil)
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
	var policiesResp PoliciesResponse
	if err := json.NewDecoder(resp.Body).Decode(&policiesResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if policiesOutputJSON {
		return formatJSONOutput(policiesResp)
	}

	// Human-readable table format
	displayPoliciesTable(policiesResp.Policies)
	return nil
}

func displayPoliciesTable(policies []Policy) {
	if len(policies) == 0 {
		fmt.Println("No policies found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tKIND\tSCOPE\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t-----\t-----------")

	// Rows
	for _, policy := range policies {
		scope := "Cluster"
		if len(policy.Scope) > 0 {
			scope = fmt.Sprintf("%d namespaces", len(policy.Scope))
		}

		// Truncate long descriptions
		description := policy.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			policy.Name,
			policy.Kind,
			scope,
			description,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d policies\n", len(policies))
}
