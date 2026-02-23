package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Manage infrastructure resources (Claims)",
	Long: `Manage Crossplane Claims for Azure infrastructure resources.

Commands:
  list               List all infrastructure Claims
  status             Show detailed status for a specific Claim
  create storage     Create StorageBucket Claim (interactive)
  create vault       Create Vault Claim (interactive)
  delete             Delete Claim (pending implementation)`,
}

var infraListCmd = &cobra.Command{
	Use:   "list [storage|vaults]",
	Short: "List infrastructure Claims",
	Long: `List all Crossplane Claims or filter by type.

Examples:
  rdp infra list              # List all Claims
  rdp infra list storage      # List only StorageBucket Claims
  rdp infra list vaults       # List only Vault Claims`,
	RunE: runInfraList,
	Args: cobra.MaximumNArgs(1),
}

var infraStatusCmd = &cobra.Command{
	Use:   "status <kind> <name>",
	Short: "Show detailed status for a Claim",
	Long: `Show detailed status for a specific Crossplane Claim including:
  - Claim status and conditions
  - Composite resource status
  - Managed Azure resources
  - Recent Kubernetes events

Examples:
  rdp infra status storage my-bucket
  rdp infra status vault my-vault`,
	RunE: runInfraStatus,
	Args: cobra.ExactArgs(2),
}

// Flags
var (
	infraNamespace string
	infraOutputJSON bool
)

func init() {
	rootCmd.AddCommand(infraCmd)
	infraCmd.AddCommand(infraListCmd)
	infraCmd.AddCommand(infraStatusCmd)

	// Flags for list command
	infraListCmd.Flags().StringVarP(&infraNamespace, "namespace", "n", "", "Filter by namespace (default: all namespaces)")
	infraListCmd.Flags().BoolVarP(&infraOutputJSON, "json", "j", false, "Output in JSON format")

	// Flags for status command
	infraStatusCmd.Flags().StringVarP(&infraNamespace, "namespace", "n", "default", "Namespace of the Claim")
	infraStatusCmd.Flags().BoolVarP(&infraOutputJSON, "json", "j", false, "Output in JSON format")
}

// ClaimSummary matches the API response type
type ClaimSummary struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Kind              string            `json:"kind"`
	Status            string            `json:"status"`
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	ConnectionSecret  string            `json:"connectionSecret,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// ListClaimsResponse matches the API response
type ListClaimsResponse struct {
	Claims []ClaimSummary `json:"claims"`
	Total  int            `json:"total"`
}

func runInfraList(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()

	// Determine endpoint based on filter argument
	endpoint := "/api/v1/infra"
	if len(args) > 0 {
		filter := strings.ToLower(args[0])
		switch filter {
		case "storage":
			endpoint = "/api/v1/infra/storage"
		case "vaults", "vault":
			endpoint = "/api/v1/infra/vaults"
		default:
			return fmt.Errorf("invalid filter: %s (must be 'storage' or 'vaults')", filter)
		}
	}

	// Make API request
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL+endpoint, nil)
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
	var listResp ListClaimsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter by namespace if specified
	if infraNamespace != "" {
		filtered := []ClaimSummary{}
		for _, claim := range listResp.Claims {
			if claim.Namespace == infraNamespace {
				filtered = append(filtered, claim)
			}
		}
		listResp.Claims = filtered
		listResp.Total = len(filtered)
	}

	// Output format
	if infraOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(listResp)
	}

	// Human-readable table format
	displayClaimsTable(listResp.Claims)
	return nil
}

func displayClaimsTable(claims []ClaimSummary) {
	if len(claims) == 0 {
		fmt.Println("No Claims found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tNAMESPACE\tKIND\tSTATUS\tREADY\tSYNCED\tAGE\tCONNECTION SECRET")
	fmt.Fprintln(w, "----\t---------\t----\t------\t-----\t------\t---\t-----------------")

	// Rows
	for _, claim := range claims {
		age := formatAge(claim.CreationTimestamp)
		readyIcon := formatBoolIcon(claim.Ready)
		syncedIcon := formatBoolIcon(claim.Synced)
		statusDisplay := formatStatus(claim.Status, claim.Ready, claim.Synced)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			claim.Name,
			claim.Namespace,
			claim.Kind,
			statusDisplay,
			readyIcon,
			syncedIcon,
			age,
			claim.ConnectionSecret,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d Claims\n", len(claims))
}

func runInfraStatus(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	kind := args[0]
	name := args[1]

	// Normalize kind (storage -> StorageBucket, vault -> Vault)
	normalizedKind := normalizeKindForAPI(kind)

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/infra/%s/%s?namespace=%s",
		config.APIBaseURL,
		normalizedKind,
		name,
		infraNamespace,
	)

	// Make API request
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
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
	var statusResp GetResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if infraOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(statusResp)
	}

	// Human-readable format
	displayResourceStatus(statusResp)
	return nil
}

// GetResourceResponse matches the API response for detailed status
type GetResourceResponse struct {
	Claim     ClaimResource      `json:"claim"`
	Composite *CompositeResource `json:"composite,omitempty"`
	Managed   []ManagedResource  `json:"managed"`
	Events    []KubernetesEvent  `json:"events"`
}

type ClaimResource struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Kind              string            `json:"kind"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	Status            string            `json:"status"`
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	ConnectionSecret  string            `json:"connectionSecret,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
}

type CompositeResource struct {
	Name              string        `json:"name"`
	Kind              string        `json:"kind"`
	Status            string        `json:"status"`
	Synced            bool          `json:"synced"`
	Ready             bool          `json:"ready"`
	CreationTimestamp time.Time     `json:"creationTimestamp"`
	ResourceRefs      []ResourceRef `json:"resourceRefs,omitempty"`
}

type ManagedResource struct {
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	Group        string    `json:"group"`
	Status       string    `json:"status"`
	Synced       bool      `json:"synced"`
	Ready        bool      `json:"ready"`
	ExternalName string    `json:"externalName,omitempty"`
	Message      string    `json:"message,omitempty"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

type ResourceRef struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion,omitempty"`
}

type KubernetesEvent struct {
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	InvolvedObject string    `json:"involvedObject"`
	Source         string    `json:"source,omitempty"`
	Count          int32     `json:"count,omitempty"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
}

func displayResourceStatus(resp GetResourceResponse) {
	// Claim summary
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Claim: %s/%s\n", resp.Claim.Namespace, resp.Claim.Name)
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("┌─ Claim Details ───────────────────────────────────────────┐")
	fmt.Printf("│ Kind:             %s\n", resp.Claim.Kind)
	fmt.Printf("│ Status:           %s\n", formatStatus(resp.Claim.Status, resp.Claim.Ready, resp.Claim.Synced))
	fmt.Printf("│ Ready:            %s\n", formatBoolStatus(resp.Claim.Ready))
	fmt.Printf("│ Synced:           %s\n", formatBoolStatus(resp.Claim.Synced))
	fmt.Printf("│ Age:              %s\n", formatAge(resp.Claim.CreationTimestamp))
	if resp.Claim.ConnectionSecret != "" {
		fmt.Printf("│ Connection Secret: %s\n", resp.Claim.ConnectionSecret)
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Composite resource
	if resp.Composite != nil {
		fmt.Println("┌─ Composite Resource ──────────────────────────────────────┐")
		fmt.Printf("│ Name:   %s\n", resp.Composite.Name)
		fmt.Printf("│ Kind:   %s\n", resp.Composite.Kind)
		fmt.Printf("│ Status: %s\n", formatStatus(resp.Composite.Status, resp.Composite.Ready, resp.Composite.Synced))
		fmt.Printf("│ Ready:  %s\n", formatBoolStatus(resp.Composite.Ready))
		fmt.Printf("│ Synced: %s\n", formatBoolStatus(resp.Composite.Synced))
		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}

	// Managed resources
	if len(resp.Managed) > 0 {
		fmt.Println("┌─ Managed Azure Resources ─────────────────────────────────┐")
		for i, mr := range resp.Managed {
			if i > 0 {
				fmt.Println("├───────────────────────────────────────────────────────────┤")
			}
			fmt.Printf("│ %-15s %s\n", "Kind:", mr.Kind)
			fmt.Printf("│ %-15s %s\n", "Name:", mr.Name)
			if mr.ExternalName != "" {
				fmt.Printf("│ %-15s %s\n", "Azure Name:", mr.ExternalName)
			}
			fmt.Printf("│ %-15s %s\n", "Status:", formatStatus(mr.Status, mr.Ready, mr.Synced))
			fmt.Printf("│ %-15s %s\n", "Ready:", formatBoolStatus(mr.Ready))
			fmt.Printf("│ %-15s %s\n", "Synced:", formatBoolStatus(mr.Synced))
			if mr.Message != "" {
				// Truncate long messages
				message := mr.Message
				if len(message) > 50 {
					message = message[:47] + "..."
				}
				fmt.Printf("│ %-15s %s\n", "Message:", message)
			}
		}
		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}

	// Events
	if len(resp.Events) > 0 {
		fmt.Println("┌─ Recent Events ───────────────────────────────────────────┐")
		// Show last 5 events
		maxEvents := 5
		startIdx := 0
		if len(resp.Events) > maxEvents {
			startIdx = len(resp.Events) - maxEvents
		}

		for i := startIdx; i < len(resp.Events); i++ {
			event := resp.Events[i]
			age := formatAge(event.LastTimestamp)
			typeIcon := "ℹ"
			if event.Type == "Warning" {
				typeIcon = "⚠"
			}

			if i > startIdx {
				fmt.Println("├───────────────────────────────────────────────────────────┤")
			}

			fmt.Printf("│ %s %s (%s)\n", typeIcon, event.Reason, age)

			// Wrap long messages
			message := event.Message
			if len(message) > 55 {
				message = message[:52] + "..."
			}
			fmt.Printf("│   %s\n", message)
		}
		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}
}

// Helper functions

func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() >= 1 {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh", hours)
	} else if duration.Minutes() >= 1 {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

func formatBoolIcon(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

func formatBoolStatus(b bool) string {
	if b {
		return "✓ True"
	}
	return "✗ False"
}

func formatStatus(status string, ready, synced bool) string {
	icon := "○"
	if ready && synced {
		icon = "✓"
	} else if !ready || !synced {
		icon = "⚠"
	}

	return fmt.Sprintf("%s %s", icon, status)
}

func normalizeKindForAPI(kind string) string {
	lower := strings.ToLower(kind)
	switch lower {
	case "storage", "storagebucket", "storagebuckets":
		return "storage"
	case "vault", "vaults", "keyvault", "keyvaults":
		return "vault"
	default:
		return kind
	}
}
