package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Storage tier options
var storageTiers = []string{"Standard", "Premium"}

// Storage redundancy options
var storageRedundancies = []string{"LRS", "ZRS", "GRS", "GZRS", "RAGRS", "RAGZRS"}

// Azure location options
var locations = []string{"southcentralus", "eastus2"}

// StorageModel represents the state of the storage creation TUI
type StorageModel struct {
	state            string // welcome, inputName, inputNamespace, inputLocation, inputTier, inputRedundancy, inputVersioning, inputRepoOwner, inputRepoName, confirmation, submitting, success, error
	name             string
	namespace        string
	location         string
	tier             string
	redundancy       string
	enableVersioning bool
	repoOwner        string
	repoName         string
	cursor           int
	input            textinput.Model
	err              error
	apiResponse      *CreateClaimResponse
	apiBaseURL       string
	authToken        string
}

// CreateClaimResponse matches the Platform API response
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

// CreateClaimRequest matches the Platform API request
type CreateClaimRequest struct {
	Kind       string                 `json:"kind"`
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Parameters map[string]interface{} `json:"parameters"`
	RepoOwner  string                 `json:"repoOwner"`
	RepoName   string                 `json:"repoName"`
}

// Messages for bubbletea
type submitSuccessMsg struct {
	response CreateClaimResponse
}

type submitErrorMsg struct {
	err error
}

// NewStorageModel creates a new storage creation model
func NewStorageModel(apiBaseURL, authToken string) StorageModel {
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

	return StorageModel{
		state:      "welcome",
		input:      ti,
		apiBaseURL: apiBaseURL,
		authToken:  authToken,
		namespace:  "default", // Default namespace
		repoOwner:  repoOwner,
		repoName:   repoName,
	}
}

func (m StorageModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m StorageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.state == "inputVersioning" {
				m.enableVersioning = true
				m.state = "inputRepoOwner"
				m.input.SetValue(m.repoOwner)
				m.input.Placeholder = "GitHub repository owner"
				return m, nil
			}

		case "n", "N":
			if m.state == "confirmation" {
				return m, tea.Quit
			}
			if m.state == "inputVersioning" {
				m.enableVersioning = false
				m.state = "inputRepoOwner"
				m.input.SetValue(m.repoOwner)
				m.input.Placeholder = "GitHub repository owner"
				return m, nil
			}

		case "up":
			if m.state == "inputLocation" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
			if m.state == "inputTier" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
			if m.state == "inputRedundancy" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}

		case "down":
			if m.state == "inputLocation" && m.cursor < len(locations)-1 {
				m.cursor++
				return m, nil
			}
			if m.state == "inputTier" && m.cursor < len(storageTiers)-1 {
				m.cursor++
				return m, nil
			}
			if m.state == "inputRedundancy" && m.cursor < len(storageRedundancies)-1 {
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

func (m StorageModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case "welcome":
		m.state = "inputName"
		m.input.Placeholder = "e.g., my-storage"
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
		m.state = "inputTier"
		m.cursor = 0
		return m, nil

	case "inputTier":
		m.tier = storageTiers[m.cursor]
		m.state = "inputRedundancy"
		m.cursor = 0
		return m, nil

	case "inputRedundancy":
		m.redundancy = storageRedundancies[m.cursor]
		m.state = "inputVersioning"
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

func (m StorageModel) submitClaim() tea.Cmd {
	return func() tea.Msg {
		req := CreateClaimRequest{
			Kind:      "StorageBucket",
			Name:      m.name,
			Namespace: m.namespace,
			Parameters: map[string]interface{}{
				"location":         m.location,
				"tier":             m.tier,
				"redundancy":       m.redundancy,
				"enableVersioning": m.enableVersioning,
				"publicAccess":     false, // Always false (Gatekeeper enforced)
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

func (m StorageModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Create StorageBucket Claim"))
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
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.state != "inputTier" && m.tier != "" {
		b.WriteString(RenderFieldRow("Tier", m.tier))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.state != "inputTier" && m.state != "inputRedundancy" && m.redundancy != "" {
		b.WriteString(RenderFieldRow("Redundancy", m.redundancy))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.state != "inputTier" && m.state != "inputRedundancy" && m.state != "inputVersioning" && m.state != "inputRepoOwner" {
		versionStr := "Disabled"
		if m.enableVersioning {
			versionStr = "Enabled"
		}
		b.WriteString(RenderFieldRow("Versioning", versionStr))
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
		b.WriteString("This wizard will guide you through creating an Azure Storage Account via Crossplane.\n\n")
		b.WriteString(HelpStyle.Render("Press Enter to begin, Ctrl+C to cancel"))

	case "inputName":
		b.WriteString(FieldLabelStyle.Render("Storage Bucket Name:"))
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

	case "inputTier":
		b.WriteString(FieldLabelStyle.Render("Storage Tier:"))
		b.WriteString("\n")
		for i, tier := range storageTiers {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, tier))
		}
		b.WriteString(HelpStyle.Render("Use arrow keys, Enter to select"))

	case "inputRedundancy":
		b.WriteString(FieldLabelStyle.Render("Redundancy:"))
		b.WriteString("\n")
		for i, red := range storageRedundancies {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, red))
		}
		b.WriteString(HelpStyle.Render("Use arrow keys, Enter to select"))

	case "inputVersioning":
		b.WriteString(FieldLabelStyle.Render("Enable Versioning?"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Y/N"))

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
		b.WriteString(FieldLabelStyle.Render("Ready to create StorageBucket Claim?"))
		b.WriteString("\n\n")
		b.WriteString("This will commit a Claim YAML to your repository.\n")
		b.WriteString("Argo CD will sync it within 60 seconds.\n\n")
		b.WriteString(HelpStyle.Render("Y to create, N to cancel"))

	case "submitting":
		b.WriteString(RenderSpinner("Creating StorageBucket Claim..."))

	case "success":
		if m.apiResponse != nil {
			details := map[string]string{
				"Commit SHA":        m.apiResponse.CommitSHA,
				"File Path":         m.apiResponse.FilePath,
				"Connection Secret": m.apiResponse.ConnectionSecret,
				"Repository":        m.apiResponse.RepoURL,
			}
			return RenderSuccess("StorageBucket Claim Created!", "Argo CD will sync this Claim within 60 seconds.", details)
		}
		return RenderSuccess("StorageBucket Claim Created!", "Check Argo CD for sync status.", nil)

	case "error":
		if m.err != nil {
			return RenderError("Failed to Create Claim", m.err.Error())
		}
		return RenderError("Failed to Create Claim", "An unknown error occurred")
	}

	return b.String()
}

// State returns the current state (for external checks)
func (m StorageModel) State() string {
	return m.state
}
