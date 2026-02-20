package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display platform health summary",
	Long: `Display a comprehensive health summary of the platform including:
  - API health and readiness
  - Compliance score and violation count
  - Application health status
  - Infrastructure resources (Claims)

This provides a quick overview of the platform's operational state.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// PlatformStatus holds the aggregated status from multiple API endpoints
type PlatformStatus struct {
	APIHealth      HealthStatus
	Compliance     ComplianceStatus
	Applications   ApplicationStatus
	Infrastructure InfraStatus
}

type HealthStatus struct {
	Healthy bool
	Ready   bool
	Error   string
}

type ComplianceStatus struct {
	Score       int
	Violations  int
	Policies    int
	CVEs        int
	Error       string
}

type ApplicationStatus struct {
	Total    int
	Healthy  int
	Degraded int
	Error    string
}

type InfraStatus struct {
	TotalClaims   int
	StorageClaims int
	VaultClaims   int
	Error         string
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Gather status from all endpoints
	status := PlatformStatus{}

	// Check API health
	status.APIHealth = checkAPIHealth(client, config.APIBaseURL, config.AuthToken)

	// Only proceed with other checks if API is reachable
	if status.APIHealth.Healthy {
		status.Compliance = getComplianceStatus(client, config.APIBaseURL, config.AuthToken)
		status.Applications = getApplicationStatus(client, config.APIBaseURL, config.AuthToken)
		status.Infrastructure = getInfraStatus(client, config.APIBaseURL, config.AuthToken)
	}

	// Display status summary
	displayStatus(status)

	// Return error if API is unhealthy
	if !status.APIHealth.Healthy {
		return fmt.Errorf("platform API is unhealthy")
	}

	return nil
}

func checkAPIHealth(client *http.Client, baseURL, token string) HealthStatus {
	status := HealthStatus{}

	// Check /health endpoint
	healthReq, err := http.NewRequest("GET", baseURL+"/health", nil)
	if err != nil {
		status.Error = fmt.Sprintf("failed to create request: %v", err)
		return status
	}
	healthReq.Header.Set("Authorization", "Bearer "+token)

	healthResp, err := client.Do(healthReq)
	if err != nil {
		status.Error = fmt.Sprintf("connection failed: %v", err)
		return status
	}
	defer healthResp.Body.Close()

	status.Healthy = (healthResp.StatusCode == http.StatusOK)

	// Check /ready endpoint
	readyReq, err := http.NewRequest("GET", baseURL+"/ready", nil)
	if err != nil {
		status.Error = fmt.Sprintf("failed to create ready request: %v", err)
		return status
	}
	readyReq.Header.Set("Authorization", "Bearer "+token)

	readyResp, err := client.Do(readyReq)
	if err != nil {
		status.Error = fmt.Sprintf("ready check failed: %v", err)
		return status
	}
	defer readyResp.Body.Close()

	status.Ready = (readyResp.StatusCode == http.StatusOK)

	return status
}

func getComplianceStatus(client *http.Client, baseURL, token string) ComplianceStatus {
	status := ComplianceStatus{}

	req, err := http.NewRequest("GET", baseURL+"/api/v1/compliance/summary", nil)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		status.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return status
	}

	var summary struct {
		ComplianceScore int `json:"complianceScore"`
		Violations      int `json:"violations"`
		Policies        int `json:"policies"`
		Vulnerabilities int `json:"vulnerabilities"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		status.Error = fmt.Sprintf("decode failed: %v", err)
		return status
	}

	status.Score = summary.ComplianceScore
	status.Violations = summary.Violations
	status.Policies = summary.Policies
	status.CVEs = summary.Vulnerabilities

	return status
}

func getApplicationStatus(client *http.Client, baseURL, token string) ApplicationStatus {
	status := ApplicationStatus{}

	req, err := http.NewRequest("GET", baseURL+"/api/v1/apps", nil)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		status.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return status
	}

	var apps struct {
		Apps []struct {
			Health string `json:"health"`
		} `json:"apps"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		status.Error = fmt.Sprintf("decode failed: %v", err)
		return status
	}

	status.Total = len(apps.Apps)
	for _, app := range apps.Apps {
		if app.Health == "Healthy" {
			status.Healthy++
		} else {
			status.Degraded++
		}
	}

	return status
}

func getInfraStatus(client *http.Client, baseURL, token string) InfraStatus {
	status := InfraStatus{}

	req, err := http.NewRequest("GET", baseURL+"/api/v1/infra", nil)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("request failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		status.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return status
	}

	var infra struct {
		Claims []struct {
			Kind string `json:"kind"`
		} `json:"claims"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&infra); err != nil {
		status.Error = fmt.Sprintf("decode failed: %v", err)
		return status
	}

	status.TotalClaims = len(infra.Claims)
	for _, claim := range infra.Claims {
		if claim.Kind == "StorageBucket" {
			status.StorageClaims++
		} else if claim.Kind == "Vault" {
			status.VaultClaims++
		}
	}

	return status
}

func displayStatus(status PlatformStatus) {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         RNLabs Developer Platform Status                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// API Health
	fmt.Println("┌─ Platform API ────────────────────────────────────────────┐")
	if status.APIHealth.Error != "" {
		fmt.Printf("│ Status:      %s\n", formatError("UNREACHABLE"))
		fmt.Printf("│ Error:       %s\n", status.APIHealth.Error)
	} else {
		healthIcon := formatHealthIcon(status.APIHealth.Healthy)
		readyIcon := formatHealthIcon(status.APIHealth.Ready)
		fmt.Printf("│ Health:      %s %s\n", healthIcon, formatHealthText(status.APIHealth.Healthy))
		fmt.Printf("│ Ready:       %s %s\n", readyIcon, formatHealthText(status.APIHealth.Ready))
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Compliance
	fmt.Println("┌─ Compliance ──────────────────────────────────────────────┐")
	if status.Compliance.Error != "" {
		fmt.Printf("│ Status:      %s\n", formatError("ERROR"))
		fmt.Printf("│ Error:       %s\n", status.Compliance.Error)
	} else {
		scoreIcon := formatComplianceIcon(status.Compliance.Score)
		fmt.Printf("│ Score:       %s %d/100\n", scoreIcon, status.Compliance.Score)
		fmt.Printf("│ Policies:    %d active\n", status.Compliance.Policies)
		fmt.Printf("│ Violations:  %d\n", status.Compliance.Violations)
		fmt.Printf("│ CVEs:        %d\n", status.Compliance.CVEs)
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Applications
	fmt.Println("┌─ Applications ────────────────────────────────────────────┐")
	if status.Applications.Error != "" {
		fmt.Printf("│ Status:      %s\n", formatError("ERROR"))
		fmt.Printf("│ Error:       %s\n", status.Applications.Error)
	} else {
		fmt.Printf("│ Total:       %d\n", status.Applications.Total)
		fmt.Printf("│ Healthy:     %s %d\n", formatHealthIcon(true), status.Applications.Healthy)
		if status.Applications.Degraded > 0 {
			fmt.Printf("│ Degraded:    %s %d\n", formatHealthIcon(false), status.Applications.Degraded)
		}
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Infrastructure
	fmt.Println("┌─ Infrastructure ──────────────────────────────────────────┐")
	if status.Infrastructure.Error != "" {
		fmt.Printf("│ Status:      %s\n", formatError("ERROR"))
		fmt.Printf("│ Error:       %s\n", status.Infrastructure.Error)
	} else {
		fmt.Printf("│ Total Claims: %d\n", status.Infrastructure.TotalClaims)
		fmt.Printf("│   Storage:    %d\n", status.Infrastructure.StorageClaims)
		fmt.Printf("│   Vaults:     %d\n", status.Infrastructure.VaultClaims)
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Overall status indicator
	overallHealthy := status.APIHealth.Healthy && status.APIHealth.Ready
	if overallHealthy {
		fmt.Println("Overall Status: ✓ Platform is operational")
	} else {
		fmt.Println("Overall Status: ✗ Platform has issues")
	}
	fmt.Println()
}

func formatHealthIcon(healthy bool) string {
	if healthy {
		return "✓"
	}
	return "✗"
}

func formatHealthText(healthy bool) string {
	if healthy {
		return "OK"
	}
	return "FAILED"
}

func formatComplianceIcon(score int) string {
	if score >= 90 {
		return "✓"
	} else if score >= 70 {
		return "⚠"
	}
	return "✗"
}

func formatError(text string) string {
	return fmt.Sprintf("✗ %s", text)
}
