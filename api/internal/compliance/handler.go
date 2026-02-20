package compliance

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Handler handles compliance API requests
type Handler struct {
	client     *Client
	eventStore EventStore
}

// NewHandler creates a new compliance handler
func NewHandler(cfg *Config, eventStore EventStore) (*Handler, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Handler{
		client:     client,
		eventStore: eventStore,
	}, nil
}

// HandleSummary handles GET /api/v1/compliance/summary
// Returns overall compliance score and metrics
func (h *Handler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Generating compliance summary")

	// Query all constraints to count violations
	constraintLists, err := h.client.ListAllConstraints(ctx)
	if err != nil {
		slog.Error("Failed to list constraints", "error", err)
		// Continue with empty results (graceful degradation)
	}

	totalViolations := 0
	violationsBySeverity := map[string]int{
		"policy":   0,
		"config":   0,
		"security": 0,
	}

	// Count violations from all constraints
	for kind, list := range constraintLists {
		if list == nil {
			continue
		}

		for _, constraint := range list.Items {
			violations, found, err := getViolationsFromConstraint(&constraint)
			if err != nil || !found {
				continue
			}

			count := len(violations)
			totalViolations += count

			// Categorize by severity based on constraint kind
			severity := categorizeSeverity(kind)
			violationsBySeverity[severity] += count
		}
	}

	// Query vulnerability reports
	vulnReports, err := h.client.ListVulnerabilityReportsInWorkloads(ctx)
	if err != nil {
		slog.Error("Failed to list vulnerability reports", "error", err)
		// Continue with empty results
		vulnReports = nil
	}

	totalVulnerabilities := 0
	vulnerabilitiesBySeverity := map[string]int{
		"CRITICAL": 0,
		"HIGH":     0,
		"MEDIUM":   0,
		"LOW":      0,
		"UNKNOWN":  0,
	}

	if vulnReports != nil {
		for _, report := range vulnReports.Items {
			vulns, found, err := getVulnerabilitiesFromReport(&report)
			if err != nil || !found {
				continue
			}

			for _, vuln := range vulns {
				severity, _ := vuln["severity"].(string)
				if severity == "" {
					severity = "UNKNOWN"
				}
				vulnerabilitiesBySeverity[severity]++
				totalVulnerabilities++
			}
		}
	}

	// Query recent Falco events (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	recentEvents := h.eventStore.List(EventFilters{Since: since})

	// Count Falco events by severity
	criticalEvents := 0
	errorEvents := 0
	for _, event := range recentEvents {
		switch event.Severity {
		case "Critical", "Alert", "Emergency":
			criticalEvents++
		case "Error":
			errorEvents++
		}
	}

	// Calculate compliance score
	// Formula: max(0, 100 - (violations * 5) - (critical_cves * 10) - (high_cves * 5) - (critical_events * 15) - (error_events * 8))
	// Rationale: Critical runtime events (15×) weighted heavier than critical CVEs (10×)
	// because they indicate active threats vs potential vulnerabilities
	score := 100.0
	score -= float64(totalViolations) * 5.0
	score -= float64(vulnerabilitiesBySeverity["CRITICAL"]) * 10.0
	score -= float64(vulnerabilitiesBySeverity["HIGH"]) * 5.0
	score -= float64(criticalEvents) * 15.0
	score -= float64(errorEvents) * 8.0
	score = math.Max(0, score)

	response := SummaryResponse{
		ComplianceScore:           score,
		TotalViolations:           totalViolations,
		TotalVulnerabilities:      totalVulnerabilities,
		ViolationsBySeverity:      violationsBySeverity,
		VulnerabilitiesBySeverity: vulnerabilitiesBySeverity,
	}

	slog.Info("Compliance summary generated",
		"score", score,
		"violations", totalViolations,
		"vulnerabilities", totalVulnerabilities,
		"critical_events", criticalEvents,
		"error_events", errorEvents,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandlePolicies handles GET /api/v1/compliance/policies
// Returns list of active Gatekeeper policies
func (h *Handler) HandlePolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Listing compliance policies")

	templates, err := h.client.ListConstraintTemplates(ctx)
	if err != nil {
		slog.Error("Failed to list constraint templates", "error", err)
		http.Error(w, `{"error":"failed to list constraint templates"}`, http.StatusInternalServerError)
		return
	}

	policies := make([]Policy, 0, len(templates.Items))

	for _, template := range templates.Items {
		name := template.GetName()

		// Extract kind from spec.crd.spec.names.kind
		kind, _, _ := getNestedString(template.Object, "spec", "crd", "spec", "names", "kind")

		// Extract description from annotations or metadata
		description := template.GetAnnotations()["description"]
		if description == "" {
			description = fmt.Sprintf("Validates %s policy", name)
		}

		// Determine scope (cluster-wide for all our constraints)
		scope := []string{"cluster"}

		// Extract parameters from spec.crd.spec.validation.openAPIV3Schema.properties.parameters
		parameters := make(map[string]interface{})
		params, found, _ := getNestedMap(template.Object, "spec", "crd", "spec", "validation", "openAPIV3Schema", "properties", "parameters", "properties")
		if found {
			parameters = params
		}

		policy := Policy{
			Name:        name,
			Kind:        kind,
			Description: description,
			Scope:       scope,
			Parameters:  parameters,
		}

		policies = append(policies, policy)
	}

	response := PoliciesResponse{
		Policies: policies,
	}

	slog.Info("Policies listed", "count", len(policies))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleViolations handles GET /api/v1/compliance/violations
// Returns Gatekeeper audit violations with optional filtering
func (h *Handler) HandleViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	namespace := r.URL.Query().Get("namespace")
	kind := r.URL.Query().Get("kind")
	constraint := r.URL.Query().Get("constraint")

	slog.Info("Listing violations",
		"namespace", namespace,
		"kind", kind,
		"constraint", constraint,
	)

	// Query all constraints
	constraintLists, err := h.client.ListAllConstraints(ctx)
	if err != nil {
		slog.Error("Failed to list constraints", "error", err)
		http.Error(w, `{"error":"failed to list constraints"}`, http.StatusInternalServerError)
		return
	}

	violations := []Violation{}

	// Extract violations from all constraints
	for constraintKind, list := range constraintLists {
		if list == nil {
			continue
		}

		for _, constraintObj := range list.Items {
			constraintName := constraintObj.GetName()

			// Apply constraint filter
			if constraint != "" && constraintName != constraint {
				continue
			}

			violationList, found, err := getViolationsFromConstraint(&constraintObj)
			if err != nil || !found {
				continue
			}

			for _, v := range violationList {
				// Extract violation details
				message, _ := v["message"].(string)

				// Extract enforcement action
				enforcementAction, _ := v["enforcementAction"].(string)

				// Extract resource kind and name
				resourceKind := ""
				resourceName := ""
				resourceNamespace := ""

				if kindVal, ok := v["kind"].(string); ok {
					resourceKind = kindVal
				}
				if nameVal, ok := v["name"].(string); ok {
					resourceName = nameVal
				}
				if nsVal, ok := v["namespace"].(string); ok {
					resourceNamespace = nsVal
				}

				// Apply filters
				if namespace != "" && resourceNamespace != namespace {
					continue
				}
				if kind != "" && resourceKind != kind {
					continue
				}

				// Build resource identifier
				resource := fmt.Sprintf("%s/%s/%s", resourceNamespace, resourceKind, resourceName)
				if resourceNamespace == "" {
					resource = fmt.Sprintf("%s/%s", resourceKind, resourceName)
				}

				violation := Violation{
					ConstraintName: constraintName,
					ConstraintKind: constraintKind,
					Resource:       resource,
					Namespace:      resourceNamespace,
					Message:        message,
				}

				// Add enforcement action to message if present
				if enforcementAction != "" && enforcementAction != "deny" {
					violation.Message = fmt.Sprintf("[%s] %s", enforcementAction, message)
				}

				violations = append(violations, violation)
			}
		}
	}

	response := ViolationsResponse{
		Violations: violations,
	}

	slog.Info("Violations listed", "count", len(violations))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleVulnerabilities handles GET /api/v1/compliance/vulnerabilities
// Returns Trivy CVE scan results with optional filtering
func (h *Handler) HandleVulnerabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	namespace := r.URL.Query().Get("namespace")
	severity := strings.ToUpper(r.URL.Query().Get("severity"))
	image := r.URL.Query().Get("image")

	slog.Info("Listing vulnerabilities",
		"namespace", namespace,
		"severity", severity,
		"image", image,
	)

	// Query vulnerability reports
	var reports *unstructured.UnstructuredList
	var err error

	if namespace != "" {
		reports, err = h.client.ListVulnerabilityReports(ctx, namespace)
	} else {
		reports, err = h.client.ListVulnerabilityReportsInWorkloads(ctx)
	}

	if err != nil {
		slog.Error("Failed to list vulnerability reports", "error", err)
		http.Error(w, `{"error":"failed to list vulnerability reports"}`, http.StatusInternalServerError)
		return
	}

	vulnerabilities := []Vulnerability{}

	// Extract vulnerabilities from reports
	for _, report := range reports.Items {
		reportNamespace := report.GetNamespace()
		reportName := report.GetName()

		// Extract image info
		imageRepo, _, _ := getNestedString(report.Object, "report", "artifact", "repository")
		imageTag, _, _ := getNestedString(report.Object, "report", "artifact", "tag")
		imageFull := fmt.Sprintf("%s:%s", imageRepo, imageTag)

		// Apply image filter
		if image != "" && !strings.Contains(imageFull, image) {
			continue
		}

		// Extract workload info (from report name: typically "replicaset-<name>-<hash>")
		workload := reportName

		// Extract vulnerabilities array
		vulns, found, err := getVulnerabilitiesFromReport(&report)
		if err != nil || !found {
			continue
		}

		for _, v := range vulns {
			vulnSeverity, _ := v["severity"].(string)

			// Apply severity filter
			if severity != "" && vulnSeverity != severity {
				continue
			}

			vulnID, _ := v["vulnerabilityID"].(string)
			pkg, _ := v["resource"].(string)
			fixedVersion, _ := v["fixedVersion"].(string)
			primaryLink, _ := v["primaryLink"].(string)

			// Extract score
			score := 0.0
			if scoreMap, ok := v["score"].(map[string]interface{}); ok {
				if v3Score, ok := scoreMap["nvd"].(map[string]interface{}); ok {
					if v3Val, ok := v3Score["V3Score"].(float64); ok {
						score = v3Val
					}
				}
			}

			vulnerability := Vulnerability{
				Image:           imageFull,
				Namespace:       reportNamespace,
				Workload:        workload,
				CVEID:           vulnID,
				Severity:        vulnSeverity,
				Score:           score,
				AffectedPackage: pkg,
				FixedVersion:    fixedVersion,
				PrimaryLink:     primaryLink,
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	response := VulnerabilitiesResponse{
		Vulnerabilities: vulnerabilities,
	}

	slog.Info("Vulnerabilities listed", "count", len(vulnerabilities))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleEvents handles GET /api/v1/compliance/events
// Returns security events from Falco with optional filtering
func (h *Handler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	namespace := r.URL.Query().Get("namespace")
	severity := r.URL.Query().Get("severity")
	rule := r.URL.Query().Get("rule")
	sinceStr := r.URL.Query().Get("since")

	filters := EventFilters{
		Namespace: namespace,
		Severity:  severity,
		Rule:      rule,
		Limit:     100, // Default limit to prevent excessive response sizes
	}

	// Parse since parameter (RFC3339 timestamp)
	if sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			filters.Since = since
		} else {
			slog.Warn("Invalid since parameter, ignoring", "since", sinceStr, "error", err)
		}
	}

	// Query event store
	events := h.eventStore.List(filters)

	slog.Info("Listed security events",
		"count", len(events),
		"namespace", namespace,
		"severity", severity,
		"rule", rule,
	)

	response := EventsResponse{Events: events}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// Helper functions

// getViolationsFromConstraint extracts violations array from a Constraint's status
func getViolationsFromConstraint(constraint *unstructured.Unstructured) ([]map[string]interface{}, bool, error) {
	violations, found, err := unstructured.NestedSlice(constraint.Object, "status", "violations")
	if err != nil || !found {
		return nil, false, err
	}

	result := make([]map[string]interface{}, 0, len(violations))
	for _, v := range violations {
		if vMap, ok := v.(map[string]interface{}); ok {
			result = append(result, vMap)
		}
	}

	return result, true, nil
}

// getVulnerabilitiesFromReport extracts vulnerabilities array from a VulnerabilityReport
func getVulnerabilitiesFromReport(report *unstructured.Unstructured) ([]map[string]interface{}, bool, error) {
	vulns, found, err := unstructured.NestedSlice(report.Object, "report", "vulnerabilities")
	if err != nil || !found {
		return nil, false, err
	}

	result := make([]map[string]interface{}, 0, len(vulns))
	for _, v := range vulns {
		if vMap, ok := v.(map[string]interface{}); ok {
			result = append(result, vMap)
		}
	}

	return result, true, nil
}

// getNestedString safely extracts a string from nested object
func getNestedString(obj map[string]interface{}, fields ...string) (string, bool, error) {
	val, found, err := unstructured.NestedString(obj, fields...)
	return val, found, err
}

// getNestedMap safely extracts a map from nested object
func getNestedMap(obj map[string]interface{}, fields ...string) (map[string]interface{}, bool, error) {
	val, found, err := unstructured.NestedMap(obj, fields...)
	return val, found, err
}

// categorizeSeverity maps constraint kind to severity category
func categorizeSeverity(kind string) string {
	switch kind {
	case "NoPrivilegedContainers", "AllowedRepos", "CrossplaneNoPublicAccess":
		return "security"
	case "K8sRequiredLabels", "NoLatestTag", "CrossplaneClaimLocation":
		return "policy"
	case "ContainerLimitsRequired", "RequireProbes":
		return "config"
	default:
		return "policy"
	}
}
