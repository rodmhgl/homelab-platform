package webhooks

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rodmhgl/homelab-platform/api/internal/compliance"
)

// Handler handles webhook requests from external systems
type Handler struct {
	eventStore compliance.EventStore
}

// NewHandler creates a new webhook handler
func NewHandler(store compliance.EventStore) *Handler {
	return &Handler{
		eventStore: store,
	}
}

// HandleFalcoWebhook processes POST /api/v1/webhooks/falco
// Receives security events from Falcosidekick and persists them to the event store
func (h *Handler) HandleFalcoWebhook(w http.ResponseWriter, r *http.Request) {
	var payload FalcoWebhookPayload

	// Decode the webhook payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("Failed to decode Falco webhook payload", "error", err)
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	slog.Info("Received Falco event",
		"rule", payload.Rule,
		"priority", payload.Priority,
		"namespace", payload.OutputFields["k8s.ns.name"],
		"pod", payload.OutputFields["k8s.pod.name"],
		"uuid", payload.UUID,
	)

	// Convert Falcosidekick payload to SecurityEvent
	event := compliance.SecurityEvent{
		Timestamp: payload.Time.Format(time.RFC3339),
		Rule:      payload.Rule,
		Severity:  payload.Priority,
		Message:   payload.Output,
		Resource:  formatResource(payload.OutputFields),
	}

	// Persist event to store
	h.eventStore.Add(event)

	// Acknowledge receipt (200 OK tells Falcosidekick to stop retrying)
	w.WriteHeader(http.StatusOK)
}

// formatResource builds a resource identifier from Falco output fields
// Format: "namespace/podname" or "hostname" if K8s metadata is missing
func formatResource(fields map[string]interface{}) string {
	ns, _ := fields["k8s.ns.name"].(string)
	pod, _ := fields["k8s.pod.name"].(string)

	if ns != "" && pod != "" {
		return fmt.Sprintf("%s/%s", ns, pod)
	}

	// Fallback to hostname for non-K8s events
	if hostname, ok := fields["hostname"].(string); ok {
		return hostname
	}

	return ""
}
