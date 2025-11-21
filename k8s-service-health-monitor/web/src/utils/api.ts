import { HealthCheck, HealthCheckList, HealthSummary } from '../types/healthcheck';

const API_BASE = '/api/v1';

export const api = {
  async listHealthChecks(namespace?: string, labelSelector?: string): Promise<HealthCheckList> {
    const params = new URLSearchParams();
    if (namespace) params.append('namespace', namespace);
    if (labelSelector) params.append('labelSelector', labelSelector);

    const url = `${API_BASE}/healthchecks${params.toString() ? '?' + params.toString() : ''}`;
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to fetch health checks: ${response.statusText}`);
    }
    return response.json();
  },

  async getHealthCheck(namespace: string, name: string): Promise<HealthCheck> {
    const response = await fetch(`${API_BASE}/healthchecks/${namespace}/${name}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch health check: ${response.statusText}`);
    }
    return response.json();
  },

  async getClusterSummary(): Promise<HealthSummary> {
    const response = await fetch(`${API_BASE}/health-summary`);
    if (!response.ok) {
      throw new Error(`Failed to fetch cluster summary: ${response.statusText}`);
    }
    return response.json();
  },

  async getNamespaceSummary(namespace: string): Promise<HealthSummary> {
    const response = await fetch(`${API_BASE}/namespaces/${namespace}/health-summary`);
    if (!response.ok) {
      throw new Error(`Failed to fetch namespace summary: ${response.statusText}`);
    }
    return response.json();
  },
};
