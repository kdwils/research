import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../utils/api';
import { HealthCheck } from '../types/healthcheck';
import HealthBadge from './HealthBadge';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

export default function HealthCheckDetail() {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();
  const [healthCheck, setHealthCheck] = useState<HealthCheck | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!namespace || !name) return;

    api
      .getHealthCheck(namespace, name)
      .then((hc) => {
        setHealthCheck(hc);
        setError(null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load health check');
      })
      .finally(() => setLoading(false));
  }, [namespace, name]);

  if (loading) {
    return <div className="text-center py-12">Loading...</div>;
  }

  if (error || !healthCheck) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-800">{error || 'Health check not found'}</p>
        <Link to="/" className="mt-2 text-blue-600 hover:text-blue-800 underline">
          Back to dashboard
        </Link>
      </div>
    );
  }

  const uptimeData = healthCheck.status.uptime?.map((u) => ({
    window: u.window,
    uptime: u.percentage,
  })) || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <Link to="/" className="text-blue-600 hover:text-blue-800 text-sm">
            ← Back to dashboard
          </Link>
          <h1 className="mt-2 text-3xl font-bold text-gray-900">
            {healthCheck.metadata.name}
          </h1>
          <p className="text-gray-600">Namespace: {healthCheck.metadata.namespace}</p>
        </div>
        <HealthBadge phase={healthCheck.status.phase} />
      </div>

      {/* Target Info */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Target</h2>
        {healthCheck.spec.targetRef ? (
          <div className="space-y-2">
            <div><span className="font-medium">Kind:</span> {healthCheck.spec.targetRef.kind}</div>
            <div><span className="font-medium">Name:</span> {healthCheck.spec.targetRef.name}</div>
            <div><span className="font-medium">Namespace:</span> {healthCheck.spec.targetRef.namespace || healthCheck.metadata.namespace}</div>
          </div>
        ) : healthCheck.spec.endpoints ? (
          <div>
            <div className="font-medium mb-2">Manual Endpoints ({healthCheck.spec.endpoints.length})</div>
            <div className="space-y-2">
              {healthCheck.spec.endpoints.map((ep) => (
                <div key={ep.name} className="bg-gray-50 p-3 rounded">
                  <div><span className="font-medium">Name:</span> {ep.name}</div>
                  <div><span className="font-medium">Address:</span> {ep.address}</div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="text-gray-500">No target specified</div>
        )}
      </div>

      {/* Uptime Chart */}
      {uptimeData.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Uptime</h2>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={uptimeData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="window" />
              <YAxis domain={[0, 100]} />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="uptime" stroke="#10b981" name="Uptime %" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Checks */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Health Checks</h2>
        <div className="space-y-4">
          {healthCheck.status.checkResults?.map((check) => (
            <div key={check.name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <div className="font-medium text-gray-900">{check.name}</div>
                <span
                  className={`px-2 py-1 text-xs font-semibold rounded ${
                    check.healthy
                      ? 'bg-green-100 text-green-800'
                      : 'bg-red-100 text-red-800'
                  }`}
                >
                  {check.healthy ? 'Healthy' : 'Unhealthy'}
                </span>
              </div>
              <div className="text-sm text-gray-600 space-y-1">
                <div>Message: {check.message || 'N/A'}</div>
                <div>Consecutive Successes: {check.consecutiveSuccesses || 0}</div>
                <div>Consecutive Failures: {check.consecutiveFailures || 0}</div>
              </div>
            </div>
          )) || <div className="text-gray-500">No check results available</div>}
        </div>
      </div>

      {/* Endpoints Health */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Endpoint Health</h2>
        <div className="space-y-3">
          {healthCheck.status.podHealth?.map((pod) => (
            <div key={pod.endpointName} className="border border-gray-200 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <div>
                  <div className="font-medium text-gray-900">{pod.endpointName}</div>
                  <div className="text-sm text-gray-500">
                    {pod.endpointType === 'manual' ? 'Manual Endpoint' : 'Pod'} - {pod.podIP}
                  </div>
                </div>
                <span
                  className={`px-2 py-1 text-xs font-semibold rounded ${
                    pod.healthy ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}
                >
                  {pod.healthy ? 'Healthy' : 'Unhealthy'}
                </span>
              </div>
              <div className="text-sm text-gray-600">
                Healthy: {pod.checksHealthy} / Unhealthy: {pod.checksUnhealthy}
              </div>
            </div>
          )) || <div className="text-gray-500">No endpoint health data available</div>}
        </div>
      </div>
    </div>
  );
}
