package infra

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// GitHubClient handles GitHub API operations for infrastructure Claims
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client with the provided token
func NewGitHubClient(token string) *GitHubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &GitHubClient{
		client: github.NewClient(tc),
	}
}

// CommitClaim commits a Claim YAML file to the app repository
// Returns the commit SHA on success
func (g *GitHubClient) CommitClaim(ctx context.Context, owner, repo, filePath, yamlContent, commitMessage string) (string, error) {
	slog.Info("Committing Claim to GitHub",
		"owner", owner,
		"repo", repo,
		"path", filePath,
	)

	// 1. Get the latest commit SHA for main branch
	ref, _, err := g.client.Git.GetRef(ctx, owner, repo, "refs/heads/main")
	if err != nil {
		return "", fmt.Errorf("failed to get main branch ref: %w", err)
	}

	// 2. Get the tree SHA for the latest commit
	commit, _, err := g.client.Git.GetCommit(ctx, owner, repo, *ref.Object.SHA)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit: %w", err)
	}

	// 3. Create a blob for the YAML content
	blob := &github.Blob{
		Content:  github.String(yamlContent),
		Encoding: github.String("utf-8"),
	}
	createdBlob, _, err := g.client.Git.CreateBlob(ctx, owner, repo, blob)
	if err != nil {
		return "", fmt.Errorf("failed to create blob: %w", err)
	}

	// 4. Create a tree with the new file
	entries := []*github.TreeEntry{
		{
			Path: github.String(filePath),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  createdBlob.SHA,
		},
	}

	tree, _, err := g.client.Git.CreateTree(ctx, owner, repo, *commit.Tree.SHA, entries)
	if err != nil {
		return "", fmt.Errorf("failed to create tree: %w", err)
	}

	// 5. Create a new commit
	parent := commit
	newCommit := &github.Commit{
		Message: github.String(commitMessage),
		Tree:    tree,
		Parents: []*github.Commit{parent},
	}

	createdCommit, _, err := g.client.Git.CreateCommit(ctx, owner, repo, newCommit, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// 6. Update the reference to point to the new commit
	ref.Object.SHA = createdCommit.SHA
	_, _, err = g.client.Git.UpdateRef(ctx, owner, repo, ref, false)
	if err != nil {
		return "", fmt.Errorf("failed to update ref: %w", err)
	}

	slog.Info("Claim committed successfully",
		"commit_sha", *createdCommit.SHA,
		"file_path", filePath,
	)

	return *createdCommit.SHA, nil
}
