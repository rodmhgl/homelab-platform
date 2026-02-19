#!/usr/bin/env bash
## Bootstrap Argo CD onto an empty AKS cluster.
## Run once — Argo CD self-manages via GitOps after this.
##
## Prerequisites:
##   - kubectl context pointed at the target AKS cluster
##   - helm 3 installed
##   - REPO_URL set to this repo's HTTPS clone URL
##
## Usage:
##   REPO_URL=https://github.com/rodmhgl/homelab-platform ./bootstrap.sh

set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/rodmhgl/homelab-platform}"
ARGOCD_NAMESPACE="argocd"
ARGOCD_CHART_VERSION="7.9.1"

echo "==> Adding argo Helm repo"
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

echo "==> Creating argocd namespace"
kubectl create namespace "${ARGOCD_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Installing Argo CD via Helm"
helm upgrade --install argocd argo/argo-cd \
--namespace "${ARGOCD_NAMESPACE}" \
--version "${ARGOCD_CHART_VERSION}" \
--values platform/argocd/values.yaml \
--wait

echo "==> Waiting for Argo CD server to be ready"
kubectl rollout status deployment/argocd-server -n "${ARGOCD_NAMESPACE}" --timeout=120s

echo "==> Applying Projects (platform + workloads)"
kubectl apply -f platform/argocd/projects.yaml

echo "==> Applying root App-of-Apps (platform-root)"
kubectl apply -n "${ARGOCD_NAMESPACE}" -f platform/argocd/root-app.yaml

echo ""
echo "Bootstrap complete. Argo CD is now managing the platform."
echo ""
echo "  Port-forward:  kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "  Initial admin: kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath='{.data.password}' | base64 -d"
