package tui

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	FieldLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	FieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	StatusCheckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	StatusPendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

// DNS label regex: lowercase alphanumeric with hyphens, no leading/trailing hyphens
var dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidateDNSLabel validates Kubernetes DNS label format
func ValidateDNSLabel(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("name cannot be empty")
	}
	if len(s) > 63 {
		return fmt.Errorf("name must be 63 characters or less (got %d)", len(s))
	}
	if !dnsLabelRegex.MatchString(s) {
		return fmt.Errorf("invalid DNS label: must be lowercase alphanumeric with hyphens, no leading/trailing hyphens")
	}
	return nil
}

// ValidateNamespace validates Kubernetes namespace format
func ValidateNamespace(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("namespace cannot be empty")
	}
	return ValidateDNSLabel(s)
}

// ValidateLocation validates Azure location against allowed list
func ValidateLocation(s string) error {
	allowed := []string{"southcentralus", "eastus2"}
	for _, loc := range allowed {
		if s == loc {
			return nil
		}
	}
	return fmt.Errorf("location must be one of: %v", allowed)
}

// ValidateRetentionDays validates Key Vault soft delete retention days
func ValidateRetentionDays(n int) error {
	if n < 7 || n > 90 {
		return fmt.Errorf("retention days must be between 7 and 90 (got %d)", n)
	}
	return nil
}

// GitRepo represents a detected Git repository
type GitRepo struct {
	Owner string
	Name  string
}

// DetectGitRepo attempts to parse git remote origin URL
func DetectGitRepo() (*GitRepo, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository or no origin remote")
	}

	url := strings.TrimSpace(string(output))
	return ParseGitURL(url)
}

// ParseGitURL parses SSH and HTTPS GitHub URLs
func ParseGitURL(url string) (*GitRepo, error) {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// SSH format: git@github.com:owner/repo
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid SSH URL format")
		}
		path := parts[1]
		ownerRepo := strings.Split(path, "/")
		if len(ownerRepo) != 2 {
			return nil, fmt.Errorf("invalid repository path")
		}
		return &GitRepo{Owner: ownerRepo[0], Name: ownerRepo[1]}, nil
	}

	// HTTPS format: https://github.com/owner/repo
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.Split(url, "/")
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid HTTPS URL format")
		}
		return &GitRepo{Owner: parts[3], Name: parts[4]}, nil
	}

	return nil, fmt.Errorf("unsupported URL format: %s", url)
}

// RenderFieldRow renders a completed field with label, value, and checkmark
func RenderFieldRow(label, value string) string {
	checkmark := StatusCheckStyle.Render("✓")
	labelStr := FieldLabelStyle.Render(label + ":")
	valueStr := FieldValueStyle.Render(value)
	return fmt.Sprintf("%s %s %s", checkmark, labelStr, valueStr)
}

// RenderSpinner renders a loading spinner message
func RenderSpinner(msg string) string {
	return StatusPendingStyle.Render("⏳ " + msg)
}

// RenderSuccess renders a success screen with details
func RenderSuccess(title, message string, details map[string]string) string {
	var b strings.Builder

	b.WriteString(SuccessStyle.Render("✓ " + title))
	b.WriteString("\n\n")
	b.WriteString(message)
	b.WriteString("\n\n")

	if len(details) > 0 {
		b.WriteString(FieldLabelStyle.Render("Details:"))
		b.WriteString("\n")
		for k, v := range details {
			b.WriteString(fmt.Sprintf("  %s %s\n", FieldLabelStyle.Render(k+":"), FieldValueStyle.Render(v)))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Press q to quit"))

	return BoxStyle.Render(b.String())
}

// RenderError renders an error screen
func RenderError(title, message string) string {
	var b strings.Builder

	b.WriteString(ErrorStyle.Render("✗ " + title))
	b.WriteString("\n\n")
	b.WriteString(message)
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("Press r to retry, q to quit"))

	return BoxStyle.Render(b.String())
}

// StringInSlice checks if a string exists in a slice
func StringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
