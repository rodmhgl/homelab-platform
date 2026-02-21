import ApplicationsPanel from '../components/dashboard/ApplicationsPanel';

export default function Dashboard() {
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-gray-900">Platform Dashboard</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <ApplicationsPanel />
        {/* Additional panels will be added here (tasks #80-#84) */}
      </div>
    </div>
  );
}
