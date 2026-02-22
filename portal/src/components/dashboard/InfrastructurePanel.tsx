import { useQuery } from '@tanstack/react-query';
import { infraApi } from '../../api/infra';
import StatusCard from '../common/StatusCard';
import Badge from '../common/Badge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { ClaimSummary } from '../../api/types';

function getStatusVariant(status: string): 'success' | 'warning' | 'danger' | 'info' | 'default' {
  switch (status.toLowerCase()) {
    case 'ready':
      return 'success';
    case 'progressing':
      return 'info';
    case 'failed':
      return 'danger';
    default:
      return 'default';
  }
}

function getReadyVariant(ready: boolean): 'success' | 'warning' {
  return ready ? 'success' : 'warning';
}

function getSyncedVariant(synced: boolean): 'success' | 'warning' {
  return synced ? 'success' : 'warning';
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}

interface ClaimCardProps {
  claim: ClaimSummary;
}

function ClaimCard({ claim }: ClaimCardProps) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div>
          <h4 className="text-base font-semibold text-gray-900">{claim.name}</h4>
          <p className="text-sm text-gray-500 mt-1">{claim.namespace}</p>
        </div>
        <Badge variant={getStatusVariant(claim.status)} size="sm">
          {claim.status}
        </Badge>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Kind:</span>
          <span className="font-medium text-gray-900">{claim.kind}</span>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Ready:</span>
          <Badge variant={getReadyVariant(claim.ready)} size="sm">
            {claim.ready ? 'Ready' : 'Not Ready'}
          </Badge>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Synced:</span>
          <Badge variant={getSyncedVariant(claim.synced)} size="sm">
            {claim.synced ? 'Synced' : 'Out of Sync'}
          </Badge>
        </div>

        {claim.connectionSecret && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">Secret:</span>
            <span className="font-medium text-gray-900 font-mono text-xs truncate">
              {claim.connectionSecret}
            </span>
          </div>
        )}

        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Created:</span>
          <span className="font-medium text-gray-900">
            {formatTimestamp(claim.creationTimestamp)}
          </span>
        </div>
      </div>
    </div>
  );
}

export default function InfrastructurePanel() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['claims'],
    queryFn: infraApi.list,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  const readyCount = data?.claims.filter((c) => c.ready).length ?? 0;

  return (
    <StatusCard title="Infrastructure">
      {isLoading && (
        <div className="py-8">
          <LoadingSpinner message="Loading infrastructure claims..." />
        </div>
      )}

      {error && (
        <div className="py-8 text-center">
          <p className="text-red-600">Failed to load infrastructure claims</p>
          <p className="text-sm text-gray-500 mt-1">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      )}

      {data && data.claims.length === 0 && (
        <div className="py-8 text-center">
          <p className="text-gray-500">No infrastructure claims found</p>
          <p className="text-sm text-gray-400 mt-1">
            Create storage or vaults using the scaffold tool or rdp CLI
          </p>
        </div>
      )}

      {data && data.claims.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.claims.map((claim) => (
              <ClaimCard key={`${claim.namespace}/${claim.name}`} claim={claim} />
            ))}
          </div>

          <div className="mt-4 pt-4 border-t border-gray-200 text-sm text-gray-600">
            Showing {data.total} claim{data.total !== 1 ? 's' : ''} ({readyCount} ready)
          </div>
        </>
      )}
    </StatusCard>
  );
}
