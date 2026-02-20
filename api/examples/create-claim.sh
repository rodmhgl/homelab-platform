#!/bin/bash
# Example script to test POST /api/v1/infra endpoint

# Set your API URL and token
API_URL="${API_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-your-token-here}"

echo "Testing POST /api/v1/infra - Create StorageBucket Claim"
echo "=========================================="

# Example 1: Create a StorageBucket Claim
curl -X POST "${API_URL}/api/v1/infra" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "demo-app-storage",
    "namespace": "demo-app",
    "repoOwner": "rodmhgl",
    "repoName": "demo-app",
    "parameters": {
      "location": "southcentralus",
      "tier": "Standard",
      "redundancy": "LRS",
      "enableVersioning": false
    },
    "labels": {
      "app.kubernetes.io/name": "demo-app"
    }
  }' | jq .

echo ""
echo "=========================================="
echo "Testing POST /api/v1/infra - Create Vault Claim"
echo "=========================================="

# Example 2: Create a Vault Claim
curl -X POST "${API_URL}/api/v1/infra" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "Vault",
    "name": "demo-app-vault",
    "namespace": "demo-app",
    "repoOwner": "rodmhgl",
    "repoName": "demo-app",
    "parameters": {
      "location": "eastus2",
      "skuName": "standard",
      "softDeleteRetentionDays": 7
    }
  }' | jq .

echo ""
echo "=========================================="
echo "Testing validation - Invalid location"
echo "=========================================="

# Example 3: Test validation failure (invalid location)
curl -X POST "${API_URL}/api/v1/infra" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "test-storage",
    "namespace": "default",
    "repoOwner": "rodmhgl",
    "repoName": "test-app",
    "parameters": {
      "location": "westeurope"
    }
  }' | jq .

echo ""
echo "=========================================="
echo "Testing validation - Public access blocked"
echo "=========================================="

# Example 4: Test validation failure (publicAccess: true)
curl -X POST "${API_URL}/api/v1/infra" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "test-storage",
    "namespace": "default",
    "repoOwner": "rodmhgl",
    "repoName": "test-app",
    "parameters": {
      "location": "southcentralus",
      "publicAccess": true
    }
  }' | jq .
