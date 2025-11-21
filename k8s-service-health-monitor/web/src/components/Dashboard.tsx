import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../utils/api';
import { useWebSocket } from '../hooks/useWebSocket';
import { HealthCheck, HealthSummary } from '../types/healthcheck';
import HealthBadge from './HealthBadge';

export default function Dashboard() {
  const [healthChecks, setHealthChecks] = useState<HealthCheck[]>([]);
  const [summary, setSummary] = useState<HealthSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState('');

  const loadData = async () => {
    try {
      const [checks, summaryData] = await Promise.all([
        api.listHealthChecks(),
        api.getClusterSummary(),
      ]);
      setHealthChecks(checks.items || []);
      setSummary(summaryData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const { connected } = useWebSocket((message) => {
    if (message.type === 'update' || message.type === 'add') {
      setHealthChecks((prev) => {
        const existing = prev.findIndex(
          (hc) =>
            hc.metadata.name === message.resource.metadata.name &&
            hc.metadata.namespace === message.resource.metadata.namespace
        );
        if (existing >= 0) {
          const updated = [...prev];
          updated[existing] = message.resource;
          return updated;
        }
        return [...prev, message.resource];
      });
      // Reload summary
      api.getClusterSummary().then(setSummary);
    } else if (message.type === 'delete') {
      setHealthChecks((prev) =>
        prev.filter(
          (hc) =>
            !(
              hc.metadata.name === message.resource.metadata.name &&
              hc.metadata.namespace === message.resource.metadata.namespace
            )
        )
      );
      api.getClusterSummary().then(setSummary);
    }
  });

  const filteredHealthChecks = healthChecks.filter((hc) => {
    if (!filter) return true;
    const search = filter.toLowerCase();
    return (
      hc.metadata.name.toLowerCase().includes(search) ||
      hc.metadata.namespace.toLowerCase().includes(search) ||
      hc.status.phase?.toLowerCase().includes(search)
    );
  });

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-600">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-800">{error}</p>
        <button
          onClick={() => loadData()}
          className="mt-2 text-red-600 hover:text-red-800 underline"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* WebSocket Status */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <div
            className={`w-3 h-3 rounded-full ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          <span className="text-sm text-gray-600">
            {connected ? 'Live updates' : 'Disconnected'}
          </span>
        </div>
      </div>

      {/* Summary Cards */}
      {summary && (
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm font-medium text-gray-500">Total</div>
            <div className="mt-2 text-3xl font-semibold text-gray-900">
              {summary.total}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm font-medium text-gray-500">Healthy</div>
            <div className="mt-2 text-3xl font-semibold text-green-600">
              {summary.healthy}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm font-medium text-gray-500">Degraded</div>
            <div className="mt-2 text-3xl font-semibold text-yellow-600">
              {summary.degraded}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm font-medium text-gray-500">Unhealthy</div>
            <div className="mt-2 text-3xl font-semibold text-red-600">
              {summary.unhealthy}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm font-medium text-gray-500">Unknown</div>
            <div className="mt-2 text-3xl font-semibold text-gray-600">
              {summary.unknown}
            </div>
          </div>
        </div>
      )}

      {/* Filter */}
      <div className="bg-white rounded-lg shadow p-4">
        <input
          type="text"
          placeholder="Filter by name, namespace, or status..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {/* HealthChecks List */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Namespace
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Target
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Checks
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Uptime (24h)
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {filteredHealthChecks.map((hc) => {
              const uptime24h = hc.status.uptime?.find((u) => u.window === '24h');
              const target = hc.spec.targetRef
                ? `Service: ${hc.spec.targetRef.name}`
                : hc.spec.endpoints
                ? `Manual (${hc.spec.endpoints.length})`
                : 'N/A';

              return (
                <tr
                  key={`${hc.metadata.namespace}/${hc.metadata.name}`}
                  className="hover:bg-gray-50"
                >
                  <td className="px-6 py-4 whitespace-nowrap">
                    <Link
                      to={`/healthcheck/${hc.metadata.namespace}/${hc.metadata.name}`}
                      className="text-blue-600 hover:text-blue-800 font-medium"
                    >
                      {hc.metadata.name}
                    </Link>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {hc.metadata.namespace}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <HealthBadge phase={hc.status.phase} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {target}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {hc.spec.checks.length} checks
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {uptime24h ? `${uptime24h.percentage.toFixed(2)}%` : 'N/A'}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredHealthChecks.length === 0 && (
          <div className="text-center py-12 text-gray-500">
            No health checks found
          </div>
        )}
      </div>
    </div>
  );
}
