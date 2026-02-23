// API Response Types (mirroring Go API structs)

// ========================================
// Argo CD Types
// ========================================

export interface ApplicationSummary {
  name: string;
  project: string;
  syncStatus: string;
  healthStatus: string;
  namespace: string;
  repoURL: string;
  path: string;
  revision?: string;
  lastDeployed?: string;
}

export interface Application extends ApplicationSummary {
  spec: {
    source: {
      repoURL: string;
      targetRevision: string;
      path: string;
    };
    destination: {
      server: string;
      namespace: string;
    };
    project: string;
  };
  status: {
    sync: {
      status: string;
      revision?: string;
    };
    health: {
      status: string;
      message?: string;
    };
    operationState?: {
      finishedAt?: string;
      message?: string;
    };
  };
}

export interface ListAppsResponse {
  applications: ApplicationSummary[];
  total: number;
}

export interface SyncRequest {
  prune: boolean;
  dryRun: boolean;
}

// ========================================
// Crossplane Types
// ========================================

export interface ClaimSummary {
  name: string;
  namespace: string;
  kind: string;
  status: string; // Ready, Progressing, Failed
  synced: boolean;
  ready: boolean;
  connectionSecret?: string;
  creationTimestamp: string;
  labels?: Record<string, string>;
}

export interface ClaimResource {
  name: string;
  namespace: string;
  kind: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  status: string; // Ready, Progressing, Failed
  synced: boolean;
  ready: boolean;
  connectionSecret?: string;
  creationTimestamp: string;
  resourceRef?: ResourceRef;
}

export interface CompositeResource {
  name: string;
  kind: string;
  labels?: Record<string, string>;
  status: string;
  synced: boolean;
  ready: boolean;
  creationTimestamp: string;
  resourceRefs?: ResourceRef[];
}

export interface ManagedResource {
  name: string;
  kind: string;
  group: string;
  labels?: Record<string, string>;
  status: string;
  synced: boolean;
  ready: boolean;
  externalName?: string; // Azure resource name
  creationTimestamp: string;
  message?: string; // Latest status message
}

export interface ResourceRef {
  name: string;
  kind: string;
  apiVersion?: string;
}

export interface ListClaimsResponse {
  claims: ClaimSummary[];
  total: number;
}

export interface GetResourceResponse {
  claim: ClaimResource;
  composite?: CompositeResource;
  managed: ManagedResource[];
  events: KubernetesEvent[];
}

export interface KubernetesEvent {
  type: string; // Normal, Warning
  reason: string;
  message: string;
  involvedObject: string; // "kind/name"
  source?: string;
  count?: number;
  firstTimestamp: string;
  lastTimestamp: string;
}

export interface CreateInfraRequest {
  kind: string;
  name: string;
  namespace: string;
  spec: Record<string, unknown>;
  appRepoURL: string;
}

// ========================================
// Compliance Types
// ========================================

export interface SummaryResponse {
  complianceScore: number;                           // 0-100 percentage
  totalViolations: number;
  totalVulnerabilities: number;
  violationsBySeverity: Record<string, number>;      // e.g., { policy: 2, config: 1 }
  vulnerabilitiesBySeverity: Record<string, number>; // e.g., { CRITICAL: 5, HIGH: 12 }
}

export interface Violation {
  constraintName: string;        // Matches json:"constraintName"
  constraintKind: string;         // Matches json:"constraintKind"
  resource: string;               // Composite field: "namespace/kind/name" or "kind/name"
  namespace: string;              // Empty string for cluster-scoped resources
  message: string;                // May include [action] prefix like "[audit] message"
  timestamp?: string;             // Optional (omitempty in Go)
}

export interface ListViolationsResponse {
  violations: Violation[];        // Only field in response
}

export interface Vulnerability {
  image: string;            // Container image (registry/org/name:tag)
  namespace: string;        // Kubernetes namespace
  workload: string;         // ReplicaSet or Deployment name
  cveId: string;            // CVE identifier (e.g., CVE-2024-12345)
  severity: string;         // CRITICAL | HIGH | MEDIUM | LOW | UNKNOWN
  score?: number;           // CVSS v3 score (0-10)
  affectedPackage: string;  // Package name (e.g., openssl)
  fixedVersion?: string;    // Fixed version if available
  primaryLink?: string;     // Link to NVD or vendor advisory
}

export interface ListVulnerabilitiesResponse {
  vulnerabilities: Vulnerability[];
  count: number;
}

export interface SecurityEvent {
  timestamp: string;
  rule: string;
  priority: string;
  message: string;
  source: string;
  tags: string[];
  output: string;
  outputFields: Record<string, unknown>;
  hostname: string;
}

export interface ListSecurityEventsResponse {
  events: SecurityEvent[];
  count: number;
}

// ========================================
// Scaffold Types
// ========================================

export interface ScaffoldRequest {
  template: string;
  projectName: string;
  namespace: string;
  includeStorage: boolean;
  includeVault: boolean;
  githubOwner: string;
  githubRepo: string;
  visibility: string;
}

export interface ScaffoldResponse {
  message: string;
  repoURL: string;
  appConfigCommitURL: string;
}

// ========================================
// Platform Health Types
// ========================================

export interface HealthResponse {
  status: string;
  timestamp: string;
}
