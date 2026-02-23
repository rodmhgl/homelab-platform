package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Vault SKU options
var vaultSKUs = []string{"standard", "premium"}

// VaultModel represents the state of the vault creation TUI
type VaultModel struct {
	state                   string // welcome, inputName, inputNamespace, inputLocation, inputSKU, inputRetention, inputRepoOwner, inputRepoName, confirmation, submitting, success, error
	name                    string
	namespace               string
	location                string
	skuName                 string
	softDeleteRetentionDays int
	repoOwner               string
	repoName                string
	cursor                  int
	input                   textinput.Model
	err                     error
	apiResponse             *CreateClaimResponse
	apiBaseURL              string
	authToken               string
}

// NewVaultModel creates a new vault creation model
func NewVaultModel(apiBaseURL, authToken string) VaultModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 63
	ti.Width = 50

	// Try to detect git repo
	repo, _ := DetectGitRepo()
	repoOwner := ""
	repoName := ""
	if repo != nil {
		repoOwner = repo.Owner
		repoName = repo.Name
	}

	return VaultModel{
		state:                   "welcome",
		input:                   ti,
		apiBaseURL:              apiBaseURL,
		authToken:               authToken,
		namespace:               "default", // Default namespace
		softDeleteRetentionDays: 7,         // Default retention
		repoOwner:               repoOwner,
		repoName:                repoName,
	}
}

func (m VaultModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m VaultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			if m.state == "success" || m.state == "error" {
				return m, tea.Quit
			}

		case "r":
			if m.state == "error" {
				m.state = "confirmation"
				m.err = nil
				return m, nil
			}

		case "enter":
			return m.handleEnter()

		case "y", "Y":
			if m.state == "confirmation" {
				m.state = "submitting"
				return m, m.submitClaim()
			}

		case "n", "N":
			if m.state == "confirmation" {
				return m, tea.Quit
			}

		case "up":
			if m.state == "inputLocation" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
			if m.state == "inputSKU" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}

		case "down":
			if m.state == "inputLocation" && m.cursor < len(locations)-1 {
				m.cursor++
				return m, nil
			}
			if m.state == "inputSKU" && m.cursor < len(vaultSKUs)-1 {
				m.cursor++
				return m, nil
			}
		}

	case submitSuccessMsg:
		m.state = "success"
		m.apiResponse = &msg.response
		return m, nil

	case submitErrorMsg:
		m.state = "error"
		m.err = msg.err
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m VaultModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case "welcome":
		m.state = "inputName"
		m.input.Placeholder = "e.g., my-vault"
		return m, nil

	case "inputName":
		name := strings.TrimSpace(m.input.Value())
		if err := ValidateDNSLabel(name); err != nil {
			m.err = err
			return m, nil
		}
		m.name = name
		m.err = nil
		m.state = "inputNamespace"
		m.input.SetValue(m.namespace)
		m.input.Placeholder = "default"
		return m, nil

	case "inputNamespace":
		namespace := strings.TrimSpace(m.input.Value())
		if namespace == "" {
			namespace = "default"
		}
		if err := ValidateNamespace(namespace); err != nil {
			m.err = err
			return m, nil
		}
		m.namespace = namespace
		m.err = nil
		m.state = "inputLocation"
		m.cursor = 0
		return m, nil

	case "inputLocation":
		m.location = locations[m.cursor]
		m.state = "inputSKU"
		m.cursor = 0
		return m, nil

	case "inputSKU":
		m.skuName = vaultSKUs[m.cursor]
		m.state = "inputRetention"
		m.input.SetValue(strconv.Itoa(m.softDeleteRetentionDays))
		m.input.Placeholder = "7-90 days"
		return m, nil

	case "inputRetention":
		days := strings.TrimSpace(m.input.Value())
		n, err := strconv.Atoi(days)
		if err != nil {
			m.err = fmt.Errorf("retention days must be a number")
			return m, nil
		}
		if err := ValidateRetentionDays(n); err != nil {
			m.err = err
			return m, nil
		}
		m.softDeleteRetentionDays = n
		m.err = nil
		m.state = "inputRepoOwner"
		m.input.SetValue(m.repoOwner)
		m.input.Placeholder = "GitHub repository owner"
		return m, nil

	case "inputRepoOwner":
		owner := strings.TrimSpace(m.input.Value())
		if owner == "" {
			m.err = fmt.Errorf("repository owner cannot be empty")
			return m, nil
		}
		m.repoOwner = owner
		m.err = nil
		m.state = "inputRepoName"
		m.input.SetValue(m.repoName)
		m.input.Placeholder = "GitHub repository name"
		return m, nil

	case "inputRepoName":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			m.err = fmt.Errorf("repository name cannot be empty")
			return m, nil
		}
		m.repoName = name
		m.err = nil
		m.state = "confirmation"
		return m, nil
	}

	return m, nil
}

func (m VaultModel) submitClaim() tea.Cmd {
	return func() tea.Msg {
		req := CreateClaimRequest{
			Kind:      "Vault",
			Name:      m.name,
			Namespace: m.namespace,
			Parameters: map[string]interface{}{
				"location":                m.location,
				"skuName":                 m.skuName,
				"softDeleteRetentionDays": m.softDeleteRetentionDays,
				"publicAccess":            false, // Always false (Gatekeeper enforced)
			},
			RepoOwner: m.repoOwner,
			RepoName:  m.repoName,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return submitErrorMsg{err: fmt.Errorf("failed to marshal request: %w", err)}
		}

		client := &http.Client{Timeout: 30 * time.Second}
		httpReq, err := http.NewRequest("POST", m.apiBaseURL+"/api/v1/infra", bytes.NewReader(body))
		if err != nil {
			return submitErrorMsg{err: fmt.Errorf("failed to create request: %w", err)}
		}

		httpReq.Header.Set("Authorization", "Bearer "+m.authToken)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			return submitErrorMsg{err: fmt.Errorf("request failed: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			var errResp map[string]string
			if json.Unmarshal(bodyBytes, &errResp) == nil {
				if msg, ok := errResp["error"]; ok {
					return submitErrorMsg{err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)}
				}
			}
			return submitErrorMsg{err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))}
		}

		var response CreateClaimResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return submitErrorMsg{err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return submitSuccessMsg{response: response}
	}
}

func (m VaultModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Create Vault Claim"))
	b.WriteString("\n\n")

	// Show completed fields
	if m.state != "welcome" && m.name != "" {
		b.WriteString(RenderFieldRow("Name", m.name))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.namespace != "" {
		b.WriteString(RenderFieldRow("Namespace", m.namespace))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.location != "" {
		b.WriteString(RenderFieldRow("Location", m.location))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.state != "inputSKU" && m.skuName != "" {
		b.WriteString(RenderFieldRow("SKU", m.skuName))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.state != "inputSKU" && m.state != "inputRetention" {
		b.WriteString(RenderFieldRow("Retention Days", strconv.Itoa(m.softDeleteRetentionDays)))
		b.WriteString("\n")
	}
	if m.state == "inputRepoName" || m.state == "confirmation" || m.state == "submitting" {
		b.WriteString(RenderFieldRow("Repo Owner", m.repoOwner))
		b.WriteString("\n")
	}
	if m.state == "confirmation" || m.state == "submitting" {
		b.WriteString(RenderFieldRow("Repo Name", m.repoName))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Show current state
	switch m.state {
	case "welcome":
		b.WriteString("This wizard will guide you through creating an Azure Key Vault via Crossplane.\n\n")
		b.WriteString(HelpStyle.Render("Press Enter to begin, Ctrl+C to cancel"))

	case "inputName":
		b.WriteString(FieldLabelStyle.Render("Vault Name:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Lowercase alphanumeric with hyphens, max 63 chars"))

	case "inputNamespace":
		b.WriteString(FieldLabelStyle.Render("Namespace:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Press Enter for default"))

	case "inputLocation":
		b.WriteString(FieldLabelStyle.Render("Azure Location:"))
		b.WriteString("\n")
		for i, loc := range locations {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, loc))
		}
		b.WriteString(HelpStyle.Render("Use arrow keys, Enter to select"))

	case "inputSKU":
		b.WriteString(FieldLabelStyle.Render("Vault SKU:"))
		b.WriteString("\n")
		for i, sku := range vaultSKUs {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, sku))
		}
		b.WriteString(HelpStyle.Render("Use arrow keys, Enter to select"))

	case "inputRetention":
		b.WriteString(FieldLabelStyle.Render("Soft Delete Retention Days:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Must be between 7 and 90 days"))

	case "inputRepoOwner":
		b.WriteString(FieldLabelStyle.Render("GitHub Repository Owner:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Repository owner (e.g., rodmhgl)"))

	case "inputRepoName":
		b.WriteString(FieldLabelStyle.Render("GitHub Repository Name:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Repository name (e.g., my-app)"))

	case "confirmation":
		b.WriteString(FieldLabelStyle.Render("Ready to create Vault Claim?"))
		b.WriteString("\n\n")
		b.WriteString("This will commit a Claim YAML to your repository.\n")
		b.WriteString("Argo CD will sync it within 60 seconds.\n\n")
		b.WriteString(HelpStyle.Render("Y to create, N to cancel"))

	case "submitting":
		b.WriteString(RenderSpinner("Creating Vault Claim..."))

	case "success":
		if m.apiResponse != nil {
			details := map[string]string{
				"Commit SHA":        m.apiResponse.CommitSHA,
				"File Path":         m.apiResponse.FilePath,
				"Connection Secret": m.apiResponse.ConnectionSecret,
				"Repository":        m.apiResponse.RepoURL,
			}
			return RenderSuccess("Vault Claim Created!", "Argo CD will sync this Claim within 60 seconds.", details)
		}
		return RenderSuccess("Vault Claim Created!", "Check Argo CD for sync status.", nil)

	case "error":
		if m.err != nil {
			return RenderError("Failed to Create Claim", m.err.Error())
		}
		return RenderError("Failed to Create Claim", "An unknown error occurred")
	}

	return b.String()
}

// State returns the current state (for external checks)
func (m VaultModel) State() string {
	return m.state
}
