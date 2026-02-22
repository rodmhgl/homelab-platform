import { useQuery } from '@tanstack/react-query';
import { appsApi } from '../../api/apps';
import StatusCard from '../common/StatusCard';
import Badge from '../common/Badge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { ApplicationSummary } from '../../api/types';

function getHealthVariant(status: string): 'success' | 'warning' | 'danger' | 'info' | 'default' {
  switch (status.toLowerCase()) {
    case 'healthy':
      return 'success';
    case 'progressing':
      return 'info';
    case 'degraded':
      return 'warning';
    case 'suspended':
    case 'missing':
    case 'unknown':
      return 'danger';
    default:
      return 'default';
  }
}

function getSyncVariant(status: string): 'success' | 'warning' | 'danger' | 'info' | 'default' {
  switch (status.toLowerCase()) {
    case 'synced':
      return 'success';
    case 'outofsync':
      return 'warning';
    case 'unknown':
      return 'default';
    default:
      return 'info';
  }
}

function formatTimestamp(timestamp?: string): string {
  if (!timestamp) return 'Never';

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

interface ApplicationCardProps {
  app: ApplicationSummary;
}

function ApplicationCard({ app }: ApplicationCardProps) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div>
          <h4 className="text-base font-semibold text-gray-900">{app.name}</h4>
          <p className="text-sm text-gray-500 mt-1">{app.namespace}</p>
        </div>
        <Badge variant={getHealthVariant(app.healthStatus)} size="sm">
          {app.healthStatus}
        </Badge>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Sync Status:</span>
          <Badge variant={getSyncVariant(app.syncStatus)} size="sm">
            {app.syncStatus}
          </Badge>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Project:</span>
          <span className="font-medium text-gray-900">{app.project}</span>
        </div>

        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-600">Last Deployed:</span>
          <span className="font-medium text-gray-900">
            {formatTimestamp(app.lastDeployed)}
          </span>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t border-gray-100">
        <div className="flex items-center text-xs text-gray-500 truncate">
          <svg
            className="h-3.5 w-3.5 mr-1.5 flex-shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
            />
          </svg>
          <span className="truncate" title={app.repoURL}>
            {app.path}
          </span>
        </div>
      </div>
    </div>
  );
}

export default function ApplicationsPanel() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['apps'],
    queryFn: appsApi.list,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  return (
    <StatusCard title="Applications">
      {isLoading && (
        <div className="py-8">
          <LoadingSpinner message="Loading applications..." />
        </div>
      )}

      {error && (
        <div className="py-8 text-center">
          <p className="text-red-600">Failed to load applications</p>
          <p className="text-sm text-gray-500 mt-1">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      )}

      {data && data.applications.length === 0 && (
        <div className="py-8 text-center">
          <p className="text-gray-500">No applications found</p>
          <p className="text-sm text-gray-400 mt-1">
            Create your first application using the scaffold tool
          </p>
        </div>
      )}

      {data && data.applications.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.applications.map((app) => (
              <ApplicationCard key={app.name} app={app} />
            ))}
          </div>

          <div className="mt-4 pt-4 border-t border-gray-200 text-sm text-gray-600">
            Showing {data.applications.length} application{data.applications.length !== 1 ? 's' : ''}
          </div>
        </>
      )}
    </StatusCard>
  );
}
