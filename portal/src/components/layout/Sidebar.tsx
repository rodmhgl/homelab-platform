import { NavLink } from 'react-router-dom';

const navItems = [
  { to: '/', label: 'Dashboard', icon: '📊' },
  { to: '/apps', label: 'Applications', icon: '🚀' },
  { to: '/infra', label: 'Infrastructure', icon: '🏗️' },
  { to: '/compliance', label: 'Compliance', icon: '🛡️' },
  { to: '/scaffold', label: 'Scaffold', icon: '🔧' },
];

export default function Sidebar() {
  return (
    <div className="w-64 bg-gray-900 text-white min-h-screen flex flex-col">
      <div className="p-6 border-b border-gray-800">
        <h1 className="text-2xl font-bold">Homelab Platform</h1>
        <p className="text-sm text-gray-400 mt-1">Internal Developer Platform</p>
      </div>

      <nav className="flex-1 px-4 py-6 space-y-2">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                isActive
                  ? 'bg-primary text-white'
                  : 'text-gray-300 hover:bg-gray-800 hover:text-white'
              }`
            }
          >
            <span className="text-xl">{item.icon}</span>
            <span className="font-medium">{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="p-4 border-t border-gray-800">
        <div className="text-xs text-gray-500">
          <p>Platform v0.1.0</p>
          <p className="mt-1">Powered by Crossplane + Argo CD</p>
        </div>
      </div>
    </div>
  );
}
