import { useQuery } from '@tanstack/react-query';
import { complianceApi } from '../../api/compliance';
import StatusCard from '../common/StatusCard';
import Badge from '../common/Badge';
import LoadingSpinner from '../common/LoadingSpinner';
import type { ListViolationsResponse, Violation } from '../../api/types';

function getConstraintVariant(kind: string): 'danger' | 'warning' | 'info' {
  const lowerKind = kind.toLowerCase();
  // Security-related constraints → danger (red)
  if (lowerKind.includes('privileged') || lowerKind.includes('publicaccess')) {
    return 'danger';
  }
  // Policy/config constraints → warning (yellow)
  if (lowerKind.includes('required') || lowerKind.includes('allowed') || lowerKind.includes('latest')) {
    return 'warning';
  }
  // Default → info (blue)
  return 'info';
}

export default function PolicyViolationsPanel() {
  const { data, isLoading, error } = useQuery<ListViolationsResponse>({
    queryKey: ['compliance-violations'],
    queryFn: () => complianceApi.violations(),
    refetchInterval: 30000,
  });

  return (
    <StatusCard title="Policy Violations">
      {isLoading && (
        <div className="py-8">
          <LoadingSpinner message="Loading policy violations..." />
        </div>
      )}

      {error && (
        <div className="py-8 text-center">
          <p className="text-red-600">Failed to load policy violations</p>
          <p className="text-sm text-gray-500 mt-2">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      )}

      {data && data.violations.length === 0 && (
        <div className="py-8 text-center text-gray-500">
          <p className="text-lg">✓ No policy violations found</p>
          <p className="text-sm mt-1">All resources are compliant with platform policies</p>
        </div>
      )}

      {data && data.violations.length > 0 && (
        <>
          <div className="max-h-96 overflow-y-auto">
            <table className="table-auto w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200 sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">Constraint</th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">Kind</th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">Resource</th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">Namespace</th>
                  <th className="px-4 py-2 text-left font-semibold text-gray-700">Message</th>
                </tr>
              </thead>
              <tbody>
                {data.violations.map((violation: Violation, index: number) => (
                  <tr
                    key={index}
                    className="border-b border-gray-100 hover:bg-gray-50"
                  >
                    <td className="px-4 py-3 text-gray-900 font-medium">
                      {violation.constraintName}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={getConstraintVariant(violation.constraintKind)}>
                        {violation.constraintKind}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-gray-700 font-mono text-xs">
                      {violation.resource}
                    </td>
                    <td className="px-4 py-3 text-gray-600">
                      {violation.namespace || '-'}
                    </td>
                    <td className="px-4 py-3 text-gray-600 text-xs">
                      {violation.message}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-4 pt-4 border-t border-gray-200 text-sm text-gray-600">
            {data.violations.length} violation{data.violations.length !== 1 ? 's' : ''} found
          </div>
        </>
      )}
    </StatusCard>
  );
}
