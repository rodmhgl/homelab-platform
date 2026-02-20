package compliance

// SummaryResponse returns overall compliance metrics
type SummaryResponse struct {
	ComplianceScore           float64        `json:"complianceScore"` // Percentage (0-100)
	TotalViolations           int            `json:"totalViolations"`
	TotalVulnerabilities      int            `json:"totalVulnerabilities"`
	ViolationsBySeverity      map[string]int `json:"violationsBySeverity"`      // policy, config, security
	VulnerabilitiesBySeverity map[string]int `json:"vulnerabilitiesBySeverity"` // CRITICAL, HIGH, MEDIUM, LOW
}

// PoliciesResponse returns list of active policies
type PoliciesResponse struct {
	Policies []Policy `json:"policies"`
}

// Policy represents a Gatekeeper ConstraintTemplate
type Policy struct {
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"` // ConstraintTemplate kind
	Description string                 `json:"description"`
	Scope       []string               `json:"scope"` // Namespaces or cluster-wide
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ViolationsResponse returns Gatekeeper audit violations
type ViolationsResponse struct {
	Violations []Violation `json:"violations"`
}

// Violation represents a single Gatekeeper policy violation
type Violation struct {
	ConstraintName string `json:"constraintName"`
	ConstraintKind string `json:"constraintKind"`
	Resource       string `json:"resource"` // namespace/kind/name
	Namespace      string `json:"namespace"`
	Message        string `json:"message"`
	Timestamp      string `json:"timestamp,omitempty"`
}

// VulnerabilitiesResponse returns Trivy scan results
type VulnerabilitiesResponse struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// Vulnerability represents a CVE found by Trivy Operator
type Vulnerability struct {
	Image           string  `json:"image"`
	Namespace       string  `json:"namespace"`
	Workload        string  `json:"workload"`
	CVEID           string  `json:"cveId"`
	Severity        string  `json:"severity"`
	Score           float64 `json:"score,omitempty"`
	AffectedPackage string  `json:"affectedPackage"`
	FixedVersion    string  `json:"fixedVersion,omitempty"`
	PrimaryLink     string  `json:"primaryLink,omitempty"`
}

// EventsResponse returns security events (Falco)
type EventsResponse struct {
	Events []SecurityEvent `json:"events"`
}

// SecurityEvent represents a Falco security event
type SecurityEvent struct {
	Timestamp string `json:"timestamp"`
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Resource  string `json:"resource,omitempty"`
}
