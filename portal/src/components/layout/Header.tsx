import { useQuery } from '@tanstack/react-query';
import { healthApi } from '../../api/health';
import Badge from '../common/Badge';

export default function Header() {
  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: healthApi.check,
    refetchInterval: 30000,
    retry: 3,
  });

  return (
    <header className="bg-white shadow-sm border-b border-gray-200">
      <div className="px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h2 className="text-xl font-semibold text-gray-800">Platform Dashboard</h2>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600">Platform API:</span>
            {health?.status === 'healthy' ? (
              <Badge variant="success" size="sm">
                Healthy
              </Badge>
            ) : (
              <Badge variant="danger" size="sm">
                Unhealthy
              </Badge>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
