package infra

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Handler handles infrastructure API requests
type Handler struct {
	client       *Client
	githubClient *GitHubClient
}

// NewHandler creates a new infrastructure handler
func NewHandler(cfg *Config, githubToken string) (*Handler, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	githubClient := NewGitHubClient(githubToken)

	return &Handler{
		client:       client,
		githubClient: githubClient,
	}, nil
}

// HandleGetResource handles GET /api/v1/infra/:kind/:name
// Returns the composed resource tree and events for a Crossplane Claim
func (h *Handler) HandleGetResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	kind := chi.URLParam(r, "kind")
	name := chi.URLParam(r, "name")

	// Extract namespace from query param (defaults to "default")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	slog.Info("Getting infrastructure resource",
		"kind", kind,
		"name", name,
		"namespace", namespace,
	)

	// Normalize kind (storagebucket -> StorageBucket, vault -> Vault)
	normalizedKind := normalizeKind(kind)

	// Get the Claim
	claimObj, err := h.client.GetClaim(ctx, namespace, normalizedKind, name)
	if err != nil {
		slog.Error("Failed to get claim", "error", err, "kind", normalizedKind, "name", name)
		http.Error(w, fmt.Sprintf(`{"error":"failed to get claim: %s"}`, err.Error()), http.StatusNotFound)
		return
	}

	// Parse Claim resource
	claim := parseClaimResource(claimObj)

	// Get Claim events
	claimEvents, err := h.client.GetEventsForResource(ctx, namespace, normalizedKind, name)
	if err != nil {
		slog.Warn("Failed to get claim events", "error", err)
		claimEvents = nil
	}

	// Initialize response
	response := GetResourceResponse{
		Claim:  claim,
		Events: parseEvents(claimEvents),
	}

	// Get Composite resource if referenced
	if claim.ResourceRef != nil {
		compositeObj, err := h.client.GetComposite(ctx, *claim.ResourceRef)
		if err != nil {
			slog.Warn("Failed to get composite", "error", err)
		} else {
			composite := parseCompositeResource(compositeObj)
			response.Composite = &composite

			// Get Composite events
			compositeEvents, err := h.client.GetEventsForResource(ctx, "", composite.Kind, composite.Name)
			if err != nil {
				slog.Warn("Failed to get composite events", "error", err)
			} else {
				response.Events = append(response.Events, parseEvents(compositeEvents)...)
			}

			// Get Managed resources
			for _, ref := range composite.ResourceRefs {
				managedObj, err := h.client.GetManagedResource(ctx, ref)
				if err != nil {
					slog.Warn("Failed to get managed resource",
						"error", err,
						"kind", ref.Kind,
						"name", ref.Name,
					)
					continue
				}

				managed := parseManagedResource(managedObj)
				response.Managed = append(response.Managed, managed)

				// Get Managed resource events
				managedEvents, err := h.client.GetEventsForResource(ctx, "", managed.Kind, managed.Name)
				if err != nil {
					slog.Warn("Failed to get managed resource events", "error", err)
				} else {
					response.Events = append(response.Events, parseEvents(managedEvents)...)
				}
			}
		}
	}

	slog.Info("Infrastructure resource retrieved",
		"kind", normalizedKind,
		"name", name,
		"status", claim.Status,
		"managed_count", len(response.Managed),
		"events_count", len(response.Events),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleListAllClaims handles GET /api/v1/infra
// Returns all Crossplane Claims (StorageBucket + Vault) across all namespaces
func (h *Handler) HandleListAllClaims(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Listing all infrastructure Claims")

	// Get all Claims
	claims, err := h.client.ListAllClaims(ctx)
	if err != nil {
		slog.Error("Failed to list all claims", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to list claims: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Convert to summary format
	summaries := make([]ClaimSummary, 0, len(claims))
	for _, claim := range claims {
		summaries = append(summaries, parseClaimSummary(&claim))
	}

	response := ListClaimsResponse{
		Claims: summaries,
		Total:  len(summaries),
	}

	slog.Info("All Claims listed successfully", "total", response.Total)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleListStorageClaims handles GET /api/v1/infra/storage
// Returns all StorageBucket Claims across all namespaces
func (h *Handler) HandleListStorageClaims(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Listing StorageBucket Claims")

	// Get StorageBucket Claims
	claimList, err := h.client.ListClaims(ctx, "StorageBucket")
	if err != nil {
		slog.Error("Failed to list StorageBucket claims", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to list storage claims: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Convert to summary format
	summaries := make([]ClaimSummary, 0, len(claimList.Items))
	for i := range claimList.Items {
		summaries = append(summaries, parseClaimSummary(&claimList.Items[i]))
	}

	response := ListClaimsResponse{
		Claims: summaries,
		Total:  len(summaries),
	}

	slog.Info("StorageBucket Claims listed successfully", "total", response.Total)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleListVaultClaims handles GET /api/v1/infra/vaults
// Returns all Vault Claims across all namespaces
func (h *Handler) HandleListVaultClaims(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Listing Vault Claims")

	// Get Vault Claims
	claimList, err := h.client.ListClaims(ctx, "Vault")
	if err != nil {
		slog.Error("Failed to list Vault claims", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to list vault claims: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Convert to summary format
	summaries := make([]ClaimSummary, 0, len(claimList.Items))
	for i := range claimList.Items {
		summaries = append(summaries, parseClaimSummary(&claimList.Items[i]))
	}

	response := ListClaimsResponse{
		Claims: summaries,
		Total:  len(summaries),
	}

	slog.Info("Vault Claims listed successfully", "total", response.Total)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// Helper functions

// normalizeKind normalizes URL kind parameter to proper Claim kind
func normalizeKind(kind string) string {
	switch strings.ToLower(kind) {
	case "storage", "storagebucket", "storagebuckets":
		return "StorageBucket"
	case "vault", "vaults", "keyvault", "keyvaults":
		return "Vault"
	default:
		// Capitalize first letter as fallback
		if len(kind) > 0 {
			return strings.ToUpper(kind[:1]) + strings.ToLower(kind[1:])
		}
		return kind
	}
}

// parseClaimResource converts unstructured Claim to ClaimResource
func parseClaimResource(obj *unstructured.Unstructured) ClaimResource {
	ready, synced := extractConditionStatus(obj)
	status := determineStatus(ready, synced)

	// Extract connection secret name
	connectionSecret, _, _ := unstructured.NestedString(obj.Object, "spec", "writeConnectionSecretToRef", "name")

	// Extract resource ref
	resourceRef, _ := extractResourceRef(obj)

	return ClaimResource{
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Kind:              obj.GetKind(),
		Labels:            obj.GetLabels(),
		Annotations:       obj.GetAnnotations(),
		Status:            status,
		Synced:            synced,
		Ready:             ready,
		ConnectionSecret:  connectionSecret,
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		ResourceRef:       resourceRef,
	}
}

// parseClaimSummary converts unstructured Claim to ClaimSummary (lightweight)
func parseClaimSummary(obj *unstructured.Unstructured) ClaimSummary {
	ready, synced := extractConditionStatus(obj)
	status := determineStatus(ready, synced)

	// Extract connection secret name
	connectionSecret, _, _ := unstructured.NestedString(obj.Object, "spec", "writeConnectionSecretToRef", "name")

	return ClaimSummary{
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Kind:              obj.GetKind(),
		Status:            status,
		Synced:            synced,
		Ready:             ready,
		ConnectionSecret:  connectionSecret,
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		Labels:            obj.GetLabels(),
	}
}

// parseCompositeResource converts unstructured Composite to CompositeResource
func parseCompositeResource(obj *unstructured.Unstructured) CompositeResource {
	ready, synced := extractConditionStatus(obj)
	status := determineStatus(ready, synced)

	// Extract resource refs
	resourceRefs := extractResourceRefs(obj)

	return CompositeResource{
		Name:              obj.GetName(),
		Kind:              obj.GetKind(),
		Labels:            obj.GetLabels(),
		Status:            status,
		Synced:            synced,
		Ready:             ready,
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		ResourceRefs:      resourceRefs,
	}
}

// parseManagedResource converts unstructured Managed Resource to ManagedResource
func parseManagedResource(obj *unstructured.Unstructured) ManagedResource {
	ready, synced := extractConditionStatus(obj)
	status := determineStatus(ready, synced)

	// Extract external name (Azure resource name)
	externalName, _, _ := unstructured.NestedString(obj.Object, "status", "atProvider", "name")
	if externalName == "" {
		// Fallback to metadata annotation
		externalName = obj.GetAnnotations()["crossplane.io/external-name"]
	}

	// Extract status message
	message := extractStatusMessage(obj)

	// Extract group from APIVersion
	apiVersion := obj.GetAPIVersion()
	group := ""
	if parts := strings.Split(apiVersion, "/"); len(parts) >= 1 {
		group = parts[0]
	}

	return ManagedResource{
		Name:              obj.GetName(),
		Kind:              obj.GetKind(),
		Group:             group,
		Labels:            obj.GetLabels(),
		Status:            status,
		Synced:            synced,
		Ready:             ready,
		ExternalName:      externalName,
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		Message:           message,
	}
}

// parseEvents converts Kubernetes events to KubernetesEvent slice
func parseEvents(events []corev1.Event) []KubernetesEvent {
	result := make([]KubernetesEvent, 0, len(events))

	for _, event := range events {
		// Build involved object identifier
		involvedObject := fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)
		if event.InvolvedObject.Namespace != "" {
			involvedObject = fmt.Sprintf("%s/%s", event.InvolvedObject.Namespace, involvedObject)
		}

		// Extract source
		source := event.Source.Component
		if event.Source.Host != "" {
			source = fmt.Sprintf("%s/%s", source, event.Source.Host)
		}

		// Handle timestamp fields (FirstTimestamp might be zero if using EventSeries)
		firstTimestamp := event.FirstTimestamp.Time
		lastTimestamp := event.LastTimestamp.Time

		if firstTimestamp.IsZero() && !event.EventTime.IsZero() {
			firstTimestamp = event.EventTime.Time
			lastTimestamp = event.EventTime.Time
		}

		result = append(result, KubernetesEvent{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			InvolvedObject: involvedObject,
			Source:         source,
			Count:          event.Count,
			FirstTimestamp: firstTimestamp,
			LastTimestamp:  lastTimestamp,
		})
	}

	// Sort events by last timestamp (most recent first)
	// Using a simple bubble sort for small event lists
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].LastTimestamp.After(result[i].LastTimestamp) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// HandleCreateClaim handles POST /api/v1/infra
// Creates a Crossplane Claim by committing YAML to the app's Git repository
func (h *Handler) HandleCreateClaim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse request
	var req CreateClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"invalid request body: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	slog.Info("Creating infrastructure Claim",
		"kind", req.Kind,
		"name", req.Name,
		"namespace", req.Namespace,
		"repo", fmt.Sprintf("%s/%s", req.RepoOwner, req.RepoName),
	)

	// 2. Validate request
	if err := validateCreateClaimRequest(&req); err != nil {
		slog.Error("Request validation failed", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"validation failed: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// 3. Validate against Gatekeeper constraints
	if err := validateAgainstGatekeeperConstraints(req.Kind, req.Parameters); err != nil {
		slog.Error("Gatekeeper constraint violation", "error", err, "kind", req.Kind)
		http.Error(w, fmt.Sprintf(`{"error":"policy violation: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// 4. Generate YAML
	yamlContent, err := h.generateClaimYAML(&req)
	if err != nil {
		slog.Error("Failed to generate YAML", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to generate claim YAML: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// 5. Commit to GitHub
	filePath := fmt.Sprintf("k8s/claims/%s.yaml", req.Name)
	commitMessage := buildCommitMessage(&req)

	commitSHA, err := h.githubClient.CommitClaim(ctx, req.RepoOwner, req.RepoName, filePath, yamlContent, commitMessage)
	if err != nil {
		slog.Error("Failed to commit to GitHub", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to commit to repository: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// 6. Build response
	response := CreateClaimResponse{
		Success:          true,
		Message:          "Claim committed successfully. Argo CD will sync it to the cluster.",
		Kind:             req.Kind,
		Name:             req.Name,
		Namespace:        req.Namespace,
		CommitSHA:        commitSHA,
		FilePath:         filePath,
		RepoURL:          fmt.Sprintf("https://github.com/%s/%s", req.RepoOwner, req.RepoName),
		ConnectionSecret: req.Name, // Connection secret name matches claim name
	}

	slog.Info("Claim committed successfully",
		"kind", req.Kind,
		"name", req.Name,
		"commit_sha", commitSHA,
		"file_path", filePath,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleDeleteClaim handles DELETE /api/v1/infra/:kind/:name
// Deletes a Crossplane Claim by removing YAML from the app's Git repository
func (h *Handler) HandleDeleteClaim(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract path parameters
	kind := chi.URLParam(r, "kind")
	name := chi.URLParam(r, "name")

	// Extract namespace from query param (defaults to "default")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Normalize kind
	normalizedKind := normalizeKind(kind)

	slog.Info("Deleting infrastructure Claim",
		"kind", normalizedKind,
		"name", name,
		"namespace", namespace,
	)

	// 2. Parse request body
	var req DeleteClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"invalid request body: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// 3. Validate required fields
	if req.RepoOwner == "" || req.RepoName == "" {
		slog.Error("Missing required fields", "repo_owner", req.RepoOwner, "repo_name", req.RepoName)
		http.Error(w, `{"error":"repoOwner and repoName are required"}`, http.StatusBadRequest)
		return
	}

	// 4. Verify the Claim exists in the cluster before deleting from Git
	// This prevents orphaned deletions if the Claim was already removed
	claimObj, err := h.client.GetClaim(ctx, namespace, normalizedKind, name)
	if err != nil {
		slog.Warn("Claim not found in cluster (may have been manually deleted)",
			"error", err,
			"kind", normalizedKind,
			"name", name,
		)
		// Continue with deletion from Git anyway - GitOps reconciliation will handle it
	} else {
		slog.Info("Claim exists in cluster",
			"kind", normalizedKind,
			"name", name,
			"resource_version", claimObj.GetResourceVersion(),
		)
	}

	// 5. Delete from GitHub
	filePath := fmt.Sprintf("k8s/claims/%s.yaml", name)
	commitMessage := fmt.Sprintf("chore(infra): delete %s Claim %s\n\nNamespace: %s\nRemoved via Platform API",
		normalizedKind,
		name,
		namespace,
	)

	commitSHA, err := h.githubClient.DeleteClaim(ctx, req.RepoOwner, req.RepoName, filePath, commitMessage)
	if err != nil {
		slog.Error("Failed to delete from GitHub", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to delete from repository: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// 6. Build response
	response := DeleteClaimResponse{
		Success:   true,
		Message:   "Claim deleted successfully from Git. Argo CD will remove it from the cluster.",
		Kind:      normalizedKind,
		Name:      name,
		Namespace: namespace,
		CommitSHA: commitSHA,
		FilePath:  filePath,
		RepoURL:   fmt.Sprintf("https://github.com/%s/%s", req.RepoOwner, req.RepoName),
	}

	slog.Info("Claim deleted successfully",
		"kind", normalizedKind,
		"name", name,
		"commit_sha", commitSHA,
		"file_path", filePath,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
