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

var secretsListCmd = &cobra.Command{
	Use:   "list <namespace>",
	Short: "List secrets in a namespace",
	Long: `List all secrets (ExternalSecrets and Crossplane connection secrets) in a namespace.

Displays secret name, namespace, kind, sync status, available keys (not values),
source Claim (for connection secrets), and age.

Examples:
  rdp secrets list default              # List all secrets in default namespace
  rdp secrets list platform --json      # JSON output
  rdp secrets list default --kind external     # Filter by ExternalSecrets only
  rdp secrets list default --kind connection   # Filter by connection secrets only`,
	RunE: runSecretsList,
	Args: cobra.ExactArgs(1),
}

var (
	secretsListOutputJSON bool
	secretsListKind       string
)

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	secretsListCmd.Flags().BoolVarP(&secretsListOutputJSON, "json", "j", false, "Output in JSON format")
	secretsListCmd.Flags().StringVarP(&secretsListKind, "kind", "k", "", "Filter by kind: 'external' (ExternalSecrets), 'connection' (connection secrets), or '' (all)")
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	namespace := args[0]

	// Validate namespace is not empty
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/api/v1/secrets/%s", config.APIBaseURL, namespace)

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

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("invalid request: %s", string(body))
		case http.StatusNotFound:
			return fmt.Errorf("namespace '%s' not found", namespace)
		case http.StatusInternalServerError:
			return fmt.Errorf("API error: %s", string(body))
		default:
			return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
		}
	}

	// Parse response
	var listResp SecretsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Apply client-side kind filtering if specified
	if secretsListKind != "" {
		filtered := []SecretSummary{}
		for _, secret := range listResp.Secrets {
			shouldInclude := false
			switch secretsListKind {
			case "external":
				shouldInclude = secret.Kind == "ExternalSecret"
			case "connection":
				shouldInclude = secret.Kind == "Secret" && secret.SourceClaim != nil
			default:
				return fmt.Errorf("invalid kind filter: %s (must be 'external' or 'connection')", secretsListKind)
			}

			if shouldInclude {
				filtered = append(filtered, secret)
			}
		}
		listResp.Secrets = filtered
		listResp.Total = len(filtered)
	}

	// Output format
	if secretsListOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(listResp)
	}

	// Human-readable table format
	displaySecretsTable(listResp.Secrets)
	return nil
}

func displaySecretsTable(secrets []SecretSummary) {
	if len(secrets) == 0 {
		fmt.Println("No secrets found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tNAMESPACE\tKIND\tSTATUS\tKEYS\tSOURCE CLAIM\tAGE")
	fmt.Fprintln(w, "----\t---------\t----\t------\t----\t------------\t---")

	// Counters for summary
	externalSecretCount := 0
	connectionSecretCount := 0

	// Rows
	for _, secret := range secrets {
		age := formatAge(secret.CreationTimestamp)
		status := formatSecretStatus(secret.Status)
		kind := formatSecretKind(secret.Kind)
		keys := formatKeys(secret.Keys)
		sourceClaim := formatSourceClaim(secret.SourceClaim)

		// Count by type
		if secret.Kind == "ExternalSecret" {
			externalSecretCount++
		} else if secret.Kind == "Secret" && secret.SourceClaim != nil {
			connectionSecretCount++
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			secret.Name,
			secret.Namespace,
			kind,
			status,
			keys,
			sourceClaim,
			age,
		)
	}

	// Summary footer
	fmt.Fprintf(w, "\nTotal: %d secrets (%d ExternalSecrets, %d connection secrets)\n",
		len(secrets),
		externalSecretCount,
		connectionSecretCount,
	)
}
