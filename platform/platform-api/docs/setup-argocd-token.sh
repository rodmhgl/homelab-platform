#!/bin/bash
#
# setup-argocd-token.sh
#
# Generates an Argo CD API token and stores it in Azure Key Vault.
# This is a ONE-TIME bootstrap step — the service account and RBAC are managed via GitOps (values.yaml).
#
# Prerequisites:
# - kubectl configured with cluster access
# - argocd CLI installed (https://argo-cd.readthedocs.io/en/stable/cli_installation/)
# - az CLI installed and authenticated
# - Argo CD deployed with service account 'platform-api' (defined in values.yaml)
#
# Usage:
#   ./setup-argocd-token.sh [--kv-name <keyvault-name>]

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Parse command-line arguments
KV_NAME=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --kv-name)
            KV_NAME="$2"
            shift 2
            ;;
        *)
            error "Unknown argument: $1"
            echo "Usage: $0 [--kv-name <keyvault-name>]"
            exit 1
            ;;
    esac
done

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."

    local missing_tools=()

    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi

    if ! command -v argocd &> /dev/null; then
        missing_tools+=("argocd")
    fi

    if ! command -v az &> /dev/null; then
        missing_tools+=("az")
    fi

    if [ ${#missing_tools[@]} -gt 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        echo ""
        echo "Install missing tools:"
        for tool in "${missing_tools[@]}"; do
            case $tool in
                kubectl)
                    echo "  kubectl: https://kubernetes.io/docs/tasks/tools/"
                    ;;
                argocd)
                    echo "  argocd: https://argo-cd.readthedocs.io/en/stable/cli_installation/"
                    ;;
                az)
                    echo "  az: https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"
                    ;;
            esac
        done
        exit 1
    fi

    # Check kubectl access
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        echo "Please configure kubectl with cluster credentials:"
        echo "  az aks get-credentials --resource-group <rg> --name <cluster>"
        exit 1
    fi

    # Check az login
    if ! az account show &> /dev/null; then
        error "Not logged in to Azure CLI"
        echo "Please run 'az login' and try again."
        exit 1
    fi

    # Check Argo CD is deployed
    if ! kubectl get namespace argocd &> /dev/null; then
        error "Argo CD namespace not found"
        echo "Please deploy Argo CD via GitOps before running this script."
        exit 1
    fi

    # Check service account exists
    info "Verifying Argo CD service account 'platform-api' exists..."
    if ! kubectl get configmap argocd-cm -n argocd -o yaml | grep -q "accounts.platform-api"; then
        error "Service account 'platform-api' not found in argocd-cm ConfigMap"
        echo ""
        echo "The service account should be defined in platform/argocd/values.yaml:"
        echo "  configs:"
        echo "    cm:"
        echo "      accounts.platform-api: apiKey"
        echo ""
        echo "Deploy/update Argo CD via GitOps and try again."
        exit 1
    fi

    success "Service account 'platform-api' found"
    success "All prerequisites met"
}

# Generate Argo CD token
generate_argocd_token() {
    info "Generating Argo CD token for 'platform-api' account..."

    # Get Argo CD admin password
    local admin_password
    if ! admin_password=$(kubectl -n argocd get secret argocd-initial-admin-secret \
        -o jsonpath='{.data.password}' 2>/dev/null | base64 -d); then
        error "Cannot retrieve Argo CD admin password"
        echo "The secret 'argocd-initial-admin-secret' may have been deleted."
        echo "Please reset the admin password or use an existing admin account."
        exit 1
    fi

    # Start port-forward in background
    info "Starting port-forward to Argo CD server..."
    kubectl port-forward svc/argocd-server -n argocd 8080:443 > /dev/null 2>&1 &
    local pf_pid=$!

    # Cleanup function to kill port-forward on exit
    trap "kill $pf_pid 2>/dev/null || true" EXIT

    # Wait for port-forward to be ready
    info "Waiting for port-forward to be ready..."
    local ready=false
    for i in {1..10}; do
        if curl -k -s https://localhost:8080/healthz > /dev/null 2>&1; then
            ready=true
            break
        fi
        sleep 1
    done

    if ! $ready; then
        error "Port-forward failed to become ready"
        echo "Please check Argo CD server status: kubectl get pods -n argocd"
        exit 1
    fi

    # Login to Argo CD
    info "Logging in to Argo CD CLI..."
    if ! argocd login localhost:8080 \
        --username admin \
        --password "$admin_password" \
        --insecure > /dev/null 2>&1; then

        error "Failed to login to Argo CD"
        echo "Please check Argo CD server status: kubectl get pods -n argocd"
        exit 1
    fi

    # Generate token (90 day expiry by default)
    info "Generating token for platform-api account (90 day expiry)..."
    local token
    if ! token=$(argocd account generate-token --account platform-api 2>&1); then
        error "Token generation failed"
        echo "$token"
        echo ""
        echo "Common causes:"
        echo "  - Service account not yet created in Argo CD (check values.yaml)"
        echo "  - Argo CD server not fully started (check pod status)"
        exit 1
    fi

    if [ -z "$token" ]; then
        error "Token generation returned empty result"
        exit 1
    fi

    success "Token generated successfully"

    # Return token
    echo "$token"
}

# Store token in Key Vault
store_token_in_keyvault() {
    local token=$1

    info "Storing token in Azure Key Vault..."

    # Determine Key Vault name
    if [ -z "$KV_NAME" ]; then
        # Try to get from Terraform outputs
        if [ -f "../../../infra/terraform.tfstate" ]; then
            warn "Using local Terraform state (not recommended for TFC workspaces)"
            if KV_NAME=$(cd ../../../infra && terraform output -raw keyvault_name 2>/dev/null); then
                info "Key Vault name from Terraform: $KV_NAME"
            fi
        fi

        # If still empty, prompt user
        if [ -z "$KV_NAME" ]; then
            warn "Cannot auto-detect Key Vault name from Terraform"
            echo -n "Please enter the Key Vault name (from Terraform output 'keyvault_name'): "
            read -r KV_NAME
        fi
    fi

    if [ -z "$KV_NAME" ]; then
        error "Key Vault name is required"
        echo "Usage: $0 --kv-name <keyvault-name>"
        exit 1
    fi

    info "Using Key Vault: $KV_NAME"

    # Store token
    if ! az keyvault secret set \
        --vault-name "$KV_NAME" \
        --name argocd-token \
        --value "$token" \
        --output none; then

        error "Failed to store token in Key Vault"
        echo ""
        echo "Common causes:"
        echo "  - Insufficient permissions (need 'Key Vault Secrets Officer' role)"
        echo "  - Key Vault doesn't exist"
        echo "  - Key Vault is behind a firewall/private endpoint"
        echo ""
        echo "Manual command:"
        echo "  az keyvault secret set --vault-name $KV_NAME --name argocd-token --value '<token>'"
        exit 1
    fi

    success "Token stored in Key Vault as 'argocd-token'"
}

# Verify ExternalSecret sync
verify_externalsecret_sync() {
    info "Verifying ExternalSecret sync (optional)..."

    # Check if ExternalSecret exists
    if ! kubectl get externalsecret platform-api-secrets -n platform &> /dev/null 2>&1; then
        warn "ExternalSecret 'platform-api-secrets' not found"
        echo "The ExternalSecret will be created when Platform API is deployed via Argo CD."
        echo "No further action needed — token will be synced automatically."
        return
    fi

    # Force sync
    info "Forcing ExternalSecret sync..."
    kubectl annotate externalsecret platform-api-secrets -n platform \
        force-sync="$(date +%s)" --overwrite > /dev/null 2>&1

    # Wait for sync (max 30 seconds)
    info "Waiting for sync to complete (max 30s)..."
    local count=0
    while [ $count -lt 30 ]; do
        local status
        status=$(kubectl get externalsecret platform-api-secrets -n platform \
            -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")

        if [ "$status" == "True" ]; then
            success "ExternalSecret synced successfully"
            break
        fi

        sleep 1
        ((count++))
    done

    if [ $count -eq 30 ]; then
        warn "ExternalSecret sync timed out"
        echo "Check status: kubectl describe externalsecret platform-api-secrets -n platform"
        echo "ESO will retry automatically — no action needed."
        return
    fi

    # Verify secret keys
    info "Verifying Kubernetes Secret contains ARGOCD_TOKEN..."
    if kubectl get secret platform-api-secrets -n platform -o jsonpath='{.data.ARGOCD_TOKEN}' &> /dev/null; then
        success "ARGOCD_TOKEN present in Kubernetes Secret"
    else
        warn "ARGOCD_TOKEN not yet in Secret (may be syncing)"
        echo "Wait a few moments and check: kubectl get secret platform-api-secrets -n platform -o jsonpath='{.data}' | jq 'keys'"
    fi
}

# Main execution
main() {
    echo "=========================================="
    echo "  Argo CD Token Bootstrap"
    echo "=========================================="
    echo ""
    echo "This script generates an Argo CD API token for the 'platform-api'"
    echo "service account and stores it in Azure Key Vault."
    echo ""
    echo "Note: Service account and RBAC are managed via GitOps (values.yaml)"
    echo "      This script only handles the imperative token generation step."
    echo ""

    check_prerequisites
    echo ""

    local token
    token=$(generate_argocd_token)
    echo ""

    store_token_in_keyvault "$token"
    echo ""

    verify_externalsecret_sync
    echo ""

    success "Argo CD token bootstrap complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Wait for Platform API to deploy (or restart if already running):"
    echo "     kubectl rollout status deployment platform-api -n platform"
    echo ""
    echo "  2. Verify Platform API can access Argo CD:"
    echo "     kubectl logs -n platform -l app.kubernetes.io/name=platform-api --tail=50"
    echo ""
    echo "  3. Test the /api/v1/apps endpoint:"
    echo "     kubectl port-forward svc/platform-api -n platform 8080:8080"
    echo "     curl -H 'Authorization: Bearer test' http://localhost:8080/api/v1/apps | jq"
    echo ""
    echo "Token rotation reminder:"
    echo "  Tokens expire after 90 days. Set a calendar reminder to regenerate:"
    echo "  ./setup-argocd-token.sh --kv-name $KV_NAME"
    echo ""
}

# Run main function
main
