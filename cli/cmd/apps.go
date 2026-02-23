package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage Argo CD applications",
	Long: `Manage Argo CD applications deployed on the platform.

Commands:
  list     List all applications
  status   Show detailed status for a specific application
  sync     Trigger application sync`,
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	Long: `List all Argo CD applications with summary information.

Shows application name, project, sync status, health status, repository,
path, age, and last deployed timestamp.

Examples:
  rdp apps list              # List all applications
  rdp apps list -p platform  # List only platform project apps
  rdp apps list -j           # JSON output`,
	RunE: runAppsList,
	Args: cobra.NoArgs,
}

var appsStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status for an application",
	Long: `Show detailed status for a specific Argo CD application including:
  - Application metadata and project
  - Source repository information
  - Sync and health status
  - Managed resources with status
  - Recent deployment history
  - Current conditions and errors

Examples:
  rdp apps status platform-api
  rdp apps status argocd -j`,
	RunE: runAppsStatus,
	Args: cobra.ExactArgs(1),
}

var appsSyncCmd = &cobra.Command{
	Use:   "sync <name>",
	Short: "Trigger application sync",
	Long: `Trigger a sync operation for an Argo CD application.

The sync operation reconciles the live cluster state with the desired
state defined in Git. This is an asynchronous operation - use
'rdp apps status' to check progress.

Examples:
  rdp apps sync platform-api                    # Basic sync
  rdp apps sync platform-api --prune            # Sync with pruning
  rdp apps sync platform-api --dry-run          # Preview changes
  rdp apps sync platform-api --revision abc1234 # Sync specific commit`,
	RunE: runAppsSync,
	Args: cobra.ExactArgs(1),
}

// Flags
var (
	appsProject    string
	appsOutputJSON bool
	appsSyncPrune  bool
	appsSyncDryRun bool
	appsSyncRev    string
)

func init() {
	rootCmd.AddCommand(appsCmd)
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsStatusCmd)
	appsCmd.AddCommand(appsSyncCmd)

	// List flags
	appsListCmd.Flags().StringVarP(&appsProject, "project", "p", "", "Filter by Argo CD project")
	appsListCmd.Flags().BoolVarP(&appsOutputJSON, "json", "j", false, "Output in JSON format")

	// Status flags
	appsStatusCmd.Flags().BoolVarP(&appsOutputJSON, "json", "j", false, "Output in JSON format")

	// Sync flags
	appsSyncCmd.Flags().BoolVar(&appsSyncPrune, "prune", false, "Prune resources not in Git")
	appsSyncCmd.Flags().BoolVar(&appsSyncDryRun, "dry-run", false, "Preview sync without applying")
	appsSyncCmd.Flags().StringVar(&appsSyncRev, "revision", "", "Sync specific Git revision")
}

// ApplicationSummary matches the API response type (ApplicationSummaryResponse)
type ApplicationSummary struct {
	Name         string     `json:"name"`
	Namespace    string     `json:"namespace,omitempty"`
	Project      string     `json:"project"`
	SyncStatus   string     `json:"syncStatus"`
	HealthStatus string     `json:"healthStatus"`
	RepoURL      string     `json:"repoURL"`
	Path         string     `json:"path,omitempty"`
	Revision     string     `json:"revision,omitempty"`
	LastDeployed *time.Time `json:"lastDeployed,omitempty"`
}

// ListAppsResponse matches the API response
type ListAppsResponse struct {
	Applications []ApplicationSummary `json:"applications"`
	Total        int                  `json:"total"`
}

// Application represents full application details
type Application struct {
	Metadata ApplicationMetadata `json:"metadata"`
	Spec     ApplicationSpec     `json:"spec"`
	Status   ApplicationStatus   `json:"status"`
}

type ApplicationMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp,omitempty"`
}

type ApplicationSpec struct {
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
	Project     string                 `json:"project"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Chart          string `json:"chart,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server,omitempty"`
	Namespace string `json:"namespace"`
	Name      string `json:"name,omitempty"`
}

type SyncPolicy struct {
	Automated   *AutomatedSyncPolicy `json:"automated,omitempty"`
	SyncOptions []string             `json:"syncOptions,omitempty"`
}

type AutomatedSyncPolicy struct {
	Prune    bool `json:"prune,omitempty"`
	SelfHeal bool `json:"selfHeal,omitempty"`
}

type ApplicationStatus struct {
	Resources    []ResourceStatus       `json:"resources,omitempty"`
	Sync         SyncStatus             `json:"sync"`
	Health       HealthStatus           `json:"health"`
	History      []RevisionHistory      `json:"history,omitempty"`
	Conditions   []ApplicationCondition `json:"conditions,omitempty"`
	ReconciledAt *time.Time             `json:"reconciledAt,omitempty"`
}

type ResourceStatus struct {
	Group     string       `json:"group,omitempty"`
	Version   string       `json:"version,omitempty"`
	Kind      string       `json:"kind"`
	Namespace string       `json:"namespace,omitempty"`
	Name      string       `json:"name"`
	Status    string       `json:"status,omitempty"`
	Health    HealthStatus `json:"health,omitempty"`
}

type SyncStatus struct {
	Status     string     `json:"status"`
	ComparedTo ComparedTo `json:"comparedTo,omitempty"`
	Revision   string     `json:"revision,omitempty"`
}

type ComparedTo struct {
	Source ApplicationSource `json:"source"`
}

type HealthStatus struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type RevisionHistory struct {
	Revision   string    `json:"revision"`
	DeployedAt time.Time `json:"deployedAt"`
	ID         int64     `json:"id"`
}

type ApplicationCondition struct {
	Type               string     `json:"type"`
	Message            string     `json:"message"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
}

// SyncRequest for triggering sync operations
type SyncRequest struct {
	Revision string `json:"revision,omitempty"`
	Prune    bool   `json:"prune,omitempty"`
	DryRun   bool   `json:"dryRun,omitempty"`
}

// SyncResponse from sync operation
type SyncResponse struct {
	Message string `json:"message"`
	Phase   string `json:"phase,omitempty"`
}

func runAppsList(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()

	// Make API request
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL+"/api/v1/apps", nil)
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
	var listResp ListAppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter by project if specified
	if appsProject != "" {
		filtered := []ApplicationSummary{}
		for _, app := range listResp.Applications {
			if app.Project == appsProject {
				filtered = append(filtered, app)
			}
		}
		listResp.Applications = filtered
		listResp.Total = len(filtered)
	}

	// Output format
	if appsOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(listResp)
	}

	// Human-readable table format
	displayAppsTable(listResp.Applications)
	return nil
}

func displayAppsTable(apps []ApplicationSummary) {
	if len(apps) == 0 {
		fmt.Println("No applications found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tPROJECT\tSYNC\tHEALTH\tREPO\tPATH\tAGE\tLAST DEPLOYED")
	fmt.Fprintln(w, "----\t-------\t----\t------\t----\t----\t---\t-------------")

	// Rows
	for _, app := range apps {
		syncIcon := formatSyncIcon(app.SyncStatus, app.HealthStatus)
		syncStatus := formatSyncStatusShort(app.SyncStatus)
		healthStatus := formatHealthStatusShort(app.HealthStatus)

		// Truncate long repo URLs
		repoDisplay := truncateString(app.RepoURL, 40)

		// Format age and last deployed
		age := "-"
		lastDeployed := "-"
		if app.LastDeployed != nil {
			age = formatAge(*app.LastDeployed)
			lastDeployed = app.LastDeployed.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\t%s\t%s\t%s\t%s\n",
			app.Name,
			app.Project,
			syncIcon,
			syncStatus,
			healthStatus,
			repoDisplay,
			app.Path,
			age,
			lastDeployed,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d applications\n", len(apps))
}

func runAppsStatus(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	appName := args[0]

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/apps/%s", config.APIBaseURL, appName)

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

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("application '%s' not found", appName)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if appsOutputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(app)
	}

	// Human-readable format
	displayApplicationStatus(app)
	return nil
}

func displayApplicationStatus(app Application) {
	// Application header
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Application: %s\n", app.Metadata.Name)
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Application info
	fmt.Println("┌─ Application Info ────────────────────────────────────────┐")
	fmt.Printf("│ Name:        %s\n", app.Metadata.Name)
	if app.Metadata.Namespace != "" {
		fmt.Printf("│ Namespace:   %s\n", app.Metadata.Namespace)
	}
	fmt.Printf("│ Project:     %s\n", app.Spec.Project)
	if !app.Metadata.CreationTimestamp.IsZero() {
		fmt.Printf("│ Age:         %s\n", formatAge(app.Metadata.CreationTimestamp))
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Source info
	fmt.Println("┌─ Source ──────────────────────────────────────────────────┐")
	fmt.Printf("│ Repo:        %s\n", app.Spec.Source.RepoURL)
	if app.Spec.Source.Path != "" {
		fmt.Printf("│ Path:        %s\n", app.Spec.Source.Path)
	}
	if app.Spec.Source.Chart != "" {
		fmt.Printf("│ Chart:       %s\n", app.Spec.Source.Chart)
	}
	if app.Spec.Source.TargetRevision != "" {
		fmt.Printf("│ Target:      %s\n", app.Spec.Source.TargetRevision)
	}
	if app.Status.Sync.Revision != "" {
		fmt.Printf("│ Revision:    %s\n", truncateString(app.Status.Sync.Revision, 12))
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Sync status
	fmt.Println("┌─ Sync Status ─────────────────────────────────────────────┐")
	syncIcon := formatSyncIcon(app.Status.Sync.Status, app.Status.Health.Status)
	fmt.Printf("│ Status:      %s %s\n", syncIcon, formatSyncStatus(app.Status.Sync.Status))
	if app.Status.Sync.ComparedTo.Source.RepoURL != "" {
		fmt.Printf("│ Compared To: %s @ %s\n",
			truncateString(app.Status.Sync.ComparedTo.Source.RepoURL, 30),
			app.Status.Sync.ComparedTo.Source.TargetRevision,
		)
	}
	if app.Status.ReconciledAt != nil {
		fmt.Printf("│ Last Sync:   %s\n", app.Status.ReconciledAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Health status
	fmt.Println("┌─ Health Status ───────────────────────────────────────────┐")
	healthIcon := formatHealthIcon(app.Status.Health.Status)
	fmt.Printf("│ Status:      %s %s\n", healthIcon, formatHealthStatus(app.Status.Health.Status))
	if app.Status.Health.Message != "" {
		// Wrap long messages
		message := app.Status.Health.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}
		fmt.Printf("│ Message:     %s\n", message)
	}
	fmt.Println("└───────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Resources
	if len(app.Status.Resources) > 0 {
		fmt.Println("┌─ Resources ───────────────────────────────────────────────┐")
		fmt.Println("│ KIND          NAMESPACE    NAME            STATUS    HEALTH")
		fmt.Println("│ ----          ---------    ----            ------    ------")

		// Show first 10 resources (typical apps have fewer than this)
		maxResources := 10
		resourceCount := len(app.Status.Resources)
		if resourceCount > maxResources {
			resourceCount = maxResources
		}

		for i := 0; i < resourceCount; i++ {
			res := app.Status.Resources[i]
			statusIcon := formatSyncIcon(res.Status, res.Health.Status)
			healthText := formatHealthStatusShort(res.Health.Status)

			// Truncate long names
			kindDisplay := truncateString(res.Kind, 12)
			nsDisplay := truncateString(res.Namespace, 10)
			nameDisplay := truncateString(res.Name, 14)

			fmt.Printf("│ %-13s %-11s %-15s %s %-6s %s\n",
				kindDisplay,
				nsDisplay,
				nameDisplay,
				statusIcon,
				res.Status,
				healthText,
			)
		}

		if len(app.Status.Resources) > maxResources {
			remaining := len(app.Status.Resources) - maxResources
			fmt.Printf("│ ... and %d more resources\n", remaining)
		}

		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}

	// Recent history
	if len(app.Status.History) > 0 {
		fmt.Println("┌─ Recent History ──────────────────────────────────────────┐")

		// Show last 5 deployments
		maxHistory := 5
		startIdx := 0
		if len(app.Status.History) > maxHistory {
			startIdx = len(app.Status.History) - maxHistory
		}

		for i := startIdx; i < len(app.Status.History); i++ {
			hist := app.Status.History[i]
			age := formatAge(hist.DeployedAt)
			revisionShort := truncateString(hist.Revision, 12)

			if i > startIdx {
				fmt.Println("├───────────────────────────────────────────────────────────┤")
			}

			fmt.Printf("│ Revision: %s (%s ago)\n", revisionShort, age)
			fmt.Printf("│ Deployed: %s\n", hist.DeployedAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}

	// Conditions
	if len(app.Status.Conditions) > 0 {
		fmt.Println("┌─ Conditions ──────────────────────────────────────────────┐")

		for i, cond := range app.Status.Conditions {
			if i > 0 {
				fmt.Println("├───────────────────────────────────────────────────────────┤")
			}

			condIcon := "⚠"
			fmt.Printf("│ %s %s\n", condIcon, cond.Type)

			// Wrap long messages
			message := cond.Message
			if len(message) > 55 {
				message = message[:52] + "..."
			}
			fmt.Printf("│   %s\n", message)
		}

		fmt.Println("└───────────────────────────────────────────────────────────┘")
		fmt.Println()
	}
}

func runAppsSync(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	config := GetConfig()
	appName := args[0]

	// Build sync request
	syncReq := SyncRequest{
		Prune:  appsSyncPrune,
		DryRun: appsSyncDryRun,
	}
	if appsSyncRev != "" {
		syncReq.Revision = appsSyncRev
	}

	// Marshal request body
	reqBody, err := json.Marshal(syncReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/apps/%s/sync", config.APIBaseURL, appName)

	// Make API request
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
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

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("application '%s' not found", appName)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Display result
	fmt.Printf("✓ Sync initiated for application '%s'\n\n", appName)
	fmt.Printf("Operation:    Sync\n")
	if syncResp.Phase != "" {
		fmt.Printf("Phase:        %s\n", syncResp.Phase)
	}
	fmt.Printf("Prune:        %v\n", appsSyncPrune)
	fmt.Printf("Dry Run:      %v\n", appsSyncDryRun)
	if appsSyncRev != "" {
		fmt.Printf("Revision:     %s\n", appsSyncRev)
	}
	fmt.Println()
	fmt.Printf("Use 'rdp apps status %s' to check progress.\n", appName)

	return nil
}

// Helper functions

func formatSyncIcon(syncStatus string, healthStatus string) string {
	if syncStatus == "Synced" && (healthStatus == "Healthy" || healthStatus == "Suspended") {
		return "✓"
	} else if syncStatus == "OutOfSync" || healthStatus == "Progressing" {
		return "⚠"
	}
	return "✗"
}

func formatSyncStatus(syncStatus string) string {
	switch syncStatus {
	case "Synced":
		return "✓ Synced"
	case "OutOfSync":
		return "⚠ OutOfSync"
	case "Unknown":
		return "○ Unknown"
	default:
		return "○ " + syncStatus
	}
}

func formatSyncStatusShort(syncStatus string) string {
	switch syncStatus {
	case "Synced":
		return "Synced"
	case "OutOfSync":
		return "OutOfSync"
	default:
		return syncStatus
	}
}

func formatHealthIcon(healthStatus string) string {
	switch healthStatus {
	case "Healthy":
		return "✓"
	case "Progressing":
		return "⚠"
	case "Degraded", "Missing":
		return "✗"
	case "Suspended":
		return "○"
	default:
		return "○"
	}
}

func formatHealthStatus(healthStatus string) string {
	switch healthStatus {
	case "Healthy":
		return "✓ Healthy"
	case "Progressing":
		return "⚠ Progressing"
	case "Degraded":
		return "✗ Degraded"
	case "Suspended":
		return "○ Suspended"
	case "Missing":
		return "✗ Missing"
	case "Unknown":
		return "○ Unknown"
	default:
		return "○ " + healthStatus
	}
}

func formatHealthStatusShort(healthStatus string) string {
	switch healthStatus {
	case "Healthy":
		return "Healthy"
	case "Progressing":
		return "Progressing"
	case "Degraded":
		return "Degraded"
	case "Suspended":
		return "Suspended"
	case "Missing":
		return "Missing"
	default:
		return healthStatus
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
