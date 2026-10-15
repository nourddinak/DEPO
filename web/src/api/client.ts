import {
  User, Server, Project, Deployment, Release, DeploymentLog,
  Preview, EnvVar, NotificationSettings, AuditLog, Invitation
} from '../types';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

async function fetchApi<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(path, {
    ...options,
    headers,
  });

  const body: ApiResponse<T> = await res.json().catch(() => ({
    success: false,
    data: {} as T,
    error: `HTTP error ${res.status}`,
  }));

  if (!res.ok || !body.success) {
    throw new Error(body.error || `Request failed with status ${res.status}`);
  }

  return body.data;
}

export const api = {
  // Auth
  getAuthStatus: () => fetchApi<{ setup_required: boolean; authenticated: boolean; user?: User }>('/api/v1/auth/status'),
  setupOwner: (email: string, password: string) => fetchApi<{ user: User; token: string }>('/api/v1/auth/setup', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }),
  login: (email: string, password: string, totp_code?: string) => fetchApi<{ user?: User; token?: string; totp_required?: boolean }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password, totp_code }),
  }),
  logout: () => fetchApi<{ message: string }>('/api/v1/auth/logout', { method: 'POST' }),
  getCurrentUser: () => fetchApi<User>('/api/v1/auth/me'),
  setupTOTP: () => fetchApi<{ secret: string; url: string }>('/api/v1/auth/totp/setup', { method: 'POST' }),
  verifyTOTP: (code: string) => fetchApi<{ message: string; recovery_codes: string[] }>('/api/v1/auth/totp/verify', {
    method: 'POST',
    body: JSON.stringify({ code }),
  }),

  // Users & Invitations
  getUsers: async () => (await fetchApi<User[]>('/api/v1/users')) || [],
  getInvitations: async () => (await fetchApi<Invitation[]>('/api/v1/auth/invitations')) || [],
  createInvitation: (email: string, role: string) => fetchApi<{ invitation: Invitation; invitation_token: string; invitation_url: string }>('/api/v1/auth/invitations', {
    method: 'POST',
    body: JSON.stringify({ email, role }),
  }),

  // Servers
  getServers: async () => (await fetchApi<Server[]>('/api/v1/servers')) || [],
  getServer: (id: string) => fetchApi<Server>(`/api/v1/servers/${id}`),
  createServer: (name: string, host?: string) => fetchApi<{ server: Server; enrollment_token: string; install_command: string }>('/api/v1/servers', {
    method: 'POST',
    body: JSON.stringify({ name, host }),
  }),
  deleteServer: (id: string) => fetchApi<{ message: string }>(`/api/v1/servers/${id}`, { method: 'DELETE' }),
  resetServerEnrollToken: (id: string) => fetchApi<{ server: Server; enrollment_token: string; install_command: string }>(`/api/v1/servers/${id}/enroll-token`, {
    method: 'POST',
  }),

  // Projects
  getProjects: async () => (await fetchApi<Project[]>('/api/v1/projects')) || [],
  getProject: (id: string) => fetchApi<Project>(`/api/v1/projects/${id}`),
  createProject: (data: Partial<Project>) => fetchApi<{ project: Project; deploy_token: string; workflow_yaml: string }>('/api/v1/projects', {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  updateProject: (id: string, data: Partial<Project>) => fetchApi<Project>(`/api/v1/projects/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteProject: (id: string) => fetchApi<{ message: string }>(`/api/v1/projects/${id}`, { method: 'DELETE' }),
  detectFramework: (package_json: string) => fetchApi<{ framework: string; frontend_dir: string; build_cmd: string; output_dir: string }>('/api/v1/projects/detect', {
    method: 'POST',
    body: JSON.stringify({ package_json }),
  }),
  getProjectWorkflow: (id: string) => fetchApi<{ workflow_yaml: string; target_path: string }>(`/api/v1/projects/${id}/workflow`),

  // Deployments & Releases
  getDeployments: async (projectId?: string) => (await fetchApi<Deployment[]>(`/api/v1/deployments${projectId ? `?project_id=${projectId}` : ''}`)) || [],
  getDeployment: (id: string) => fetchApi<Deployment>(`/api/v1/deployments/${id}`),
  getDeploymentLogs: async (id: string) => (await fetchApi<DeploymentLog[]>(`/api/v1/deployments/${id}/logs`)) || [],
  getProjectReleases: async (projectId: string) => (await fetchApi<Release[]>(`/api/v1/projects/${projectId}/releases`)) || [],
  rollbackRelease: (projectId: string, releaseId: string) => fetchApi<{ message: string; deployment_id: string; release_id: string }>(`/api/v1/projects/${projectId}/rollback`, {
    method: 'POST',
    body: JSON.stringify({ release_id: releaseId }),
  }),

  // Previews
  getPreviews: async (projectId?: string) => (await fetchApi<Preview[]>(`/api/v1/previews${projectId ? `?project_id=${projectId}` : ''}`)) || [],
  destroyPreview: (id: string) => fetchApi<{ message: string }>(`/api/v1/previews/${id}/destroy`, { method: 'POST' }),

  // Environment Variables
  getProjectEnv: async (projectId: string) => {
    const res = await fetchApi<{ variables: EnvVar[]; raw_content: string; warning: string }>(`/api/v1/projects/${projectId}/env`);
    return {
      variables: res?.variables || [],
      raw_content: res?.raw_content || '',
      warning: res?.warning || '',
    };
  },
  updateProjectEnv: (projectId: string, payload: { variables?: EnvVar[]; raw_content?: string; trigger_deploy?: boolean }) =>
    fetchApi<{ message: string; updated_keys: string[] }>(`/api/v1/projects/${projectId}/env`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  parseEnvExample: async (content: string) => (await fetchApi<EnvVar[]>('/api/v1/projects/parse-env-example', {
    method: 'POST',
    body: JSON.stringify({ content }),
  })) || [],

  // Notifications
  getNotifications: async () => {
    const res = await fetchApi<NotificationSettings>('/api/v1/settings/notifications');
    return {
      ...res,
      events: res?.events || [],
    };
  },
  updateNotifications: (data: Partial<NotificationSettings>) => fetchApi<{ message: string }>('/api/v1/settings/notifications', {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  testNotification: () => fetchApi<{ message: string }>('/api/v1/settings/notifications/test', { method: 'POST' }),

  // DNS Diagnostics
  checkDomain: (domain: string) => fetchApi<{ domain: string; resolved_ips: string[]; cname: string; is_routed: boolean; instructions: string }>(`/api/v1/domains/check?domain=${encodeURIComponent(domain)}`),

  // Audit Logs
  getAuditLogs: async () => (await fetchApi<AuditLog[]>('/api/v1/audit-logs')) || [],
};
