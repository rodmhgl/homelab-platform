package infra

import "time"

// GetResourceResponse is the response for GET /api/v1/infra/:kind/:name
type GetResourceResponse struct {
	Claim     ClaimResource      `json:"claim"`
	Composite *CompositeResource `json:"composite,omitempty"`
	Managed   []ManagedResource  `json:"managed"`
	Events    []KubernetesEvent  `json:"events"`
}

// ListClaimsResponse is the response for GET /api/v1/infra, /api/v1/infra/storage, /api/v1/infra/vaults
type ListClaimsResponse struct {
	Claims []ClaimSummary `json:"claims"`
	Total  int            `json:"total"`
}

// ClaimSummary is a lightweight representation of a Claim for list views
type ClaimSummary struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Kind              string            `json:"kind"`
	Status            string            `json:"status"` // Ready, Progressing, Failed
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	ConnectionSecret  string            `json:"connectionSecret,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// ClaimResource represents a Crossplane Claim (StorageBucket or Vault)
type ClaimResource struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Kind              string            `json:"kind"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	Status            string            `json:"status"` // Ready, Progressing, Failed
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	ConnectionSecret  string            `json:"connectionSecret,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	ResourceRef       *ResourceRef      `json:"resourceRef,omitempty"` // Reference to composite
}

// CompositeResource represents the XStorageBucket or XKeyVault composite
type CompositeResource struct {
	Name              string            `json:"name"`
	Kind              string            `json:"kind"`
	Labels            map[string]string `json:"labels,omitempty"`
	Status            string            `json:"status"`
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	ResourceRefs      []ResourceRef     `json:"resourceRefs,omitempty"` // References to managed resources
}

// ManagedResource represents an Azure resource provisioned by Crossplane
type ManagedResource struct {
	Name              string            `json:"name"`
	Kind              string            `json:"kind"`
	Group             string            `json:"group"`
	Labels            map[string]string `json:"labels,omitempty"`
	Status            string            `json:"status"`
	Synced            bool              `json:"synced"`
	Ready             bool              `json:"ready"`
	ExternalName      string            `json:"externalName,omitempty"` // Azure resource name
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	Message           string            `json:"message,omitempty"` // Latest status message
}

// ResourceRef represents a reference to another resource
type ResourceRef struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// KubernetesEvent represents a Kubernetes event for debugging
type KubernetesEvent struct {
	Type           string    `json:"type"` // Normal, Warning
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	InvolvedObject string    `json:"involvedObject"` // "kind/name"
	Source         string    `json:"source,omitempty"`
	Count          int32     `json:"count,omitempty"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
}

// CreateClaimRequest is the request body for POST /api/v1/infra
type CreateClaimRequest struct {
	Kind       string                 `json:"kind"`             // "StorageBucket" or "Vault"
	Name       string                 `json:"name"`             // DNS label format
	Namespace  string                 `json:"namespace"`        // Target namespace
	Parameters map[string]interface{} `json:"parameters"`       // XRD-specific params
	RepoOwner  string                 `json:"repoOwner"`        // GitHub org/user
	RepoName   string                 `json:"repoName"`         // App repo name
	Labels     map[string]string      `json:"labels,omitempty"` // Optional labels
}

// CreateClaimResponse is the response for POST /api/v1/infra
type CreateClaimResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message,omitempty"`
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	CommitSHA        string `json:"commitSha"`
	FilePath         string `json:"filePath"`
	RepoURL          string `json:"repoUrl"`
	ConnectionSecret string `json:"connectionSecret"`
}

// DeleteClaimRequest is the request body for DELETE /api/v1/infra/:kind/:name
type DeleteClaimRequest struct {
	RepoOwner string `json:"repoOwner"` // GitHub org/user
	RepoName  string `json:"repoName"`  // App repo name
}

// DeleteClaimResponse is the response for DELETE /api/v1/infra/:kind/:name
type DeleteClaimResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CommitSHA string `json:"commitSha"`
	FilePath  string `json:"filePath"`
	RepoURL   string `json:"repoUrl"`
}
