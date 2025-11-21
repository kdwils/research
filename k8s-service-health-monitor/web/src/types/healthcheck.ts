export interface HealthCheck {
  metadata: {
    name: string;
    namespace: string;
    creationTimestamp: string;
  };
  spec: HealthCheckSpec;
  status: HealthCheckStatus;
}

export interface HealthCheckSpec {
  targetRef?: {
    kind: string;
    name: string;
    namespace?: string;
  };
  endpoints?: ManualEndpoint[];
  interval?: string;
  timeout?: string;
  failureThreshold?: number;
  successThreshold?: number;
  checks: Check[];
  uptime?: {
    enabled: boolean;
    windows: string[];
  };
}

export interface ManualEndpoint {
  name: string;
  address: string;
  labels?: Record<string, string>;
}

export interface Check {
  name: string;
  type: 'HTTP' | 'TCP' | 'gRPC' | 'Exec';
  http?: HTTPCheck;
  tcp?: TCPCheck;
  grpc?: GRPCCheck;
  exec?: ExecCheck;
}

export interface HTTPCheck {
  scheme?: string;
  host?: string;
  port: number;
  path?: string;
  httpHeaders?: Array<{ name: string; value: string }>;
  expectedStatus?: number[];
  expectedBodyRegex?: string;
  expectedBodyContains?: string;
}

export interface TCPCheck {
  port: number;
}

export interface GRPCCheck {
  port: number;
  service?: string;
  useTLS?: boolean;
}

export interface ExecCheck {
  command: string[];
}

export interface HealthCheckStatus {
  phase?: 'Healthy' | 'Degraded' | 'Unhealthy' | 'Unknown';
  checkResults?: CheckResult[];
  podHealth?: PodHealthStatus[];
  uptime?: UptimeStat[];
  serviceEndpoints?: {
    ready: number;
    total: number;
  };
  conditions?: Condition[];
  observedGeneration?: number;
}

export interface CheckResult {
  name: string;
  healthy: boolean;
  lastCheckTime?: string;
  lastTransitionTime?: string;
  consecutiveSuccesses?: number;
  consecutiveFailures?: number;
  message?: string;
}

export interface PodHealthStatus {
  endpointName: string;
  endpointType?: string;
  podName?: string;
  podIP?: string;
  healthy: boolean;
  checksHealthy?: number;
  checksUnhealthy?: number;
  lastCheckTime?: string;
}

export interface UptimeStat {
  window: string;
  percentage: number;
  totalChecks?: number;
  successfulChecks?: number;
}

export interface Condition {
  type: string;
  status: string;
  lastTransitionTime: string;
  reason: string;
  message: string;
  observedGeneration?: number;
}

export interface HealthCheckList {
  items: HealthCheck[];
}

export interface HealthSummary {
  total: number;
  healthy: number;
  degraded: number;
  unhealthy: number;
  unknown: number;
  scope?: string;
  namespace?: string;
}
