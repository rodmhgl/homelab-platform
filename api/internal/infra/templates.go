package infra

import (
	"bytes"
	"fmt"
	"text/template"
)

// generateClaimYAML generates YAML content for a Claim based on its kind
func (h *Handler) generateClaimYAML(req *CreateClaimRequest) (string, error) {
	switch req.Kind {
	case "StorageBucket":
		return generateStorageBucketYAML(req)
	case "Vault":
		return generateVaultYAML(req)
	default:
		return "", fmt.Errorf("unsupported kind: %s", req.Kind)
	}
}

// StorageBucketTemplateData holds template data for StorageBucket Claims
type StorageBucketTemplateData struct {
	Name             string
	Namespace        string
	AppName          string
	Location         string
	Tier             string
	Redundancy       string
	EnableVersioning bool
	CustomLabels     map[string]string
}

// VaultTemplateData holds template data for Vault Claims
type VaultTemplateData struct {
	Name                    string
	Namespace               string
	AppName                 string
	Location                string
	SKUName                 string
	SoftDeleteRetentionDays int
	CustomLabels            map[string]string
}

// generateStorageBucketYAML generates YAML for a StorageBucket Claim
func generateStorageBucketYAML(req *CreateClaimRequest) (string, error) {
	// Extract parameters with defaults
	location := getStringParam(req.Parameters, "location", "southcentralus")
	tier := getStringParam(req.Parameters, "tier", "Standard")
	redundancy := getStringParam(req.Parameters, "redundancy", "LRS")
	enableVersioning := getBoolParam(req.Parameters, "enableVersioning", false)

	// Determine app name (infer from repo name if not in labels)
	appName := req.RepoName
	if name, ok := req.Labels["app.kubernetes.io/name"]; ok {
		appName = name
	}

	// Merge labels
	labels := mergeLabels(req.Labels, map[string]string{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/instance":   req.Name,
		"app.kubernetes.io/version":    "1.0.0",
		"app.kubernetes.io/component":  "infrastructure",
		"app.kubernetes.io/part-of":    appName,
		"app.kubernetes.io/managed-by": "crossplane",
	})

	data := StorageBucketTemplateData{
		Name:             req.Name,
		Namespace:        req.Namespace,
		AppName:          appName,
		Location:         location,
		Tier:             tier,
		Redundancy:       redundancy,
		EnableVersioning: enableVersioning,
		CustomLabels:     labels,
	}

	tmpl := `apiVersion: platform.example.com/v1alpha1
kind: StorageBucket
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
{{- range $key, $value := .CustomLabels }}
    {{ $key }}: {{ $value }}
{{- end }}
spec:
  parameters:
    location: {{ .Location }}
    tier: {{ .Tier }}
    redundancy: {{ .Redundancy }}
    enableVersioning: {{ .EnableVersioning }}
    publicAccess: false  # Enforced by Gatekeeper CrossplaneNoPublicAccess constraint
  writeConnectionSecretToRef:
    name: {{ .Name }}
    namespace: {{ .Namespace }}
  compositionSelector:
    matchLabels:
      provider: azure
      type: storage
`

	return renderTemplate("storagebucket", tmpl, data)
}

// generateVaultYAML generates YAML for a Vault Claim
func generateVaultYAML(req *CreateClaimRequest) (string, error) {
	// Extract parameters with defaults
	location := getStringParam(req.Parameters, "location", "southcentralus")
	skuName := getStringParam(req.Parameters, "skuName", "standard")
	softDeleteRetentionDays := getIntParam(req.Parameters, "softDeleteRetentionDays", 7)

	// Determine app name (infer from repo name if not in labels)
	appName := req.RepoName
	if name, ok := req.Labels["app.kubernetes.io/name"]; ok {
		appName = name
	}

	// Merge labels
	labels := mergeLabels(req.Labels, map[string]string{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/instance":   req.Name,
		"app.kubernetes.io/version":    "1.0.0",
		"app.kubernetes.io/component":  "infrastructure",
		"app.kubernetes.io/part-of":    appName,
		"app.kubernetes.io/managed-by": "crossplane",
	})

	data := VaultTemplateData{
		Name:                    req.Name,
		Namespace:               req.Namespace,
		AppName:                 appName,
		Location:                location,
		SKUName:                 skuName,
		SoftDeleteRetentionDays: softDeleteRetentionDays,
		CustomLabels:            labels,
	}

	tmpl := `apiVersion: platform.example.com/v1alpha1
kind: Vault
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
{{- range $key, $value := .CustomLabels }}
    {{ $key }}: {{ $value }}
{{- end }}
spec:
  parameters:
    location: {{ .Location }}
    skuName: {{ .SKUName }}
    publicAccess: false  # Enforced by Gatekeeper CrossplaneNoPublicAccess constraint
    softDeleteRetentionDays: {{ .SoftDeleteRetentionDays }}
  writeConnectionSecretToRef:
    name: {{ .Name }}
    namespace: {{ .Namespace }}
  compositionSelector:
    matchLabels:
      provider: azure
      type: keyvault
`

	return renderTemplate("vault", tmpl, data)
}

// Helper functions

// getStringParam extracts a string parameter with a default value
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

// getBoolParam extracts a bool parameter with a default value
func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultValue
}

// getIntParam extracts an int parameter with a default value
func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	// JSON numbers are float64
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

// mergeLabels merges custom labels with required labels (required labels take precedence)
func mergeLabels(custom, required map[string]string) map[string]string {
	merged := make(map[string]string)

	// Start with custom labels
	for k, v := range custom {
		merged[k] = v
	}

	// Override with required labels
	for k, v := range required {
		merged[k] = v
	}

	return merged
}

// renderTemplate renders a Go text template with the provided data
func renderTemplate(name, tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
