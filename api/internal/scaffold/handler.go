package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// Config holds configuration for the scaffold service
type Config struct {
	GithubToken       string
	GithubOrg         string
	PlatformRepo      string
	ScaffoldTemplates string
	WorkDir           string
}

// Handler handles scaffold HTTP requests
type Handler struct {
	config *Config
	github *github.Client
}

// NewHandler creates a new scaffold handler
func NewHandler(cfg *Config) (*Handler, error) {
	// Validate required configuration
	if cfg.GithubToken == "" {
		return nil, fmt.Errorf("github token is required")
	}
	if cfg.GithubOrg == "" {
		return nil, fmt.Errorf("github org is required")
	}
	if cfg.PlatformRepo == "" {
		return nil, fmt.Errorf("platform repo is required")
	}
	if cfg.ScaffoldTemplates == "" {
		return nil, fmt.Errorf("scaffold templates path is required")
	}

	// Initialize GitHub client with OAuth2 token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GithubToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	githubClient := github.NewClient(tc)

	return &Handler{
		config: cfg,
		github: githubClient,
	}, nil
}

// HandleCreate handles POST /api/v1/scaffold requests
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req ScaffoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to parse scaffold request", "error", err)
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		slog.Error("Invalid scaffold request", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Apply defaults
	h.applyDefaults(&req)

	slog.Info("Processing scaffold request",
		"template", req.Template,
		"project_name", req.ProjectName,
		"github_org", req.GithubOrg,
		"github_repo", req.GithubRepo,
	)

	// Execute scaffold workflow
	response, err := h.executeScaffold(ctx, &req)
	if err != nil {
		slog.Error("Scaffold execution failed", "error", err, "project", req.ProjectName)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Scaffold failed: %v", err))
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	slog.Info("Scaffold completed successfully",
		"project", req.ProjectName,
		"repo_url", response.RepoURL,
	)
}

// validateRequest validates the scaffold request
func (h *Handler) validateRequest(req *ScaffoldRequest) error {
	// Validate template
	validTemplates := map[string]bool{
		"go-service":     true,
		"python-service": true,
	}
	if !validTemplates[req.Template] {
		return fmt.Errorf("invalid template: %s (must be 'go-service' or 'python-service')", req.Template)
	}

	// Validate project name (must match Copier validators)
	if req.ProjectName == "" {
		return fmt.Errorf("project_name is required")
	}
	if strings.ToLower(req.ProjectName) != req.ProjectName {
		return fmt.Errorf("project_name must be lowercase")
	}
	if len(req.ProjectName) < 3 {
		return fmt.Errorf("project_name must be at least 3 characters")
	}
	if len(req.ProjectName) > 63 {
		return fmt.Errorf("project_name cannot exceed 63 characters")
	}
	if req.ProjectName[0] == '-' || req.ProjectName[len(req.ProjectName)-1] == '-' {
		return fmt.Errorf("project_name cannot start or end with a hyphen")
	}

	// Validate ports if specified
	if req.HTTPPort != 0 && (req.HTTPPort < 1024 || req.HTTPPort > 65535) {
		return fmt.Errorf("http_port must be between 1024 and 65535")
	}
	if req.EnableGRPC && req.GRPCPort != 0 && (req.GRPCPort < 1024 || req.GRPCPort > 65535) {
		return fmt.Errorf("grpc_port must be between 1024 and 65535")
	}

	// Validate storage configuration if enabled
	if req.EnableStorage {
		validLocations := map[string]bool{
			"eastus": true, "westus": true, "centralus": true, "southcentralus": true,
		}
		if req.StorageLocation != "" && !validLocations[req.StorageLocation] {
			return fmt.Errorf("invalid storage_location: %s", req.StorageLocation)
		}

		validReplications := map[string]bool{
			"LRS": true, "GRS": true, "ZRS": true,
		}
		if req.StorageReplication != "" && !validReplications[req.StorageReplication] {
			return fmt.Errorf("invalid storage_replication: %s (must be LRS, GRS, or ZRS)", req.StorageReplication)
		}
	}

	return nil
}

// applyDefaults applies default values to the request
func (h *Handler) applyDefaults(req *ScaffoldRequest) {
	// GitHub defaults
	if req.GithubOrg == "" {
		req.GithubOrg = h.config.GithubOrg
	}
	if req.GithubRepo == "" {
		req.GithubRepo = req.ProjectName
	}

	// Go defaults
	if req.Template == "go-service" {
		if req.GoModulePath == "" {
			req.GoModulePath = fmt.Sprintf("github.com/%s/%s", req.GithubOrg, req.ProjectName)
		}
		if req.HTTPPort == 0 {
			req.HTTPPort = 8080
		}
		if req.EnableGRPC && req.GRPCPort == 0 {
			req.GRPCPort = 9090
		}
	}

	// Storage defaults
	if req.EnableStorage {
		if req.StorageLocation == "" {
			req.StorageLocation = "southcentralus"
		}
		if req.StorageReplication == "" {
			req.StorageReplication = "LRS"
		}
		if req.StorageContainerName == "" {
			req.StorageContainerName = "data"
		}
		if req.StorageConnectionEnv == "" {
			req.StorageConnectionEnv = "STORAGE_CONNECTION_STRING"
		}
	}

	// Vault defaults
	if req.EnableKeyVault {
		if req.VaultLocation == "" {
			req.VaultLocation = "southcentralus"
		}
		if req.VaultSKU == "" {
			req.VaultSKU = "standard"
		}
		if req.VaultConnectionEnv == "" {
			req.VaultConnectionEnv = "KEYVAULT_URI"
		}
	}

	// Team defaults
	if req.TeamName == "" {
		req.TeamName = "platform"
	}
	if req.TeamEmail == "" {
		req.TeamEmail = "platform@example.com"
	}
	if req.Owners == "" {
		req.Owners = "@rodmhgl"
	}

	// Project description default
	if req.ProjectDescription == "" {
		req.ProjectDescription = "A microservice built on the platform"
	}
}

// executeScaffold runs the complete scaffold workflow
func (h *Handler) executeScaffold(ctx context.Context, req *ScaffoldRequest) (*ScaffoldResponse, error) {
	// Step 1: Create temporary directory for scaffold output
	workDir := filepath.Join(h.config.WorkDir, req.ProjectName)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir) // Clean up after we're done

	// Step 2: Run Copier to generate project files
	if err := h.runCopier(ctx, req, workDir); err != nil {
		return nil, fmt.Errorf("copier execution failed: %w", err)
	}

	// Step 3: Create GitHub repository
	repoURL, err := h.createGitHubRepo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub repo: %w", err)
	}

	// Step 4: Initialize git and push to new repo
	projectDir := filepath.Join(workDir, req.ProjectName)
	if err := h.initAndPushRepo(ctx, projectDir, repoURL, req); err != nil {
		return nil, fmt.Errorf("failed to push to GitHub: %w", err)
	}

	// Step 5: Commit config.json to platform repo
	configPath, err := h.commitPlatformConfig(ctx, req, repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to commit platform config: %w", err)
	}

	return &ScaffoldResponse{
		Success:            true,
		Message:            fmt.Sprintf("Successfully scaffolded %s from %s template", req.ProjectName, req.Template),
		RepoURL:            repoURL,
		RepoName:           req.GithubRepo,
		PlatformConfigPath: configPath,
		ArgoCDAppName:      req.ProjectName,
	}, nil
}

// runCopier executes the Copier CLI to generate project files
func (h *Handler) runCopier(ctx context.Context, req *ScaffoldRequest, workDir string) error {
	templatePath := filepath.Join(h.config.ScaffoldTemplates, req.Template)

	// Build Copier data flags from request
	data := h.buildCopierData(req)

	// Build command: copier copy --data key=value ... template_path dest_path
	args := []string{"copy", "--trust"}
	for key, value := range data {
		args = append(args, "--data", fmt.Sprintf("%s=%v", key, value))
	}
	args = append(args, templatePath, workDir)

	slog.Info("Executing Copier", "template", req.Template, "dest", workDir)

	cmd := exec.CommandContext(ctx, "copier", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copier command failed: %w", err)
	}

	return nil
}

// buildCopierData constructs the data map for Copier from the request
func (h *Handler) buildCopierData(req *ScaffoldRequest) map[string]interface{} {
	data := map[string]interface{}{
		"project_name":        req.ProjectName,
		"project_description": req.ProjectDescription,
		"team_name":           req.TeamName,
		"team_email":          req.TeamEmail,
		"owners":              req.Owners,
	}

	// Go-specific fields
	if req.Template == "go-service" {
		data["go_module_path"] = req.GoModulePath
		data["http_port"] = req.HTTPPort
		data["enable_grpc"] = req.EnableGRPC
		if req.EnableGRPC {
			data["grpc_port"] = req.GRPCPort
		}
	}

	// Feature flags
	data["enable_database"] = req.EnableDatabase
	data["enable_storage"] = req.EnableStorage
	data["enable_keyvault"] = req.EnableKeyVault

	// Storage configuration
	if req.EnableStorage {
		data["storage_location"] = req.StorageLocation
		data["storage_replication"] = req.StorageReplication
		data["storage_public_access"] = req.StoragePublicAccess
		data["storage_container_name"] = req.StorageContainerName
		data["storage_connection_env"] = req.StorageConnectionEnv
	}

	// Vault configuration
	if req.EnableKeyVault {
		data["vault_location"] = req.VaultLocation
		data["vault_sku"] = req.VaultSKU
		data["vault_public_access"] = req.VaultPublicAccess
		data["vault_connection_env"] = req.VaultConnectionEnv
	}

	return data
}

// Helper function to send error responses
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
