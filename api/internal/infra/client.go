package infra

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes clients for Crossplane resource queries
type Client struct {
	dynamicClient dynamic.Interface
	coreClient    kubernetes.Interface
	config        *Config
}

// Config holds the configuration for the infra client
type Config struct {
	KubeConfig string
	InCluster  bool
}

// NewClient creates a new Kubernetes client
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

	coreClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		coreClient:    coreClient,
		config:        cfg,
	}, nil
}

// GetClaim retrieves a Crossplane Claim by kind and name
func (c *Client) GetClaim(ctx context.Context, namespace, kind, name string) (*unstructured.Unstructured, error) {
	gvr, err := claimKindToGVR(kind)
	if err != nil {
		return nil, err
	}

	slog.Debug("Getting claim", "kind", kind, "namespace", namespace, "name", name, "gvr", gvr.String())

	claim, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get claim %s/%s: %w", kind, name, err)
	}

	return claim, nil
}

// GetComposite retrieves the composite resource referenced by a Claim
func (c *Client) GetComposite(ctx context.Context, ref ResourceRef) (*unstructured.Unstructured, error) {
	gvr, err := compositeKindToGVR(ref.Kind)
	if err != nil {
		return nil, err
	}

	slog.Debug("Getting composite", "kind", ref.Kind, "name", ref.Name, "gvr", gvr.String())

	composite, err := c.dynamicClient.Resource(gvr).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get composite %s/%s: %w", ref.Kind, ref.Name, err)
	}

	return composite, nil
}

// GetManagedResource retrieves a Crossplane managed resource
func (c *Client) GetManagedResource(ctx context.Context, ref ResourceRef) (*unstructured.Unstructured, error) {
	gvr, err := managedResourceToGVR(ref)
	if err != nil {
		return nil, err
	}

	slog.Debug("Getting managed resource", "kind", ref.Kind, "name", ref.Name, "gvr", gvr.String())

	resource, err := c.dynamicClient.Resource(gvr).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get managed resource %s/%s: %w", ref.Kind, ref.Name, err)
	}

	return resource, nil
}

// GetEventsForResource retrieves Kubernetes events for a resource
func (c *Client) GetEventsForResource(ctx context.Context, namespace, kind, name string) ([]corev1.Event, error) {
	slog.Debug("Getting events", "namespace", namespace, "kind", kind, "name", name)

	// Build field selector for the specific resource
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s", name, kind)

	var events *corev1.EventList
	var err error

	if namespace != "" {
		events, err = c.coreClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	} else {
		// For cluster-scoped resources (composites, managed resources)
		events, err = c.coreClient.CoreV1().Events("").List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events.Items, nil
}

// Helper functions

// claimKindToGVR maps Claim kind to GroupVersionResource
func claimKindToGVR(kind string) (schema.GroupVersionResource, error) {
	switch strings.ToLower(kind) {
	case "storagebucket":
		return schema.GroupVersionResource{
			Group:    "platform.example.com",
			Version:  "v1alpha1",
			Resource: "storagebuckets",
		}, nil
	case "vault":
		return schema.GroupVersionResource{
			Group:    "platform.example.com",
			Version:  "v1alpha1",
			Resource: "vaults",
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown claim kind: %s", kind)
	}
}

// compositeKindToGVR maps Composite kind to GroupVersionResource
func compositeKindToGVR(kind string) (schema.GroupVersionResource, error) {
	switch kind {
	case "XStorageBucket":
		return schema.GroupVersionResource{
			Group:    "platform.example.com",
			Version:  "v1alpha1",
			Resource: "xstoragebuckets",
		}, nil
	case "XKeyVault":
		return schema.GroupVersionResource{
			Group:    "platform.example.com",
			Version:  "v1alpha1",
			Resource: "xkeyvaults",
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown composite kind: %s", kind)
	}
}

// managedResourceToGVR maps managed resource kind to GroupVersionResource
func managedResourceToGVR(ref ResourceRef) (schema.GroupVersionResource, error) {
	// Extract group and version from APIVersion (format: "group/version")
	parts := strings.Split(ref.APIVersion, "/")
	if len(parts) != 2 {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid apiVersion format: %s", ref.APIVersion)
	}

	group := parts[0]
	version := parts[1]

	// Convert Kind to resource name (lowercase + 's')
	// This is a simple heuristic; may need refinement for edge cases
	resource := strings.ToLower(ref.Kind) + "s"

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}, nil
}

// extractResourceRef extracts a ResourceRef from an unstructured object's spec.resourceRef
func extractResourceRef(obj *unstructured.Unstructured) (*ResourceRef, bool) {
	refMap, found, err := unstructured.NestedMap(obj.Object, "spec", "resourceRef")
	if err != nil || !found || refMap == nil {
		return nil, false
	}

	name, _, _ := unstructured.NestedString(refMap, "name")
	kind, _, _ := unstructured.NestedString(refMap, "kind")
	apiVersion, _, _ := unstructured.NestedString(refMap, "apiVersion")

	if name == "" || kind == "" {
		return nil, false
	}

	return &ResourceRef{
		Name:       name,
		Kind:       kind,
		APIVersion: apiVersion,
	}, true
}

// extractResourceRefs extracts ResourceRef list from an unstructured object's spec.resourceRefs
func extractResourceRefs(obj *unstructured.Unstructured) []ResourceRef {
	refsSlice, found, err := unstructured.NestedSlice(obj.Object, "spec", "resourceRefs")
	if err != nil || !found {
		return nil
	}

	var refs []ResourceRef
	for _, refVal := range refsSlice {
		refMap, ok := refVal.(map[string]interface{})
		if !ok {
			continue
		}

		name, _, _ := unstructured.NestedString(refMap, "name")
		kind, _, _ := unstructured.NestedString(refMap, "kind")
		apiVersion, _, _ := unstructured.NestedString(refMap, "apiVersion")

		if name != "" && kind != "" {
			refs = append(refs, ResourceRef{
				Name:       name,
				Kind:       kind,
				APIVersion: apiVersion,
			})
		}
	}

	return refs
}

// extractConditionStatus extracts Ready and Synced status from conditions array
func extractConditionStatus(obj *unstructured.Unstructured) (ready bool, synced bool) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return false, false
	}

	for _, condVal := range conditions {
		condMap, ok := condVal.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		condStatus, _, _ := unstructured.NestedString(condMap, "status")

		if condType == "Ready" && condStatus == "True" {
			ready = true
		}
		if condType == "Synced" && condStatus == "True" {
			synced = true
		}
	}

	return ready, synced
}

// extractStatusMessage extracts the latest status message from conditions
func extractStatusMessage(obj *unstructured.Unstructured) string {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return ""
	}

	var latestMessage string
	var latestTime time.Time

	for _, condVal := range conditions {
		condMap, ok := condVal.(map[string]interface{})
		if !ok {
			continue
		}

		message, _, _ := unstructured.NestedString(condMap, "message")
		timeStr, _, _ := unstructured.NestedString(condMap, "lastTransitionTime")

		if message == "" {
			continue
		}

		if timeStr != "" {
			t, err := time.Parse(time.RFC3339, timeStr)
			if err == nil && t.After(latestTime) {
				latestTime = t
				latestMessage = message
			}
		} else if latestMessage == "" {
			latestMessage = message
		}
	}

	return latestMessage
}

// determineStatus determines overall status from ready/synced conditions
func determineStatus(ready, synced bool) string {
	if ready && synced {
		return "Ready"
	}
	if !ready && !synced {
		return "Failed"
	}
	return "Progressing"
}

// ListClaims retrieves all Claims of a specific kind across all namespaces
func (c *Client) ListClaims(ctx context.Context, kind string) (*unstructured.UnstructuredList, error) {
	gvr, err := claimKindToGVR(kind)
	if err != nil {
		return nil, err
	}

	slog.Debug("Listing claims", "kind", kind, "gvr", gvr.String())

	claims, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list claims of kind %s: %w", kind, err)
	}

	return claims, nil
}

// ListAllClaims retrieves all Claims (StorageBucket + Vault) across all namespaces
func (c *Client) ListAllClaims(ctx context.Context) ([]unstructured.Unstructured, error) {
	var allClaims []unstructured.Unstructured

	// List StorageBucket Claims
	storageClaims, err := c.ListClaims(ctx, "StorageBucket")
	if err != nil {
		slog.Warn("Failed to list StorageBucket claims", "error", err)
	} else {
		allClaims = append(allClaims, storageClaims.Items...)
	}

	// List Vault Claims
	vaultClaims, err := c.ListClaims(ctx, "Vault")
	if err != nil {
		slog.Warn("Failed to list Vault claims", "error", err)
	} else {
		allClaims = append(allClaims, vaultClaims.Items...)
	}

	return allClaims, nil
}
