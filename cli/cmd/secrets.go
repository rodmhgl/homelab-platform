package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage and view secrets (ExternalSecrets and Crossplane connection secrets)",
	Long: `Manage and view secrets in the platform.

Provides unified visibility into both ExternalSecrets (ESO-managed) and
Crossplane connection secrets. Shows sync status, available keys (not values),
and source Claims for connection secrets.

Commands:
  list    List all secrets in a namespace`,
}

func init() {
	rootCmd.AddCommand(secretsCmd)
}

// SecretsListResponse matches api/internal/secrets/types.go ListSecretsResponse
type SecretsListResponse struct {
	Secrets []SecretSummary `json:"secrets"`
	Total   int             `json:"total"`
}

// SecretSummary matches api/internal/secrets/types.go SecretSummary
type SecretSummary struct {
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	Kind              string                 `json:"kind"` // "ExternalSecret" or "Secret"
	Type              string                 `json:"type,omitempty"`
	Status            string                 `json:"status,omitempty"` // "Ready", "Error", "Synced", "Unknown"
	Message           string                 `json:"message,omitempty"`
	CreationTimestamp time.Time              `json:"creationTimestamp"`
	Labels            map[string]string      `json:"labels,omitempty"`
	Keys              []string               `json:"keys,omitempty"`
	SourceClaim       *SecretResourceRef     `json:"sourceClaim,omitempty"`
}

// SecretResourceRef matches api/internal/secrets/types.go ResourceRef
type SecretResourceRef struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // "StorageBucket" or "Vault"
}

// formatSecretStatus returns color-coded status with icon
func formatSecretStatus(status string) string {
	switch status {
	case "Ready", "Synced":
		return fmt.Sprintf("\033[32m✓ %s\033[0m", status) // Green
	case "Error":
		return fmt.Sprintf("\033[31m✗ %s\033[0m", status) // Red
	case "Unknown", "":
		return "\033[90m○ Unknown\033[0m" // Gray
	default:
		return fmt.Sprintf("○ %s", status)
	}
}

// formatSecretKind returns display-friendly kind name
func formatSecretKind(kind string) string {
	// Both "ExternalSecret" and "Secret" are already display-friendly
	return kind
}

// formatKeys returns formatted key display (list for ≤3 keys, count for >3)
func formatKeys(keys []string) string {
	if len(keys) == 0 {
		return "-"
	}
	if len(keys) <= 3 {
		return strings.Join(keys, ", ")
	}
	return fmt.Sprintf("%d keys", len(keys))
}

// formatSourceClaim returns formatted source claim reference or "-"
func formatSourceClaim(sourceClaim *SecretResourceRef) string {
	if sourceClaim == nil {
		return "-"
	}
	return fmt.Sprintf("%s (%s)", sourceClaim.Name, sourceClaim.Kind)
}
