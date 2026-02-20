package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v66/github"
)

// createGitHubRepo creates a new GitHub repository for the scaffolded project
func (h *Handler) createGitHubRepo(ctx context.Context, req *ScaffoldRequest) (string, error) {
	slog.Info("Creating GitHub repository",
		"org", req.GithubOrg,
		"repo", req.GithubRepo,
		"private", req.RepoPrivate,
	)

	// Check if repo already exists
	existing, _, err := h.github.Repositories.Get(ctx, req.GithubOrg, req.GithubRepo)
	if err == nil {
		return "", fmt.Errorf("repository %s/%s already exists: %s", req.GithubOrg, req.GithubRepo, *existing.HTMLURL)
	}

	// Create repository
	description := req.ProjectDescription
	repo := &github.Repository{
		Name:        github.String(req.GithubRepo),
		Description: &description,
		Private:     github.Bool(req.RepoPrivate),
		AutoInit:    github.Bool(false), // We'll push our own initial commit
	}

	created, _, err := h.github.Repositories.Create(ctx, req.GithubOrg, repo)
	if err != nil {
		return "", fmt.Errorf("failed to create repository: %w", err)
	}

	slog.Info("GitHub repository created",
		"url", *created.CloneURL,
		"ssh_url", *created.SSHURL,
	)

	return *created.CloneURL, nil
}

// commitPlatformConfig commits the apps/<name>/config.json file to the platform repo
// This triggers Argo CD's ApplicationSet to discover and deploy the new application
func (h *Handler) commitPlatformConfig(ctx context.Context, req *ScaffoldRequest, repoURL string) (string, error) {
	configPath := fmt.Sprintf("apps/%s/config.json", req.ProjectName)

	slog.Info("Committing platform config",
		"path", configPath,
		"platform_repo", h.config.PlatformRepo,
	)

	// Create Argo CD config structure
	config := &ArgoAppConfig{
		Name:       req.ProjectName,
		RepoURL:    repoURL,
		Path:       "k8s",
		Namespace:  req.ProjectName,
		Project:    "workloads",
		SyncPolicy: "automated",
		AutoSync:   true,
	}

	// Marshal to JSON with indentation
	content, err := marshalJSON(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Commit the file to the platform repo
	commitMessage := fmt.Sprintf("Add %s application config\n\nScaffolded from %s template", req.ProjectName, req.Template)

	// Get the latest commit SHA for the main branch
	ref, _, err := h.github.Git.GetRef(ctx, h.config.GithubOrg, h.config.PlatformRepo, "refs/heads/main")
	if err != nil {
		return "", fmt.Errorf("failed to get main branch ref: %w", err)
	}

	// Get the tree SHA for the latest commit
	commit, _, err := h.github.Git.GetCommit(ctx, h.config.GithubOrg, h.config.PlatformRepo, *ref.Object.SHA)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit: %w", err)
	}

	// Create a blob for the new file
	blob := &github.Blob{
		Content:  github.String(content),
		Encoding: github.String("utf-8"),
	}
	createdBlob, _, err := h.github.Git.CreateBlob(ctx, h.config.GithubOrg, h.config.PlatformRepo, blob)
	if err != nil {
		return "", fmt.Errorf("failed to create blob: %w", err)
	}

	// Create a tree with the new file
	entries := []*github.TreeEntry{
		{
			Path: github.String(configPath),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  createdBlob.SHA,
		},
	}

	tree, _, err := h.github.Git.CreateTree(ctx, h.config.GithubOrg, h.config.PlatformRepo, *commit.Tree.SHA, entries)
	if err != nil {
		return "", fmt.Errorf("failed to create tree: %w", err)
	}

	// Create a new commit
	parent := commit
	newCommit := &github.Commit{
		Message: github.String(commitMessage),
		Tree:    tree,
		Parents: []*github.Commit{parent},
	}

	createdCommit, _, err := h.github.Git.CreateCommit(ctx, h.config.GithubOrg, h.config.PlatformRepo, newCommit, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Update the reference to point to the new commit
	ref.Object.SHA = createdCommit.SHA
	_, _, err = h.github.Git.UpdateRef(ctx, h.config.GithubOrg, h.config.PlatformRepo, ref, false)
	if err != nil {
		return "", fmt.Errorf("failed to update ref: %w", err)
	}

	slog.Info("Platform config committed",
		"path", configPath,
		"commit_sha", *createdCommit.SHA,
	)

	return configPath, nil
}

// marshalJSON is a helper to marshal structs to indented JSON
func marshalJSON(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
