package secrets

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the secrets endpoints.
type Handler struct {
	client *Client
}

// NewHandler creates a new secrets handler.
func NewHandler(cfg *Config) (*Handler, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Handler{
		client: client,
	}, nil
}

// HandleListSecrets handles GET /api/v1/secrets/{namespace}.
func (h *Handler) HandleListSecrets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namespace := chi.URLParam(r, "namespace")

	slog.Info("Getting secrets", "namespace", namespace)

	// Validation
	if namespace == "" {
		slog.Warn("Missing namespace parameter")
		http.Error(w, `{"error":"namespace parameter required"}`, http.StatusBadRequest)
		return
	}

	// Query secrets (includes graceful ExternalSecret handling)
	secrets, err := h.client.ListSecrets(ctx, namespace)
	if err != nil {
		slog.Error("Failed to list secrets", "namespace", namespace, "error", err)
		http.Error(w, `{"error":"failed to list secrets"}`, http.StatusInternalServerError)
		return
	}

	response := ListSecretsResponse{
		Secrets: secrets,
		Total:   len(secrets),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("Listed secrets", "namespace", namespace, "total", len(secrets))
}
