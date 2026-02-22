import { useQuery } from '@tanstack/react-query';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import { complianceApi } from '../../api/compliance';
import StatusCard from '../common/StatusCard';
import Badge from '../common/Badge';
import LoadingSpinner from '../common/LoadingSpinner';

function getScoreColor(score: number): string {
  if (score >= 90) return '#10b981'; // green-500
  if (score >= 70) return '#f59e0b'; // amber-500
  return '#ef4444'; // red-500
}

function getScoreVariant(score: number): 'success' | 'warning' | 'danger' {
  if (score >= 90) return 'success';
  if (score >= 70) return 'warning';
  return 'danger';
}

export default function CompliancePanel() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['compliance-summary'],
    queryFn: complianceApi.summary,
    refetchInterval: 30000, // 30 seconds
  });

  return (
    <StatusCard title="Compliance Score">
      {/* Loading state */}
      {isLoading && (
        <div className="py-8">
          <LoadingSpinner message="Loading compliance data..." />
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="py-8 text-center">
          <p className="text-red-600">Failed to load compliance summary</p>
          <p className="text-sm text-gray-500 mt-1">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      )}

      {/* Success state */}
      {data && (
        <div className="space-y-6">
          {/* Score display - centered above chart */}
          <div className="text-center">
            <div className="text-5xl font-bold mb-2" style={{ color: getScoreColor(data.complianceScore) }}>
              {Math.round(data.complianceScore)}
            </div>
            <Badge variant={getScoreVariant(data.complianceScore)} size="sm">
              Compliance Score
            </Badge>
          </div>

          {/* Donut chart */}
          <div className="relative h-48">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={[
                    { name: 'Compliant', value: data.complianceScore },
                    { name: 'At Risk', value: Math.max(0, 100 - data.complianceScore) },
                  ]}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={80}
                  dataKey="value"
                  startAngle={90}
                  endAngle={-270}
                >
                  <Cell fill={getScoreColor(data.complianceScore)} />
                  <Cell fill="#e5e7eb" /> {/* gray-200 for remaining */}
                </Pie>
                <Tooltip formatter={(value) => `${value}%`} />
              </PieChart>
            </ResponsiveContainer>
          </div>

          {/* Breakdown metrics - grid below chart */}
          <div className="grid grid-cols-2 gap-4 pt-4 border-t border-gray-200">
            <div>
              <p className="text-sm text-gray-600 mb-1">Policy Violations</p>
              <p className="text-2xl font-semibold text-gray-900">{data.totalViolations}</p>
              {data.violationsBySeverity && Object.keys(data.violationsBySeverity).length > 0 && (
                <div className="flex gap-2 mt-2 flex-wrap">
                  {Object.entries(data.violationsBySeverity).map(([severity, count]) => (
                    <Badge key={severity} variant="warning" size="sm">
                      {severity}: {count}
                    </Badge>
                  ))}
                </div>
              )}
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-1">Vulnerabilities</p>
              <p className="text-2xl font-semibold text-gray-900">{data.totalVulnerabilities}</p>
              {data.vulnerabilitiesBySeverity && Object.keys(data.vulnerabilitiesBySeverity).length > 0 && (
                <div className="flex gap-2 mt-2 flex-wrap">
                  {Object.entries(data.vulnerabilitiesBySeverity)
                    .filter(([_, count]) => count > 0)
                    .map(([severity, count]) => (
                      <Badge
                        key={severity}
                        variant={severity === 'CRITICAL' || severity === 'HIGH' ? 'danger' : 'warning'}
                        size="sm"
                      >
                        {severity}: {count}
                      </Badge>
                    ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </StatusCard>
  );
}
