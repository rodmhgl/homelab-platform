package infra

import (
	"testing"
)

func TestValidateCreateClaimRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateClaimRequest
		wantErr bool
	}{
		{
			name: "valid StorageBucket request",
			req: &CreateClaimRequest{
				Kind:      "StorageBucket",
				Name:      "demo-storage",
				Namespace: "demo-app",
				RepoOwner: "rodmhgl",
				RepoName:  "demo-app",
				Parameters: map[string]interface{}{
					"location": "southcentralus",
				},
			},
			wantErr: false,
		},
		{
			name: "valid Vault request",
			req: &CreateClaimRequest{
				Kind:      "Vault",
				Name:      "demo-vault",
				Namespace: "demo-app",
				RepoOwner: "rodmhgl",
				RepoName:  "demo-app",
				Parameters: map[string]interface{}{
					"location": "eastus2",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid kind",
			req: &CreateClaimRequest{
				Kind:      "InvalidKind",
				Name:      "test",
				Namespace: "default",
				RepoOwner: "user",
				RepoName:  "repo",
				Parameters: map[string]interface{}{
					"location": "southcentralus",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid name - uppercase",
			req: &CreateClaimRequest{
				Kind:      "StorageBucket",
				Name:      "Demo-Storage",
				Namespace: "default",
				RepoOwner: "user",
				RepoName:  "repo",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "invalid name - too long",
			req: &CreateClaimRequest{
				Kind:      "StorageBucket",
				Name:      "this-is-a-very-long-name-that-exceeds-the-maximum-allowed-length-for-dns-labels",
				Namespace: "default",
				RepoOwner: "user",
				RepoName:  "repo",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "missing namespace",
			req: &CreateClaimRequest{
				Kind:      "StorageBucket",
				Name:      "test",
				RepoOwner: "user",
				RepoName:  "repo",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "missing repoOwner",
			req: &CreateClaimRequest{
				Kind:      "StorageBucket",
				Name:      "test",
				Namespace: "default",
				RepoName:  "repo",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateClaimRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreateClaimRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAgainstGatekeeperConstraints(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid location - southcentralus",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location": "southcentralus",
			},
			wantErr: false,
		},
		{
			name: "valid location - eastus2",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location": "eastus2",
			},
			wantErr: false,
		},
		{
			name: "invalid location",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location": "westeurope",
			},
			wantErr: true,
		},
		{
			name: "publicAccess true - blocked",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location":     "southcentralus",
				"publicAccess": true,
			},
			wantErr: true,
		},
		{
			name: "publicAccess false - allowed",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location":     "southcentralus",
				"publicAccess": false,
			},
			wantErr: false,
		},
		{
			name: "valid StorageBucket tier",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location": "southcentralus",
				"tier":     "Standard",
			},
			wantErr: false,
		},
		{
			name: "invalid StorageBucket tier",
			kind: "StorageBucket",
			params: map[string]interface{}{
				"location": "southcentralus",
				"tier":     "Invalid",
			},
			wantErr: true,
		},
		{
			name: "valid Vault skuName",
			kind: "Vault",
			params: map[string]interface{}{
				"location": "southcentralus",
				"skuName":  "premium",
			},
			wantErr: false,
		},
		{
			name: "invalid Vault skuName",
			kind: "Vault",
			params: map[string]interface{}{
				"location": "southcentralus",
				"skuName":  "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid Vault retention days",
			kind: "Vault",
			params: map[string]interface{}{
				"location":                "southcentralus",
				"softDeleteRetentionDays": float64(30),
			},
			wantErr: false,
		},
		{
			name: "invalid Vault retention days - too low",
			kind: "Vault",
			params: map[string]interface{}{
				"location":                "southcentralus",
				"softDeleteRetentionDays": float64(5),
			},
			wantErr: true,
		},
		{
			name: "invalid Vault retention days - too high",
			kind: "Vault",
			params: map[string]interface{}{
				"location":                "southcentralus",
				"softDeleteRetentionDays": float64(100),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgainstGatekeeperConstraints(tt.kind, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAgainstGatekeeperConstraints() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateStorageBucketYAML(t *testing.T) {
	req := &CreateClaimRequest{
		Kind:      "StorageBucket",
		Name:      "demo-storage",
		Namespace: "demo-app",
		RepoOwner: "rodmhgl",
		RepoName:  "demo-app",
		Parameters: map[string]interface{}{
			"location":   "southcentralus",
			"tier":       "Standard",
			"redundancy": "LRS",
		},
		Labels: map[string]string{
			"app.kubernetes.io/name": "demo-app",
		},
	}

	yaml, err := generateStorageBucketYAML(req)
	if err != nil {
		t.Fatalf("generateStorageBucketYAML() error = %v", err)
	}

	// Basic checks
	if yaml == "" {
		t.Error("generateStorageBucketYAML() returned empty YAML")
	}

	// Check for required fields
	requiredStrings := []string{
		"apiVersion: platform.example.com/v1alpha1",
		"kind: StorageBucket",
		"name: demo-storage",
		"namespace: demo-app",
		"location: southcentralus",
		"tier: Standard",
		"redundancy: LRS",
		"publicAccess: false",
		"provider: azure",
		"type: storage",
	}

	for _, s := range requiredStrings {
		if !contains(yaml, s) {
			t.Errorf("generateStorageBucketYAML() missing expected string: %s", s)
		}
	}
}

func TestGenerateVaultYAML(t *testing.T) {
	req := &CreateClaimRequest{
		Kind:      "Vault",
		Name:      "demo-vault",
		Namespace: "demo-app",
		RepoOwner: "rodmhgl",
		RepoName:  "demo-app",
		Parameters: map[string]interface{}{
			"location":                "eastus2",
			"skuName":                 "premium",
			"softDeleteRetentionDays": float64(30),
		},
	}

	yaml, err := generateVaultYAML(req)
	if err != nil {
		t.Fatalf("generateVaultYAML() error = %v", err)
	}

	// Basic checks
	if yaml == "" {
		t.Error("generateVaultYAML() returned empty YAML")
	}

	// Check for required fields
	requiredStrings := []string{
		"apiVersion: platform.example.com/v1alpha1",
		"kind: Vault",
		"name: demo-vault",
		"namespace: demo-app",
		"location: eastus2",
		"skuName: premium",
		"softDeleteRetentionDays: 30",
		"publicAccess: false",
		"provider: azure",
		"type: keyvault",
	}

	for _, s := range requiredStrings {
		if !contains(yaml, s) {
			t.Errorf("generateVaultYAML() missing expected string: %s", s)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsRec(s, substr))
}

func containsRec(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
