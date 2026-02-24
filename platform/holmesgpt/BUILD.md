# Building HolmesGPT Docker Image

Since no official public Docker image exists for HolmesGPT, you must build from source and push to the homelab ACR.

## Prerequisites

1. **Docker with buildx** (multi-arch support)
2. **Azure CLI** (authenticated to subscription)
3. **Git** (to clone HolmesGPT repository)

## Quick Build

```bash
# 1. Clone HolmesGPT repository
git clone https://github.com/robusta-dev/holmesgpt.git
cd holmesgpt

# 2. Authenticate to ACR
az acr login --name homelabplatformacr

# 3. Build and push multi-arch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:v1.0.0 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:latest \
  --push \
  .

# 4. Verify
docker pull homelabplatformacr.azurecr.io/holmesgpt:v1.0.0
docker inspect homelabplatformacr.azurecr.io/holmesgpt:v1.0.0
```

## Build Details

**Base image:** `python:3.11-slim-bookworm`
**Build time:** ~5-10 minutes (depends on network speed for Python dependencies)
**Image size:** ~500MB compressed

**Included tools:**
- Python 3.11 + HolmesGPT application
- kubectl (latest stable)
- argocd CLI v3.2.0
- kube-lineage v2.2.4 (dependency graph tool)

## Version Tagging

**Current convention:**
- `v1.0.0` — Semantic version for stable releases
- `latest` — Always points to most recent build

**Updating versions:**
```bash
# For new releases, increment version
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:v1.1.0 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:latest \
  --push \
  .

# Update deployment.yaml image reference
# platform/holmesgpt/base/deployment.yaml
# Change: image: homelabplatformacr.azurecr.io/holmesgpt:v1.0.0
# To: image: homelabplatformacr.azurecr.io/holmesgpt:v1.1.0
```

## Troubleshooting Builds

### Build fails with "platform not supported"

**Solution:** Enable Docker buildx multi-platform builds

```bash
# Create new builder
docker buildx create --name multiarch --use

# Verify platforms
docker buildx inspect --bootstrap
# Should show: linux/amd64, linux/arm64
```

### ACR authentication fails

**Solution:** Re-authenticate with proper permissions

```bash
# Check current subscription
az account show

# Switch if needed
az account set --subscription <subscription-id>

# Re-authenticate to ACR
az acr login --name homelabplatformacr
```

### Push fails with "denied: requested access to the resource is denied"

**Solution:** Verify ACR permissions

```bash
# Check if you have AcrPush role
az role assignment list \
  --assignee <your-user-principal-id> \
  --scope /subscriptions/<sub-id>/resourceGroups/rg-homelab-aks-dev/providers/Microsoft.ContainerRegistry/registries/homelabplatformacr

# If missing, add role (requires Owner/Contributor)
az role assignment create \
  --assignee <your-user-principal-id> \
  --role AcrPush \
  --scope /subscriptions/<sub-id>/resourceGroups/rg-homelab-aks-dev/providers/Microsoft.ContainerRegistry/registries/homelabplatformacr
```

## Alternative: Single-Arch Build

If multi-arch build fails, build for your cluster's architecture only:

```bash
# For AMD64 (most AKS clusters)
docker build \
  --tag homelabplatformacr.azurecr.io/holmesgpt:v1.0.0 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:latest \
  .

docker push homelabplatformacr.azurecr.io/holmesgpt:v1.0.0
docker push homelabplatformacr.azurecr.io/holmesgpt:latest
```

## Updating HolmesGPT

To pull latest changes from upstream:

```bash
cd holmesgpt
git pull origin master

# Rebuild with incremented version
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:v1.1.0 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:latest \
  --push \
  .

# Update Kubernetes deployment
kubectl set image deployment/holmesgpt \
  -n holmesgpt \
  holmesgpt=homelabplatformacr.azurecr.io/holmesgpt:v1.1.0
```
