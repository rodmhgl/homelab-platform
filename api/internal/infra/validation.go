package infra

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// DNS label regex: lowercase alphanumeric + hyphens, max 63 chars
	dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// Allowed Azure regions (mirrors Gatekeeper CrossplaneClaimLocation constraint)
	allowedLocations = map[string]bool{
		"southcentralus": true,
		"eastus2":        true,
	}

	// StorageBucket valid tier values
	validStorageTiers = map[string]bool{
		"Standard": true,
		"Premium":  true,
	}

	// StorageBucket valid redundancy values
	validStorageRedundancy = map[string]bool{
		"LRS":    true,
		"ZRS":    true,
		"GRS":    true,
		"GZRS":   true,
		"RAGRS":  true,
		"RAGZRS": true,
	}

	// Vault valid SKU values
	validVaultSKUs = map[string]bool{
		"standard": true,
		"premium":  true,
	}
)

// validateCreateClaimRequest validates the basic request structure
func validateCreateClaimRequest(req *CreateClaimRequest) error {
	// Validate kind
	if req.Kind != "StorageBucket" && req.Kind != "Vault" {
		return fmt.Errorf("invalid kind: must be 'StorageBucket' or 'Vault', got '%s'", req.Kind)
	}

	// Validate name is DNS label format
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 63 {
		return fmt.Errorf("name must be 63 characters or less, got %d", len(req.Name))
	}
	if !dnsLabelRegex.MatchString(req.Name) {
		return fmt.Errorf("name must be a valid DNS label (lowercase alphanumeric + hyphens, cannot start/end with hyphen)")
	}

	// Validate namespace
	if req.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	// Validate repo fields
	if req.RepoOwner == "" {
		return fmt.Errorf("repoOwner is required")
	}
	if req.RepoName == "" {
		return fmt.Errorf("repoName is required")
	}

	// Validate parameters exist
	if req.Parameters == nil {
		return fmt.Errorf("parameters is required")
	}

	return nil
}

// validateAgainstGatekeeperConstraints validates parameters against Gatekeeper constraints
// This prevents committing Claims that will be rejected by Gatekeeper admission
func validateAgainstGatekeeperConstraints(kind string, params map[string]interface{}) error {
	// Validate location (CrossplaneClaimLocation constraint)
	location, ok := params["location"].(string)
	if !ok || location == "" {
		// Default location is allowed
		location = "southcentralus"
	}
	if !allowedLocations[location] {
		return fmt.Errorf("location '%s' is not allowed (allowed: southcentralus, eastus2)", location)
	}

	// Validate publicAccess (CrossplaneNoPublicAccess constraint)
	if publicAccess, ok := params["publicAccess"].(bool); ok && publicAccess {
		return fmt.Errorf("publicAccess: true is not allowed (enforced by Gatekeeper)")
	}

	// Kind-specific validations
	switch kind {
	case "StorageBucket":
		return validateStorageBucketParams(params)
	case "Vault":
		return validateVaultParams(params)
	}

	return nil
}

// validateStorageBucketParams validates StorageBucket-specific parameters
func validateStorageBucketParams(params map[string]interface{}) error {
	// Validate tier
	if tier, ok := params["tier"].(string); ok {
		if !validStorageTiers[tier] {
			return fmt.Errorf("invalid tier '%s' (allowed: Standard, Premium)", tier)
		}
	}

	// Validate redundancy
	if redundancy, ok := params["redundancy"].(string); ok {
		if !validStorageRedundancy[redundancy] {
			return fmt.Errorf("invalid redundancy '%s' (allowed: LRS, ZRS, GRS, GZRS, RAGRS, RAGZRS)", redundancy)
		}
	}

	return nil
}

// validateVaultParams validates Vault-specific parameters
func validateVaultParams(params map[string]interface{}) error {
	// Validate skuName
	if skuName, ok := params["skuName"].(string); ok {
		if !validVaultSKUs[skuName] {
			return fmt.Errorf("invalid skuName '%s' (allowed: standard, premium)", skuName)
		}
	}

	// Validate softDeleteRetentionDays
	if days, ok := params["softDeleteRetentionDays"].(float64); ok {
		if days < 7 || days > 90 {
			return fmt.Errorf("softDeleteRetentionDays must be between 7 and 90, got %.0f", days)
		}
	}

	return nil
}

// buildCommitMessage generates a descriptive commit message for the Claim
func buildCommitMessage(req *CreateClaimRequest) string {
	location := "southcentralus"
	if loc, ok := req.Parameters["location"].(string); ok {
		location = loc
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Add %s Claim: %s\n\n", req.Kind, req.Name))
	sb.WriteString(fmt.Sprintf("Provisions %s infrastructure for %s namespace.\n\n", req.Kind, req.Namespace))
	sb.WriteString(fmt.Sprintf("Location: %s\n", location))
	sb.WriteString("Managed by: Platform API\n")

	return sb.String()
}
