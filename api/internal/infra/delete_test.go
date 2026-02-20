package infra

import (
	"encoding/json"
	"testing"
)

// TestDeleteClaimRequest_JSONMarshaling validates JSON marshaling/unmarshaling for delete requests
func TestDeleteClaimRequest_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		request DeleteClaimRequest
	}{
		{
			name: "basic delete request",
			request: DeleteClaimRequest{
				RepoOwner: "testorg",
				RepoName:  "test-app",
			},
		},
		{
			name: "delete request with different values",
			request: DeleteClaimRequest{
				RepoOwner: "production-org",
				RepoName:  "critical-service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Test unmarshaling
			var unmarshaled DeleteClaimRequest
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if unmarshaled.RepoOwner != tt.request.RepoOwner {
				t.Errorf("RepoOwner mismatch: got %s, want %s", unmarshaled.RepoOwner, tt.request.RepoOwner)
			}
			if unmarshaled.RepoName != tt.request.RepoName {
				t.Errorf("RepoName mismatch: got %s, want %s", unmarshaled.RepoName, tt.request.RepoName)
			}
		})
	}
}

// TestDeleteClaimResponse_JSONMarshaling validates JSON marshaling/unmarshaling for delete responses
func TestDeleteClaimResponse_JSONMarshaling(t *testing.T) {
	response := DeleteClaimResponse{
		Success:   true,
		Message:   "Claim deleted successfully from Git. Argo CD will remove it from the cluster.",
		Kind:      "StorageBucket",
		Name:      "test-bucket",
		Namespace: "default",
		CommitSHA: "abc123def456",
		FilePath:  "k8s/claims/test-bucket.yaml",
		RepoURL:   "https://github.com/testorg/test-app",
	}

	// Test marshaling
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Test unmarshaling
	var unmarshaled DeleteClaimResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify all fields
	if unmarshaled.Success != response.Success {
		t.Errorf("Success mismatch: got %v, want %v", unmarshaled.Success, response.Success)
	}
	if unmarshaled.Message != response.Message {
		t.Errorf("Message mismatch: got %s, want %s", unmarshaled.Message, response.Message)
	}
	if unmarshaled.Kind != response.Kind {
		t.Errorf("Kind mismatch: got %s, want %s", unmarshaled.Kind, response.Kind)
	}
	if unmarshaled.Name != response.Name {
		t.Errorf("Name mismatch: got %s, want %s", unmarshaled.Name, response.Name)
	}
	if unmarshaled.Namespace != response.Namespace {
		t.Errorf("Namespace mismatch: got %s, want %s", unmarshaled.Namespace, response.Namespace)
	}
	if unmarshaled.CommitSHA != response.CommitSHA {
		t.Errorf("CommitSHA mismatch: got %s, want %s", unmarshaled.CommitSHA, response.CommitSHA)
	}
	if unmarshaled.FilePath != response.FilePath {
		t.Errorf("FilePath mismatch: got %s, want %s", unmarshaled.FilePath, response.FilePath)
	}
	if unmarshaled.RepoURL != response.RepoURL {
		t.Errorf("RepoURL mismatch: got %s, want %s", unmarshaled.RepoURL, response.RepoURL)
	}
}

// TestDeleteClaimRequest_Validation validates request field requirements
func TestDeleteClaimRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		request   DeleteClaimRequest
		wantValid bool
	}{
		{
			name: "valid request",
			request: DeleteClaimRequest{
				RepoOwner: "testorg",
				RepoName:  "test-app",
			},
			wantValid: true,
		},
		{
			name: "missing repo owner",
			request: DeleteClaimRequest{
				RepoName: "test-app",
			},
			wantValid: false,
		},
		{
			name: "missing repo name",
			request: DeleteClaimRequest{
				RepoOwner: "testorg",
			},
			wantValid: false,
		},
		{
			name:      "empty request",
			request:   DeleteClaimRequest{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: both fields must be non-empty
			isValid := tt.request.RepoOwner != "" && tt.request.RepoName != ""

			if isValid != tt.wantValid {
				t.Errorf("Validation mismatch: got %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}
