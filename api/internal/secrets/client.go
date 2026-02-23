package secrets

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds the configuration for the secrets client.
type Config struct {
	KubeConfig string
	InCluster  bool
}

// Client provides access to Kubernetes secrets (both ExternalSecrets and core Secrets).
type Client struct {
	dynamicClient dynamic.Interface
	coreClient    kubernetes.Interface
}

// NewClient creates a new secrets client.
func NewClient(cfg *Config) (*Client, error) {
	var config *rest.Config
	var err error

	if cfg.InCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	coreClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		coreClient:    coreClient,
	}, nil
}

// ListSecrets returns all secrets in a namespace (both ExternalSecrets and connection Secrets).
func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]SecretSummary, error) {
	var allSecrets []SecretSummary

	// Query ExternalSecrets (graceful failure - ESO might not be installed)
	externalSecrets, err := c.ListExternalSecrets(ctx, namespace)
	if err != nil {
		slog.Warn("Failed to list ExternalSecrets", "namespace", namespace, "error", err)
	} else {
		allSecrets = append(allSecrets, externalSecrets...)
	}

	// Query connection secrets (core API - must succeed)
	connectionSecrets, err := c.ListConnectionSecrets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list connection secrets: %w", err)
	}
	allSecrets = append(allSecrets, connectionSecrets...)

	// Sort: ExternalSecrets first, then by name
	sort.Slice(allSecrets, func(i, j int) bool {
		if allSecrets[i].Kind != allSecrets[j].Kind {
			return allSecrets[i].Kind == "ExternalSecret"
		}
		return allSecrets[i].Name < allSecrets[j].Name
	})

	return allSecrets, nil
}

// ListExternalSecrets queries external-secrets.io/v1beta1 CRDs.
func (c *Client) ListExternalSecrets(ctx context.Context, namespace string) ([]SecretSummary, error) {
	gvr := schema.GroupVersionResource{
		Group:    "external-secrets.io",
		Version:  "v1beta1",
		Resource: "externalsecrets",
	}

	list, err := c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ExternalSecrets: %w", err)
	}

	var secrets []SecretSummary
	for _, item := range list.Items {
		summary := c.parseExternalSecret(item)
		secrets = append(secrets, summary)
	}

	slog.Info("Listed ExternalSecrets", "namespace", namespace, "count", len(secrets))
	return secrets, nil
}

// ListConnectionSecrets queries core Secrets with Crossplane labels.
func (c *Client) ListConnectionSecrets(ctx context.Context, namespace string) ([]SecretSummary, error) {
	// Label selector for Crossplane connection secrets
	labelSelector := "crossplane.io/claim-name"

	list, err := c.coreClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list connection secrets: %w", err)
	}

	var secrets []SecretSummary
	for _, secret := range list.Items {
		summary := c.parseConnectionSecret(secret)
		secrets = append(secrets, summary)
	}

	slog.Info("Listed connection secrets", "namespace", namespace, "count", len(secrets))
	return secrets, nil
}

// parseExternalSecret converts an unstructured ExternalSecret to SecretSummary.
func (c *Client) parseExternalSecret(obj unstructured.Unstructured) SecretSummary {
	summary := SecretSummary{
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Kind:              "ExternalSecret",
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		Labels:            obj.GetLabels(),
	}

	// Parse status conditions
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err == nil && found {
		conditions, found, err := unstructured.NestedSlice(status, "conditions")
		if err == nil && found && len(conditions) > 0 {
			// Find the Ready condition
			for _, cond := range conditions {
				condMap, ok := cond.(map[string]interface{})
				if !ok {
					continue
				}

				condType, _, _ := unstructured.NestedString(condMap, "type")
				if condType == "Ready" {
					statusStr, _, _ := unstructured.NestedString(condMap, "status")
					message, _, _ := unstructured.NestedString(condMap, "message")

					if statusStr == "True" {
						summary.Status = "Ready"
					} else if statusStr == "False" {
						summary.Status = "Error"
					} else {
						summary.Status = "Unknown"
					}
					summary.Message = message
					break
				}
			}
		}
	}

	// Parse spec.target.name (the target Secret name)
	targetName, found, err := unstructured.NestedString(obj.Object, "spec", "target", "name")
	if err == nil && found && targetName != "" {
		// Try to get the keys from the target Secret
		if keys := c.getSecretKeys(obj.GetNamespace(), targetName); len(keys) > 0 {
			summary.Keys = keys
		}
	}

	// If we couldn't get keys from target, try dataFrom/data in spec
	if len(summary.Keys) == 0 {
		summary.Keys = c.extractExternalSecretKeys(obj)
	}

	return summary
}

// parseConnectionSecret converts a core Secret to SecretSummary.
func (c *Client) parseConnectionSecret(secret corev1.Secret) SecretSummary {
	summary := SecretSummary{
		Name:              secret.Name,
		Namespace:         secret.Namespace,
		Kind:              "Secret",
		Type:              string(secret.Type),
		Status:            "Ready", // Core Secrets don't have status conditions
		CreationTimestamp: secret.CreationTimestamp.Time,
		Labels:            secret.Labels,
	}

	// Extract keys (without values)
	for key := range secret.Data {
		summary.Keys = append(summary.Keys, key)
	}
	sort.Strings(summary.Keys)

	// Parse Crossplane Claim reference
	if claimName, ok := secret.Labels["crossplane.io/claim-name"]; ok {
		claimNamespace := secret.Labels["crossplane.io/claim-namespace"]
		claimKind := secret.Labels["crossplane.io/claim-kind"]
		if claimNamespace == "" {
			claimNamespace = secret.Namespace
		}
		if claimKind == "" {
			claimKind = "Claim" // Generic fallback
		}

		summary.SourceClaim = &ResourceRef{
			Name: claimName,
			Kind: claimKind,
		}
	}

	return summary
}

// getSecretKeys retrieves the keys from a Secret by name (best-effort).
func (c *Client) getSecretKeys(namespace, name string) []string {
	ctx := context.Background()
	secret, err := c.coreClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	var keys []string
	for key := range secret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// extractExternalSecretKeys extracts expected keys from ExternalSecret spec.
func (c *Client) extractExternalSecretKeys(obj unstructured.Unstructured) []string {
	var keys []string

	// Try spec.data[]
	data, found, err := unstructured.NestedSlice(obj.Object, "spec", "data")
	if err == nil && found {
		for _, item := range data {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			secretKey, _, _ := unstructured.NestedString(itemMap, "secretKey")
			if secretKey != "" {
				keys = append(keys, secretKey)
			}
		}
	}

	// Try spec.dataFrom[] (imports all keys from source)
	dataFrom, found, err := unstructured.NestedSlice(obj.Object, "spec", "dataFrom")
	if err == nil && found && len(dataFrom) > 0 {
		// We can't determine the keys without querying the source
		// Return a placeholder indicating multiple keys
		if len(keys) == 0 {
			keys = append(keys, "(multiple keys from source)")
		}
	}

	sort.Strings(keys)
	return keys
}
