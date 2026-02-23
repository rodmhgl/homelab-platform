import ApplicationsPanel from '../components/dashboard/ApplicationsPanel';
import InfrastructurePanel from '../components/dashboard/InfrastructurePanel';
import CompliancePanel from '../components/dashboard/CompliancePanel';
import PolicyViolationsPanel from '../components/dashboard/PolicyViolationsPanel';
import VulnerabilityFeedPanel from '../components/dashboard/VulnerabilityFeedPanel';
import SecurityEventsPanel from '../components/dashboard/SecurityEventsPanel';

export default function Dashboard() {
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-gray-900">Platform Dashboard</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        <ApplicationsPanel />
        <InfrastructurePanel />
        <CompliancePanel />
        <PolicyViolationsPanel />
        <VulnerabilityFeedPanel />
        <SecurityEventsPanel />
      </div>
    </div>
  );
}
