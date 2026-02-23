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

// Template options
var templates = []string{"go-service"}

// ScaffoldModel represents the state of the scaffold creation TUI
type ScaffoldModel struct {
	state              string // welcome, selectTemplate, inputProjectName, inputProjectDescription, inputHTTPPort, inputEnableGRPC, inputGRPCPort, inputEnableDatabase, inputEnableStorage, inputEnableKeyVault, inputGithubOrg, inputGithubRepo, confirmation, submitting, success, error
	template           string
	projectName        string
	projectDescription string
	httpPort           int
	grpcPort           int
	enableGRPC         bool
	enableDatabase     bool
	enableStorage      bool
	enableKeyVault     bool
	githubOrg          string
	githubRepo         string
	cursor             int
	input              textinput.Model
	err                error
	apiResponse        *ScaffoldResponse
	apiBaseURL         string
	authToken          string
}

// ScaffoldResponse matches the Platform API response
type ScaffoldResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message,omitempty"`
	Error              string `json:"error,omitempty"`
	RepoURL            string `json:"repo_url,omitempty"`
	RepoName           string `json:"repo_name,omitempty"`
	PlatformConfigPath string `json:"platform_config_path,omitempty"`
	ArgoCDAppName      string `json:"argocd_app_name,omitempty"`
}

// Messages for scaffold bubbletea
type submitScaffoldSuccessMsg struct {
	response ScaffoldResponse
}

type submitScaffoldErrorMsg struct {
	err error
}

// ScaffoldRequest matches the Platform API request
type ScaffoldRequest struct {
	Template           string `json:"template"`
	ProjectName        string `json:"project_name"`
	ProjectDescription string `json:"project_description,omitempty"`
	GoModulePath       string `json:"go_module_path,omitempty"`
	HTTPPort           int    `json:"http_port,omitempty"`
	GRPCPort           int    `json:"grpc_port,omitempty"`
	EnableGRPC         bool   `json:"enable_grpc,omitempty"`
	EnableDatabase     bool   `json:"enable_database,omitempty"`
	EnableStorage      bool   `json:"enable_storage,omitempty"`
	EnableKeyVault     bool   `json:"enable_keyvault,omitempty"`
	GithubOrg          string `json:"github_org,omitempty"`
	GithubRepo         string `json:"github_repo,omitempty"`
	RepoPrivate        bool   `json:"repo_private,omitempty"`
}

// NewScaffoldModel creates a new scaffold creation model
func NewScaffoldModel(apiBaseURL, authToken string) ScaffoldModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 63
	ti.Width = 50

	// Git auto-detection
	repo, _ := DetectGitRepo()
	githubOrg := ""
	if repo != nil {
		githubOrg = repo.Owner
	}

	return ScaffoldModel{
		state:      "welcome",
		input:      ti,
		apiBaseURL: apiBaseURL,
		authToken:  authToken,
		httpPort:   8080, // Default
		grpcPort:   9090, // Default
		githubOrg:  githubOrg,
	}
}

func (m ScaffoldModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ScaffoldModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, m.submitRequest()
			}
			if m.state == "inputEnableGRPC" {
				m.enableGRPC = true
				m.state = "inputGRPCPort"
				m.input.SetValue(fmt.Sprintf("%d", m.grpcPort))
				m.input.Placeholder = "9090"
				return m, nil
			}
			if m.state == "inputEnableDatabase" {
				m.enableDatabase = true
				m.state = "inputEnableStorage"
				return m, nil
			}
			if m.state == "inputEnableStorage" {
				m.enableStorage = true
				m.state = "inputEnableKeyVault"
				return m, nil
			}
			if m.state == "inputEnableKeyVault" {
				m.enableKeyVault = true
				m.state = "inputGithubOrg"
				m.input.SetValue(m.githubOrg)
				m.input.Placeholder = "GitHub organization/owner"
				return m, nil
			}

		case "n", "N":
			if m.state == "confirmation" {
				return m, tea.Quit
			}
			if m.state == "inputEnableGRPC" {
				m.enableGRPC = false
				m.state = "inputEnableDatabase"
				return m, nil
			}
			if m.state == "inputEnableDatabase" {
				m.enableDatabase = false
				m.state = "inputEnableStorage"
				return m, nil
			}
			if m.state == "inputEnableStorage" {
				m.enableStorage = false
				m.state = "inputEnableKeyVault"
				return m, nil
			}
			if m.state == "inputEnableKeyVault" {
				m.enableKeyVault = false
				m.state = "inputGithubOrg"
				m.input.SetValue(m.githubOrg)
				m.input.Placeholder = "GitHub organization/owner"
				return m, nil
			}

		case "up":
			if m.state == "selectTemplate" && m.cursor > 0 {
				m.cursor--
				return m, nil
			}

		case "down":
			if m.state == "selectTemplate" && m.cursor < len(templates)-1 {
				m.cursor++
				return m, nil
			}
		}

	case submitScaffoldSuccessMsg:
		m.state = "success"
		m.apiResponse = &msg.response
		return m, nil

	case submitScaffoldErrorMsg:
		m.state = "error"
		m.err = msg.err
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ScaffoldModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case "welcome":
		m.state = "selectTemplate"
		m.cursor = 0
		return m, nil

	case "selectTemplate":
		m.template = templates[m.cursor]
		m.state = "inputProjectName"
		m.input.Placeholder = "e.g., my-api"
		return m, nil

	case "inputProjectName":
		name := strings.TrimSpace(m.input.Value())
		if err := ValidateDNSLabel(name); err != nil {
			m.err = err
			return m, nil
		}
		m.projectName = name
		m.err = nil
		m.state = "inputProjectDescription"
		m.input.SetValue("")
		m.input.Placeholder = "Optional description (press Enter to skip)"
		return m, nil

	case "inputProjectDescription":
		desc := strings.TrimSpace(m.input.Value())
		m.projectDescription = desc
		m.state = "inputHTTPPort"
		m.input.SetValue(fmt.Sprintf("%d", m.httpPort))
		m.input.Placeholder = "8080"
		return m, nil

	case "inputHTTPPort":
		portStr := strings.TrimSpace(m.input.Value())
		if portStr == "" {
			portStr = "8080"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			m.err = fmt.Errorf("invalid port number")
			return m, nil
		}
		if err := ValidatePort(port); err != nil {
			m.err = err
			return m, nil
		}
		m.httpPort = port
		m.err = nil
		m.state = "inputEnableGRPC"
		return m, nil

	case "inputGRPCPort":
		portStr := strings.TrimSpace(m.input.Value())
		if portStr == "" {
			portStr = "9090"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			m.err = fmt.Errorf("invalid port number")
			return m, nil
		}
		if err := ValidatePort(port); err != nil {
			m.err = err
			return m, nil
		}
		if port == m.httpPort {
			m.err = fmt.Errorf("gRPC port must differ from HTTP port (%d)", m.httpPort)
			return m, nil
		}
		m.grpcPort = port
		m.err = nil
		m.state = "inputEnableDatabase"
		return m, nil

	case "inputGithubOrg":
		org := strings.TrimSpace(m.input.Value())
		if org == "" {
			m.err = fmt.Errorf("GitHub organization cannot be empty")
			return m, nil
		}
		m.githubOrg = org
		m.err = nil
		m.state = "inputGithubRepo"
		m.input.SetValue(m.projectName) // Default to project name
		m.input.Placeholder = "GitHub repository name"
		return m, nil

	case "inputGithubRepo":
		repo := strings.TrimSpace(m.input.Value())
		if repo == "" {
			m.err = fmt.Errorf("GitHub repository name cannot be empty")
			return m, nil
		}
		m.githubRepo = repo
		m.err = nil
		m.state = "confirmation"
		return m, nil
	}

	return m, nil
}

func (m ScaffoldModel) submitRequest() tea.Cmd {
	return func() tea.Msg {
		// Build GoModulePath
		goModulePath := fmt.Sprintf("github.com/%s/%s", m.githubOrg, m.githubRepo)

		req := ScaffoldRequest{
			Template:           m.template,
			ProjectName:        m.projectName,
			ProjectDescription: m.projectDescription,
			GoModulePath:       goModulePath,
			HTTPPort:           m.httpPort,
			EnableGRPC:         m.enableGRPC,
			EnableDatabase:     m.enableDatabase,
			EnableStorage:      m.enableStorage,
			EnableKeyVault:     m.enableKeyVault,
			GithubOrg:          m.githubOrg,
			GithubRepo:         m.githubRepo,
			RepoPrivate:        true,
		}

		// Only set GRPCPort if gRPC is enabled
		if m.enableGRPC {
			req.GRPCPort = m.grpcPort
		}

		body, err := json.Marshal(req)
		if err != nil {
			return submitScaffoldErrorMsg{err: fmt.Errorf("failed to marshal request: %w", err)}
		}

		client := &http.Client{Timeout: 60 * time.Second} // Extended timeout for Copier + GitHub operations
		httpReq, err := http.NewRequest("POST", m.apiBaseURL+"/api/v1/scaffold", bytes.NewReader(body))
		if err != nil {
			return submitScaffoldErrorMsg{err: fmt.Errorf("failed to create request: %w", err)}
		}

		httpReq.Header.Set("Authorization", "Bearer "+m.authToken)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			return submitScaffoldErrorMsg{err: fmt.Errorf("request failed: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			var errResp map[string]string
			if json.Unmarshal(bodyBytes, &errResp) == nil {
				if msg, ok := errResp["error"]; ok {
					return submitScaffoldErrorMsg{err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)}
				}
			}
			return submitScaffoldErrorMsg{err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))}
		}

		var response ScaffoldResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return submitScaffoldErrorMsg{err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return submitScaffoldSuccessMsg{response: response}
	}
}

func (m ScaffoldModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Create New Service from Template"))
	b.WriteString("\n\n")

	// Show completed fields with checkmarks
	if m.state != "welcome" && m.state != "selectTemplate" && m.template != "" {
		b.WriteString(RenderFieldRow("Template", m.template))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.projectName != "" {
		b.WriteString(RenderFieldRow("Project Name", m.projectName))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.projectDescription != "" {
		b.WriteString(RenderFieldRow("Description", m.projectDescription))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.httpPort > 0 {
		b.WriteString(RenderFieldRow("HTTP Port", fmt.Sprintf("%d", m.httpPort)))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.state != "inputEnableGRPC" && m.state != "inputGRPCPort" {
		grpcStr := "No"
		if m.enableGRPC {
			grpcStr = "Yes"
		}
		b.WriteString(RenderFieldRow("gRPC Enabled", grpcStr))
		b.WriteString("\n")
	}
	if m.enableGRPC && m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.state != "inputEnableGRPC" && m.state != "inputGRPCPort" && m.state != "inputEnableDatabase" && m.grpcPort > 0 {
		b.WriteString(RenderFieldRow("gRPC Port", fmt.Sprintf("%d", m.grpcPort)))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.state != "inputEnableGRPC" && m.state != "inputGRPCPort" && m.state != "inputEnableDatabase" && m.state != "inputEnableStorage" {
		dbStr := "No"
		if m.enableDatabase {
			dbStr = "Yes"
		}
		b.WriteString(RenderFieldRow("Database", dbStr))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.state != "inputEnableGRPC" && m.state != "inputGRPCPort" && m.state != "inputEnableDatabase" && m.state != "inputEnableStorage" && m.state != "inputEnableKeyVault" {
		storageStr := "No"
		if m.enableStorage {
			storageStr = "Yes"
		}
		b.WriteString(RenderFieldRow("Storage", storageStr))
		b.WriteString("\n")
	}
	if m.state != "welcome" && m.state != "selectTemplate" && m.state != "inputProjectName" && m.state != "inputProjectDescription" && m.state != "inputHTTPPort" && m.state != "inputEnableGRPC" && m.state != "inputGRPCPort" && m.state != "inputEnableDatabase" && m.state != "inputEnableStorage" && m.state != "inputEnableKeyVault" && m.state != "inputGithubOrg" {
		vaultStr := "No"
		if m.enableKeyVault {
			vaultStr = "Yes"
		}
		b.WriteString(RenderFieldRow("Key Vault", vaultStr))
		b.WriteString("\n")
	}
	if m.state == "inputGithubRepo" || m.state == "confirmation" || m.state == "submitting" {
		b.WriteString(RenderFieldRow("GitHub Org", m.githubOrg))
		b.WriteString("\n")
	}
	if m.state == "confirmation" || m.state == "submitting" {
		b.WriteString(RenderFieldRow("GitHub Repo", m.githubRepo))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Show current state
	switch m.state {
	case "welcome":
		b.WriteString("This wizard will guide you through creating a new service from a template.\n\n")
		b.WriteString("You'll configure:\n")
		b.WriteString("  • Project metadata (name, description)\n")
		b.WriteString("  • Service ports (HTTP, gRPC)\n")
		b.WriteString("  • Optional features (database, storage, Key Vault)\n")
		b.WriteString("  • GitHub repository settings\n\n")
		b.WriteString(HelpStyle.Render("Press Enter to begin, Ctrl+C to cancel"))

	case "selectTemplate":
		b.WriteString(FieldLabelStyle.Render("Select Template:"))
		b.WriteString("\n")
		for i, tmpl := range templates {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, tmpl))
		}
		b.WriteString(HelpStyle.Render("Use arrow keys, Enter to select"))

	case "inputProjectName":
		b.WriteString(FieldLabelStyle.Render("Project Name:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Lowercase alphanumeric with hyphens, max 63 chars"))

	case "inputProjectDescription":
		b.WriteString(FieldLabelStyle.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Optional description (press Enter to skip)"))

	case "inputHTTPPort":
		b.WriteString(FieldLabelStyle.Render("HTTP Port:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Port number (1024-65535, default: 8080)"))

	case "inputEnableGRPC":
		b.WriteString(FieldLabelStyle.Render("Enable gRPC?"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Y/N"))

	case "inputGRPCPort":
		b.WriteString(FieldLabelStyle.Render("gRPC Port:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Port number (1024-65535, must differ from HTTP port, default: 9090)"))

	case "inputEnableDatabase":
		b.WriteString(FieldLabelStyle.Render("Enable Database Support?"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Y/N"))

	case "inputEnableStorage":
		b.WriteString(FieldLabelStyle.Render("Enable Storage (StorageBucket Claim)?"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Y/N"))

	case "inputEnableKeyVault":
		b.WriteString(FieldLabelStyle.Render("Enable Key Vault (Vault Claim)?"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("Y/N"))

	case "inputGithubOrg":
		b.WriteString(FieldLabelStyle.Render("GitHub Organization/Owner:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("GitHub organization or user (e.g., rodmhgl)"))

	case "inputGithubRepo":
		b.WriteString(FieldLabelStyle.Render("GitHub Repository Name:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(ErrorStyle.Render(m.err.Error()))
			b.WriteString("\n")
		}
		b.WriteString(HelpStyle.Render("Repository name (defaults to project name)"))

	case "confirmation":
		b.WriteString(FieldLabelStyle.Render("Ready to scaffold service?"))
		b.WriteString("\n\n")
		b.WriteString("This will:\n")
		b.WriteString("  1. Execute Copier template\n")
		b.WriteString("  2. Create GitHub repository\n")
		b.WriteString("  3. Push scaffolded code\n")
		b.WriteString("  4. Add apps/" + m.projectName + "/config.json to platform repo\n")
		b.WriteString("  5. Argo CD auto-discovers application within 60s\n\n")
		b.WriteString(HelpStyle.Render("Y to create, N to cancel"))

	case "submitting":
		b.WriteString(RenderSpinner("Scaffolding service (this may take up to 60 seconds)..."))

	case "success":
		if m.apiResponse != nil {
			b.WriteString(SuccessStyle.Render("✓ Service Scaffolded Successfully!"))
			b.WriteString("\n\n")
			b.WriteString("Argo CD will sync this application within 60 seconds.\n\n")
			b.WriteString(FieldLabelStyle.Render("Details:"))
			b.WriteString("\n")
			b.WriteString(RenderFieldRow("Repository", m.apiResponse.RepoURL))
			b.WriteString("\n")
			b.WriteString(RenderFieldRow("Argo CD App", m.apiResponse.ArgoCDAppName))
			b.WriteString("\n")
			b.WriteString(RenderFieldRow("Platform Config", m.apiResponse.PlatformConfigPath))
			b.WriteString("\n\n")
			b.WriteString(FieldLabelStyle.Render("Next Steps:"))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  1. Clone repository: git clone %s\n", m.apiResponse.RepoURL))
			b.WriteString(fmt.Sprintf("  2. Build service:    cd %s && make build\n", m.githubRepo))
			b.WriteString("  3. Run tests:        make test\n")
			b.WriteString(fmt.Sprintf("  4. Verify Argo CD:   rdp apps status %s\n", m.projectName))
			b.WriteString("\n")
			b.WriteString(HelpStyle.Render("Press Q to quit"))
		} else {
			b.WriteString(SuccessStyle.Render("✓ Service Scaffolded Successfully!"))
			b.WriteString("\n\n")
			b.WriteString(HelpStyle.Render("Press Q to quit"))
		}

	case "error":
		if m.err != nil {
			return RenderError("Failed to Scaffold Service", m.err.Error())
		}
		return RenderError("Failed to Scaffold Service", "An unknown error occurred")
	}

	return b.String()
}

// State returns the current state (for external checks)
func (m ScaffoldModel) State() string {
	return m.state
}

// ValidatePort checks if port is in valid range
func ValidatePort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}
	return nil
}
