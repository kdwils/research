interface HealthBadgeProps {
  phase?: 'Healthy' | 'Degraded' | 'Unhealthy' | 'Unknown';
}

export default function HealthBadge({ phase }: HealthBadgeProps) {
  const colors = {
    Healthy: 'bg-green-100 text-green-800',
    Degraded: 'bg-yellow-100 text-yellow-800',
    Unhealthy: 'bg-red-100 text-red-800',
    Unknown: 'bg-gray-100 text-gray-800',
  };

  return (
    <span
      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
        colors[phase || 'Unknown']
      }`}
    >
      {phase || 'Unknown'}
    </span>
  );
}
