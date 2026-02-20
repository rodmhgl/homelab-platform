package compliance

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes dynamic client for CRD queries
type Client struct {
	dynamicClient dynamic.Interface
	config        *Config
}

// Config holds the configuration for the compliance client
type Config struct {
	KubeConfig string
	InCluster  bool
}

// NewClient creates a new Kubernetes dynamic client
func NewClient(cfg *Config) (*Client, error) {
	var config *rest.Config
	var err error

	if cfg.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
		slog.Debug("Using in-cluster Kubernetes configuration")
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
		slog.Debug("Using kubeconfig from file", "path", cfg.KubeConfig)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		config:        cfg,
	}, nil
}

// Constraint kinds we need to query (all 8 deployed constraints)
var constraintKinds = []string{
	"K8sRequiredLabels",
	"ContainerLimitsRequired",
	"NoLatestTag",
	"AllowedRepos",
	"NoPrivilegedContainers",
	"RequireProbes",
	"CrossplaneClaimLocation",
	"CrossplaneNoPublicAccess",
}

// ListConstraints queries all Constraint resources of a given kind
func (c *Client) ListConstraints(ctx context.Context, constraintKind string) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "constraints.gatekeeper.sh",
		Version:  "v1beta1",
		Resource: toLowerPlural(constraintKind),
	}

	slog.Debug("Listing constraints", "kind", constraintKind, "gvr", gvr.String())

	list, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s constraints: %w", constraintKind, err)
	}

	return list, nil
}

// ListAllConstraints queries all 8 deployed constraint kinds
func (c *Client) ListAllConstraints(ctx context.Context) (map[string]*unstructured.UnstructuredList, error) {
	results := make(map[string]*unstructured.UnstructuredList)

	for _, kind := range constraintKinds {
		list, err := c.ListConstraints(ctx, kind)
		if err != nil {
			// Log error but continue with other constraints (graceful degradation)
			slog.Warn("Failed to list constraints", "kind", kind, "error", err)
			continue
		}
		results[kind] = list
	}

	return results, nil
}

// ListConstraintTemplates queries all ConstraintTemplate CRDs
func (c *Client) ListConstraintTemplates(ctx context.Context) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "templates.gatekeeper.sh",
		Version:  "v1beta1",
		Resource: "constrainttemplates",
	}

	slog.Debug("Listing constraint templates")

	list, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list constraint templates: %w", err)
	}

	return list, nil
}

// ListVulnerabilityReports queries Trivy VulnerabilityReport CRDs across namespaces
func (c *Client) ListVulnerabilityReports(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "aquasecurity.github.io",
		Version:  "v1alpha1",
		Resource: "vulnerabilityreports",
	}

	slog.Debug("Listing vulnerability reports", "namespace", namespace)

	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		// Query all namespaces
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		// Query specific namespace
		list, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list vulnerability reports: %w", err)
	}

	return list, nil
}

// ListVulnerabilityReportsInWorkloads queries VulnerabilityReports in workload namespaces
// Excludes platform namespaces (kube-system, argocd, crossplane-system, etc.)
func (c *Client) ListVulnerabilityReportsInWorkloads(ctx context.Context) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "aquasecurity.github.io",
		Version:  "v1alpha1",
		Resource: "vulnerabilityreports",
	}

	// Query all namespaces first
	allReports, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list vulnerability reports: %w", err)
	}

	// Filter out platform namespaces
	excludedNamespaces := map[string]bool{
		"kube-system":       true,
		"argocd":            true,
		"crossplane-system": true,
		"gatekeeper-system": true,
		"trivy-system":      true,
		"external-secrets":  true,
		"monitoring":        true,
		"platform":          true,
	}

	filtered := &unstructured.UnstructuredList{
		Object: allReports.Object,
	}

	for _, item := range allReports.Items {
		ns := item.GetNamespace()
		if !excludedNamespaces[ns] {
			filtered.Items = append(filtered.Items, item)
		}
	}

	slog.Debug("Filtered vulnerability reports",
		"total", len(allReports.Items),
		"workloads", len(filtered.Items),
	)

	return filtered, nil
}

// toLowerPlural converts CamelCase constraint kind to lowercase plural resource name
// Example: K8sRequiredLabels -> k8srequiredlabels
func toLowerPlural(kind string) string {
	// Gatekeeper constraint kinds use lowercase resource names (no hyphens)
	result := ""
	for _, r := range kind {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // Convert to lowercase
		} else {
			result += string(r)
		}
	}
	return result
}
