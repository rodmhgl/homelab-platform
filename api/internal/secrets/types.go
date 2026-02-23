package secrets

import "time"

// ListSecretsResponse is the top-level response for the list secrets endpoint.
type ListSecretsResponse struct {
	Secrets []SecretSummary `json:"secrets"`
	Total   int             `json:"total"`
}

// SecretSummary provides a unified view of both ExternalSecrets and connection Secrets.
type SecretSummary struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Kind              string            `json:"kind"`                    // "ExternalSecret" or "Secret"
	Type              string            `json:"type,omitempty"`          // Secret type (Opaque, kubernetes.io/tls, etc.)
	Status            string            `json:"status,omitempty"`        // "Ready", "Error", "Synced"
	Message           string            `json:"message,omitempty"`       // Status condition message
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
	Keys              []string          `json:"keys,omitempty"`          // Secret key names (non-sensitive)
	SourceClaim       *ResourceRef      `json:"sourceClaim,omitempty"`   // Crossplane Claim reference
}

// ResourceRef references a related Kubernetes resource.
type ResourceRef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}
