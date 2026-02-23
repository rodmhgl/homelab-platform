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

var complianceEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "View Falco security events",
	Long: `View runtime security events detected by Falco.

Shows timestamp, severity, rule name, resource, and event message.
Supports filtering by namespace, severity, and time window.

Examples:
  rdp compliance events
  rdp compliance events --namespace platform
  rdp compliance events --severity ERROR
  rdp compliance events --since 1h --limit 20
  rdp compliance events --json`,
	RunE: runComplianceEvents,
	Args: cobra.NoArgs,
}

var (
	eventsOutputJSON   bool
	eventsNamespace    string
	eventsSeverity     string
	eventsSince        string
	eventsLimit        int
)

func init() {
	complianceCmd.AddCommand(complianceEventsCmd)
	complianceEventsCmd.Flags().BoolVarP(&eventsOutputJSON, "json", "j", false, "Output in JSON format")
	complianceEventsCmd.Flags().StringVarP(&eventsNamespace, "namespace", "n", "", "Filter by namespace")
	complianceEventsCmd.Flags().StringVar(&eventsSeverity, "severity", "", "Filter by severity (ERROR|WARNING|NOTICE)")
	complianceEventsCmd.Flags().StringVar(&eventsSince, "since", "", "Time window (e.g., 1h, 30m, 24h)")
	complianceEventsCmd.Flags().IntVar(&eventsLimit, "limit", 50, "Maximum events to display")
}

// EventsResponse matches API response
type EventsResponse struct {
	Events []SecurityEvent `json:"events"`
}

// SecurityEvent matches API type
type SecurityEvent struct {
	Timestamp string `json:"timestamp"`
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Resource  string `json:"resource,omitempty"`
}

func runComplianceEvents(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := ValidateConfig(); err != nil {
		return err
	}

	// Validate severity filter
	if eventsSeverity != "" {
		severityUpper := strings.ToUpper(eventsSeverity)
		validSeverities := []string{"ERROR", "WARNING", "NOTICE"}
		valid := false
		for _, s := range validSeverities {
			if severityUpper == s {
				valid = true
				eventsSeverity = severityUpper
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid severity: %s (must be ERROR, WARNING, or NOTICE)", eventsSeverity)
		}
	}

	// Validate since duration
	if eventsSince != "" {
		if _, err := time.ParseDuration(eventsSince); err != nil {
			return fmt.Errorf("invalid duration format: %s (use format like 1h, 30m, 24h)", eventsSince)
		}
	}

	config := GetConfig()

	// Build request URL with query parameters
	params := url.Values{}
	if eventsNamespace != "" {
		params.Add("namespace", eventsNamespace)
	}
	if eventsSeverity != "" {
		params.Add("severity", eventsSeverity)
	}
	if eventsSince != "" {
		params.Add("since", eventsSince)
	}
	if eventsLimit > 0 {
		params.Add("limit", fmt.Sprintf("%d", eventsLimit))
	}

	reqURL := config.APIBaseURL + "/api/v1/compliance/events"
	if len(params) > 0 {
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
	var eventsResp EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&eventsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output format
	if eventsOutputJSON {
		return formatJSONOutput(eventsResp)
	}

	// Human-readable table format
	displayEventsTable(eventsResp.Events)
	return nil
}

func displayEventsTable(events []SecurityEvent) {
	if len(events) == 0 {
		fmt.Println("No security events found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "TIME\tSEVERITY\tRULE\tRESOURCE\tMESSAGE")
	fmt.Fprintln(w, "----\t--------\t----\t--------\t-------")

	// Rows
	for _, e := range events {
		// Color-code severity
		sevColor := severityColor(e.Severity)
		severityDisplay := fmt.Sprintf("%s%s%s", sevColor, e.Severity, colorReset)

		// Format timestamp
		timeDisplay := formatTimestamp(e.Timestamp)

		// Truncate long values
		rule := truncateString(e.Rule, 30)
		resource := e.Resource
		if resource == "" {
			resource = "-"
		} else {
			resource = truncateString(resource, 25)
		}
		message := truncateString(e.Message, 50)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			timeDisplay,
			severityDisplay,
			rule,
			resource,
			message,
		)
	}

	// Summary with severity breakdown
	errorCount := 0
	warningCount := 0
	noticeCount := 0
	for _, e := range events {
		switch e.Severity {
		case "ERROR":
			errorCount++
		case "WARNING":
			warningCount++
		case "NOTICE":
			noticeCount++
		}
	}

	fmt.Fprintf(w, "\nTotal: %d events", len(events))
	if errorCount > 0 || warningCount > 0 {
		fmt.Fprintf(w, " (%s%d Error%s, %s%d Warning%s, %d Notice)\n",
			colorRed, errorCount, colorReset,
			colorYellow, warningCount, colorReset,
			noticeCount)
	} else {
		fmt.Fprintf(w, " (%d Notice)\n", noticeCount)
	}
}
