import { useQuery } from '@tanstack/react-query';
import { complianceApi } from '../../api/compliance';
import StatusCard from '../common/StatusCard';
import Badge from '../common/Badge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { ListSecurityEventsResponse, SecurityEvent } from '../../api/types';

function getSeverityVariant(severity: string): 'danger' | 'warning' | 'info' | 'default' {
  const upperSeverity = severity.toUpperCase();
  // Critical, Alert, Emergency → danger (red)
  if (upperSeverity === 'CRITICAL' || upperSeverity === 'ALERT' || upperSeverity === 'EMERGENCY') {
    return 'danger';
  }
  // Error → warning (yellow)
  if (upperSeverity === 'ERROR') {
    return 'warning';
  }
  // Warning, Notice → info (blue)
  if (upperSeverity === 'WARNING' || upperSeverity === 'NOTICE') {
    return 'info';
  }
  // Debug, Informational → default (gray)
  return 'default';
}

function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
  } catch {
    return timestamp;
  }
}

function truncateMessage(message: string, maxLength: number = 100): string {
  if (message.length <= maxLength) return message;
  return message.slice(0, maxLength) + '...';
}

export default function SecurityEventsPanel() {
  const { data, isLoading, error } = useQuery<ListSecurityEventsResponse>({
    queryKey: ['compliance-events'],
    queryFn: () => complianceApi.events(),
    refetchInterval: 30000,
  });

  return (
    <StatusCard title="Security Events">
      {isLoading && (
        <div className="py-8">
          <LoadingSpinner message="Loading security events..." />
        </div>
      )}

      {error && (
        <div className="py-8 text-center">
          <p className="text-red-600">Failed to load security events</p>
          <p className="text-sm text-gray-500 mt-2">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      )}

      {data && data.events.length === 0 && (
        <div className="py-8 text-center text-gray-500">
          <p className="text-lg">✓ No security events detected</p>
          <p className="text-sm mt-1">Falco is monitoring runtime activity across all namespaces</p>
        </div>
      )}

      {data && data.events.length > 0 && (
        <>
          <div className="max-h-96 overflow-y-auto">
            <table className="table-auto w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200 sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700" style={{ width: '140px' }}>
                    Timestamp
                  </th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700" style={{ width: '100px' }}>
                    Severity
                  </th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700" style={{ width: '180px' }}>
                    Rule
                  </th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700" style={{ width: '150px' }}>
                    Resource
                  </th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">
                    Message
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.events.map((event: SecurityEvent, index: number) => (
                  <tr
                    key={index}
                    className="border-b border-gray-100 hover:bg-gray-50"
                  >
                    <td className="px-4 py-3 text-gray-600 text-xs">
                      {formatTimestamp(event.timestamp)}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={getSeverityVariant(event.severity)}>
                        {event.severity}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-gray-700 text-xs">
                      {event.rule}
                    </td>
                    <td className="px-4 py-3 text-gray-600 font-mono text-xs">
                      {event.resource || '-'}
                    </td>
                    <td
                      className="px-4 py-3 text-gray-600 text-xs"
                      title={event.message}
                    >
                      {truncateMessage(event.message)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-4 pt-4 border-t border-gray-200 text-sm text-gray-600">
            {data.events.length} event{data.events.length !== 1 ? 's' : ''} captured (most recent first)
          </div>
        </>
      )}
    </StatusCard>
  );
}
