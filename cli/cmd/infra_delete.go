package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var infraDeleteCmd = &cobra.Command{
	Use:   "delete <kind> <name>",
	Short: "Delete a Claim (commits removal to app repo)",
	Long: `Delete a Crossplane Claim via GitOps.

This command removes the Claim YAML from your application's Git repository.
Argo CD will then sync the change and remove the Claim from the cluster.
Crossplane will delete all managed Azure resources.

⚠️  WARNING: This is a destructive operation. Azure resources will be permanently deleted.

Examples:
  rdp infra delete storage demo-storage --repo-owner myorg --repo-name myapp
  rdp infra delete vault prod-vault --namespace production --repo-owner myorg --repo-name myapp --force`,
	RunE: runInfraDelete,
	Args: cobra.ExactArgs(2),
}

// Flags for delete command
var (
	infraDeleteRepoOwner string
	infraDeleteRepoName  string
	infraDeleteForce     bool
)

func init() {
	infraCmd.AddCommand(infraDeleteCmd)

	infraDeleteCmd.Flags().StringVarP(&infraNamespace, "namespace", "n", "default", "Namespace of the Claim")
	infraDeleteCmd.Flags().StringVar(&infraDeleteRepoOwner, "repo-owner", "", "GitHub organization or user (required)")
	infraDeleteCmd.Flags().StringVar(&infraDeleteRepoName, "repo-name", "", "GitHub repository name (required)")
	infraDeleteCmd.Flags().BoolVar(&infraDeleteForce, "force", false, "Skip confirmation prompt")
	infraDeleteCmd.Flags().BoolVarP(&infraOutputJSON, "json", "j", false, "Output in JSON format")

	infraDeleteCmd.MarkFlagRequired("repo-owner")
	infraDeleteCmd.MarkFlagRequired("repo-name")
}

// DeleteClaimRequest matches the API request body
type DeleteClaimRequest struct {
	RepoOwner string `json:"repoOwner"`
	RepoName  string `json:"repoName"`
}

// DeleteClaimResponse matches the API response
type DeleteClaimResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CommitSHA string `json:"commitSha"`
	FilePath  string `json:"filePath"`
	RepoURL   string `json:"repoUrl"`
}

func runInfraDelete(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	kind := args[0]
	name := args[1]

	// Normalize kind (storage -> StorageBucket, vault -> Vault)
	normalizedKind := normalizeKindForAPI(kind)

	// Display kind in human-readable format
	displayKind := normalizedKind
	if normalizedKind == "storage" {
		displayKind = "StorageBucket"
	} else if normalizedKind == "vault" {
		displayKind = "Vault"
	}

	// Confirmation prompt (unless --force)
	if !infraDeleteForce {
		confirmed, err := confirmDeletion(displayKind, name, infraNamespace, infraDeleteRepoOwner, infraDeleteRepoName)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			fmt.Println("\n✗ Deletion cancelled")
			return nil
		}
	}

	// Build request body
	deleteReq := DeleteClaimRequest{
		RepoOwner: infraDeleteRepoOwner,
		RepoName:  infraDeleteRepoName,
	}

	reqBody, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/infra/%s/%s?namespace=%s",
		config.APIBaseURL,
		normalizedKind,
		name,
		infraNamespace,
	)

	// Make API request
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("DELETE", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Claim not found in repository (may have been already deleted)")
		}
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var deleteResp DeleteClaimResponse
	if err := json.NewDecoder(resp.Body).Decode(&deleteResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if infraOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(deleteResp)
	}

	// Human-readable output
	displayDeleteSuccess(deleteResp)
	return nil
}

func confirmDeletion(kind, name, namespace, repoOwner, repoName string) (bool, error) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  ⚠️  WARNING: Destructive Operation                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("You are about to delete infrastructure:")
	fmt.Println()
	fmt.Printf("  Kind:       %s\n", kind)
	fmt.Printf("  Name:       %s\n", name)
	fmt.Printf("  Namespace:  %s\n", namespace)
	fmt.Printf("  Repository: %s/%s\n", repoOwner, repoName)
	fmt.Println()
	fmt.Println("This will:")
	fmt.Println("  1. Remove k8s/claims/" + name + ".yaml from Git")
	fmt.Println("  2. Trigger Argo CD sync (removes Claim from cluster)")
	fmt.Println("  3. Delete all Azure resources (ResourceGroup, " + getAzureResourcesForKind(kind) + ")")
	fmt.Println()
	fmt.Println("⚠️  This action is IRREVERSIBLE. Data in Azure resources will be lost.")
	fmt.Println()
	fmt.Printf("Type the Claim name '%s' to confirm: ", name)

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	// Trim whitespace and compare
	input = strings.TrimSpace(input)
	if input != name {
		return false, nil
	}

	return true, nil
}

func displayDeleteSuccess(resp DeleteClaimResponse) {
	fmt.Println()
	fmt.Println("✓ Claim deleted successfully")
	fmt.Println()
	fmt.Printf("Kind:            %s\n", formatKindDisplay(resp.Kind))
	fmt.Printf("Name:            %s\n", resp.Name)
	fmt.Printf("Namespace:       %s\n", resp.Namespace)
	fmt.Printf("Repository:      %s\n", resp.RepoURL)
	fmt.Printf("Commit:          %s\n", resp.CommitSHA[:12]+"...")
	fmt.Printf("File Removed:    %s\n", resp.FilePath)
	fmt.Println()
	fmt.Println("Next Steps:")
	fmt.Println("  • Argo CD will sync within 60 seconds")
	fmt.Println("  • Crossplane will delete Azure resources (ResourceGroup, " + getAzureResourcesForKind(resp.Kind) + ")")
	fmt.Printf("  • Monitor progress: rdp infra status %s %s\n", resp.Kind, resp.Name)
	fmt.Println()
}

// Helper functions

func getAzureResourcesForKind(kind string) string {
	normalizedKind := strings.ToLower(kind)
	switch normalizedKind {
	case "storage", "storagebucket":
		return "StorageAccount, BlobContainer"
	case "vault", "keyvault":
		return "Key Vault"
	default:
		return "Azure resources"
	}
}

func formatKindDisplay(kind string) string {
	normalizedKind := strings.ToLower(kind)
	switch normalizedKind {
	case "storage":
		return "StorageBucket"
	case "vault":
		return "Vault"
	default:
		return kind
	}
}
