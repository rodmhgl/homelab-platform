package scaffold

// ScaffoldRequest represents the request payload for POST /api/v1/scaffold
type ScaffoldRequest struct {
	// Template name (currently "go-service" or "python-service")
	Template string `json:"template"`

	// Project metadata
	ProjectName        string `json:"project_name"`
	ProjectDescription string `json:"project_description,omitempty"`

	// Go-specific fields
	GoModulePath string `json:"go_module_path,omitempty"`
	HTTPPort     int    `json:"http_port,omitempty"`
	GRPCPort     int    `json:"grpc_port,omitempty"`

	// Feature flags
	EnableGRPC     bool `json:"enable_grpc,omitempty"`
	EnableDatabase bool `json:"enable_database,omitempty"`
	EnableStorage  bool `json:"enable_storage,omitempty"`
	EnableKeyVault bool `json:"enable_keyvault,omitempty"`

	// Storage configuration (if enabled)
	StorageLocation        string `json:"storage_location,omitempty"`
	StorageReplication     string `json:"storage_replication,omitempty"`
	StoragePublicAccess    bool   `json:"storage_public_access,omitempty"`
	StorageContainerName   string `json:"storage_container_name,omitempty"`
	StorageConnectionEnv   string `json:"storage_connection_env,omitempty"`

	// Key Vault configuration (if enabled)
	VaultLocation      string `json:"vault_location,omitempty"`
	VaultSKU           string `json:"vault_sku,omitempty"`
	VaultPublicAccess  bool   `json:"vault_public_access,omitempty"`
	VaultConnectionEnv string `json:"vault_connection_env,omitempty"`

	// Team and ownership
	TeamName  string `json:"team_name,omitempty"`
	TeamEmail string `json:"team_email,omitempty"`
	Owners    string `json:"owners,omitempty"`

	// GitHub configuration (optional overrides)
	GithubOrg  string `json:"github_org,omitempty"`
	GithubRepo string `json:"github_repo,omitempty"`
	RepoPrivate bool   `json:"repo_private,omitempty"`
}

// ScaffoldResponse represents the response payload for POST /api/v1/scaffold
type ScaffoldResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`

	// Created resources
	RepoURL          string `json:"repo_url,omitempty"`
	RepoName         string `json:"repo_name,omitempty"`
	PlatformConfigPath string `json:"platform_config_path,omitempty"`

	// Argo CD information
	ArgoCDAppName string `json:"argocd_app_name,omitempty"`
}

// ArgoAppConfig represents the apps/<name>/config.json file structure
// that the Argo CD ApplicationSet watches
type ArgoAppConfig struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repoUrl"`
	Path        string `json:"path"`
	Namespace   string `json:"namespace"`
	Project     string `json:"project"`
	SyncPolicy  string `json:"syncPolicy"`
	AutoSync    bool   `json:"autoSync"`
}
