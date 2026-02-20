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
	client *Client
}

// NewHandler creates a new infrastructure handler
func NewHandler(cfg *Config) (*Handler, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Handler{
		client: client,
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
