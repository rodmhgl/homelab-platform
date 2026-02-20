interface StatusCardProps {
  title: string;
  children: React.ReactNode;
  className?: string;
}

export default function StatusCard({ title, children, className = '' }: StatusCardProps) {
  return (
    <div className={`bg-white rounded-lg shadow border border-gray-200 ${className}`}>
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
      </div>
      <div className="p-6">{children}</div>
    </div>
  );
}
