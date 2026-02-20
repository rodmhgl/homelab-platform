package argocd

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler handles Argo CD API requests
type Handler struct {
	client *Client
}

// Config holds the configuration for the Argo CD handler
type Config struct {
	ServerURL string
	Token     string
}

// NewHandler creates a new Argo CD handler
func NewHandler(cfg *Config) *Handler {
	return &Handler{
		client: NewClient(cfg.ServerURL, cfg.Token),
	}
}

// HandleListApps handles GET /api/v1/apps
// Lists all applications managed by Argo CD
func (h *Handler) HandleListApps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("Listing Argo CD applications")

	appList, err := h.client.ListApplications(ctx)
	if err != nil {
		slog.Error("Failed to list applications", "error", err)
		http.Error(w, `{"error":"failed to list applications"}`, http.StatusInternalServerError)
		return
	}

	// Transform to simplified response format
	apps := make([]ApplicationSummaryResponse, 0, len(appList.Items))
	for _, app := range appList.Items {
		summary := ApplicationSummaryResponse{
			Name:         app.Metadata.Name,
			Namespace:    app.Metadata.Namespace,
			Project:      app.Spec.Project,
			SyncStatus:   app.Status.Sync.Status,
			HealthStatus: app.Status.Health.Status,
			RepoURL:      app.Spec.Source.RepoURL,
			Path:         app.Spec.Source.Path,
			Revision:     app.Status.Sync.Revision,
		}

		// Get last deployed time from operation state or history
		if app.Status.OperationState != nil && app.Status.OperationState.FinishedAt != nil {
			summary.LastDeployed = app.Status.OperationState.FinishedAt
		} else if len(app.Status.History) > 0 {
			// Use most recent history entry
			lastHistory := app.Status.History[len(app.Status.History)-1]
			summary.LastDeployed = &lastHistory.DeployedAt
		}

		apps = append(apps, summary)
	}

	response := ListAppsResponse{
		Applications: apps,
		Total:        len(apps),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleGetApp handles GET /api/v1/apps/{name}
// Retrieves a specific application by name
func (h *Handler) HandleGetApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if name == "" {
		http.Error(w, `{"error":"application name is required"}`, http.StatusBadRequest)
		return
	}

	slog.Info("Getting Argo CD application", "name", name)

	app, err := h.client.GetApplication(ctx, name)
	if err != nil {
		slog.Error("Failed to get application", "name", name, "error", err)
		if err.Error() == "application "+name+" not found" {
			http.Error(w, `{"error":"application not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to get application"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(app); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// HandleSyncApp handles POST /api/v1/apps/{name}/sync
// Triggers a sync for a specific application
func (h *Handler) HandleSyncApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if name == "" {
		http.Error(w, `{"error":"application name is required"}`, http.StatusBadRequest)
		return
	}

	slog.Info("Syncing Argo CD application", "name", name)

	// Parse optional sync request body
	var syncReq SyncRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
			// If decoding fails, use default sync (empty request is valid)
			slog.Debug("Using default sync options", "name", name)
		}
	}

	app, err := h.client.SyncApplication(ctx, name, &syncReq)
	if err != nil {
		slog.Error("Failed to sync application", "name", name, "error", err)
		http.Error(w, `{"error":"failed to sync application"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(app); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
