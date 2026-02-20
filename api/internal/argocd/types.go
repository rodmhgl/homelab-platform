package argocd

import "time"

// Application represents an Argo CD Application resource
type Application struct {
	Metadata ApplicationMetadata `json:"metadata"`
	Spec     ApplicationSpec     `json:"spec"`
	Status   ApplicationStatus   `json:"status"`
}

// ApplicationMetadata contains application metadata
type ApplicationMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	CreationTimestamp time.Time         `json:"creationTimestamp,omitempty"`
}

// ApplicationSpec defines the desired state of an application
type ApplicationSpec struct {
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
	Project     string                 `json:"project"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
}

// ApplicationSource contains information about the application's source
type ApplicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Chart          string `json:"chart,omitempty"`
}

// ApplicationDestination contains information about the application's destination
type ApplicationDestination struct {
	Server    string `json:"server,omitempty"`
	Namespace string `json:"namespace"`
	Name      string `json:"name,omitempty"`
}

// SyncPolicy controls when and how a sync will be performed
type SyncPolicy struct {
	Automated   *AutomatedSyncPolicy `json:"automated,omitempty"`
	SyncOptions []string             `json:"syncOptions,omitempty"`
	Retry       *RetryStrategy       `json:"retry,omitempty"`
}

// AutomatedSyncPolicy controls automated sync behavior
type AutomatedSyncPolicy struct {
	Prune      bool `json:"prune,omitempty"`
	SelfHeal   bool `json:"selfHeal,omitempty"`
	AllowEmpty bool `json:"allowEmpty,omitempty"`
}

// RetryStrategy controls retry behavior
type RetryStrategy struct {
	Limit   int64         `json:"limit,omitempty"`
	Backoff *RetryBackoff `json:"backoff,omitempty"`
}

// RetryBackoff controls backoff timing
type RetryBackoff struct {
	Duration    string `json:"duration,omitempty"`
	Factor      int64  `json:"factor,omitempty"`
	MaxDuration string `json:"maxDuration,omitempty"`
}

// ApplicationStatus contains information about the application's current state
type ApplicationStatus struct {
	Resources      []ResourceStatus    `json:"resources,omitempty"`
	Sync           SyncStatus          `json:"sync"`
	Health         HealthStatus        `json:"health"`
	History        []RevisionHistory   `json:"history,omitempty"`
	Conditions     []ApplicationCondition `json:"conditions,omitempty"`
	ReconciledAt   *time.Time          `json:"reconciledAt,omitempty"`
	OperationState *OperationState     `json:"operationState,omitempty"`
	Summary        ApplicationSummary  `json:"summary,omitempty"`
}

// ResourceStatus holds the current sync and health status of a resource
type ResourceStatus struct {
	Group     string       `json:"group,omitempty"`
	Version   string       `json:"version,omitempty"`
	Kind      string       `json:"kind"`
	Namespace string       `json:"namespace,omitempty"`
	Name      string       `json:"name"`
	Status    string       `json:"status,omitempty"`
	Health    HealthStatus `json:"health,omitempty"`
	SyncWave  int64        `json:"syncWave,omitempty"`
}

// SyncStatus contains information about the currently observed live and desired states of an application
type SyncStatus struct {
	Status     string    `json:"status"` // Synced, OutOfSync, Unknown
	ComparedTo ComparedTo `json:"comparedTo,omitempty"`
	Revision   string    `json:"revision,omitempty"`
}

// ComparedTo contains application source information
type ComparedTo struct {
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
}

// HealthStatus contains information about the application's current health status
type HealthStatus struct {
	Status  string `json:"status,omitempty"` // Healthy, Progressing, Degraded, Suspended, Missing, Unknown
	Message string `json:"message,omitempty"`
}

// RevisionHistory contains information about a previous sync
type RevisionHistory struct {
	Revision   string     `json:"revision"`
	DeployedAt time.Time  `json:"deployedAt"`
	ID         int64      `json:"id"`
	Source     ApplicationSource `json:"source,omitempty"`
}

// ApplicationCondition contains details about an application condition
type ApplicationCondition struct {
	Type               string     `json:"type"`
	Message            string     `json:"message"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
}

// OperationState contains information about ongoing or recent application operations
type OperationState struct {
	Operation  Operation  `json:"operation"`
	Phase      string     `json:"phase"` // Running, Succeeded, Failed, Error, Terminating
	Message    string     `json:"message,omitempty"`
	SyncResult *SyncResult `json:"syncResult,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// Operation contains information about the operation being performed
type Operation struct {
	Sync *SyncOperation `json:"sync,omitempty"`
}

// SyncOperation contains details about a sync operation
type SyncOperation struct {
	Revision    string   `json:"revision,omitempty"`
	Prune       bool     `json:"prune,omitempty"`
	DryRun      bool     `json:"dryRun,omitempty"`
	SyncOptions []string `json:"syncOptions,omitempty"`
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	Resources []ResourceResult `json:"resources,omitempty"`
	Revision  string           `json:"revision"`
	Source    ApplicationSource `json:"source,omitempty"`
}

// ResourceResult holds the operation result details of a specific resource
type ResourceResult struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	HookType  string `json:"hookType,omitempty"`
	HookPhase string `json:"hookPhase,omitempty"`
	SyncPhase string `json:"syncPhase,omitempty"`
}

// ApplicationSummary contains a summary of the application's state
type ApplicationSummary struct {
	ExternalURLs []string            `json:"externalURLs,omitempty"`
	Images       []string            `json:"images,omitempty"`
}

// ApplicationList is a list of Application resources
type ApplicationList struct {
	Items []Application `json:"items"`
}

// SyncRequest represents a request to sync an application
type SyncRequest struct {
	Revision    string   `json:"revision,omitempty"`
	Prune       bool     `json:"prune,omitempty"`
	DryRun      bool     `json:"dryRun,omitempty"`
	SyncOptions []string `json:"syncOptions,omitempty"`
	Resources   []SyncResource `json:"resources,omitempty"`
}

// SyncResource represents a resource to sync
type SyncResource struct {
	Group     string `json:"group,omitempty"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// ListAppsResponse is the response for listing applications
type ListAppsResponse struct {
	Applications []ApplicationSummaryResponse `json:"applications"`
	Total        int                          `json:"total"`
}

// ApplicationSummaryResponse is a simplified application view for list endpoints
type ApplicationSummaryResponse struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace,omitempty"`
	Project      string    `json:"project"`
	SyncStatus   string    `json:"syncStatus"`
	HealthStatus string    `json:"healthStatus"`
	RepoURL      string    `json:"repoURL"`
	Path         string    `json:"path,omitempty"`
	Revision     string    `json:"revision,omitempty"`
	LastDeployed *time.Time `json:"lastDeployed,omitempty"`
}
