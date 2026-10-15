export interface User {
  id: string;
  email: string;
  role: 'owner' | 'admin' | 'member';
  totp_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Server {
  id: string;
  name: string;
  host: string;
  status: 'pending' | 'online' | 'offline' | 'degraded';
  os: string;
  agent_version: string;
  last_heartbeat_at?: string;
  cpu_pct: number;
  ram_pct: number;
  disk_pct: number;
  disk_free_bytes: number;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  repo_url: string;
  branch: string;
  frontend_dir: string;
  build_cmd: string;
  output_dir: string;
  server_id: string;
  server_name?: string;
  deploy_dir: string;
  run_user: string;
  caddyfile_path: string;
  domain: string;
  auto_deploy: boolean;
  preview_enabled: boolean;
  preview_base_domain: string;
  log_retention_days: number;
  active_release_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Deployment {
  id: string;
  project_id: string;
  project_name?: string;
  server_id: string;
  server_name?: string;
  release_id?: string;
  commit_hash: string;
  commit_msg: string;
  branch: string;
  trigger: string;
  status: 'queued' | 'building' | 'uploading' | 'delivering' | 'activating' | 'success' | 'failed' | 'rolled_back';
  error_message?: string;
  artifact_id?: string;
  started_at: string;
  completed_at?: string;
}

export interface Release {
  id: string;
  project_id: string;
  server_id: string;
  commit_hash: string;
  artifact_checksum: string;
  is_active: boolean;
  deployed_at: string;
}

export interface DeploymentLog {
  id: number;
  deployment_id: string;
  level: string;
  stage: string;
  message: string;
  created_at: string;
}

export interface Preview {
  id: string;
  project_id: string;
  project_name?: string;
  server_id: string;
  pr_number: number;
  branch: string;
  commit_hash: string;
  domain: string;
  status: 'active' | 'expired' | 'destroyed';
  release_id: string;
  expires_at: string;
  created_at: string;
}

export interface EnvVar {
  key: string;
  value: string;
  is_build_time: boolean;
  comment?: string;
}

export interface NotificationSettings {
  id: string;
  ntfy_url: string;
  topic: string;
  auth_token?: string;
  has_token: boolean;
  events: string[];
  enabled: boolean;
}

export interface AuditLog {
  id: string;
  user_id: string;
  user_email: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details_json: string;
  ip_address: string;
  created_at: string;
}

export interface Invitation {
  id: string;
  email: string;
  role: 'admin' | 'member';
  expires_at: string;
  accepted_at?: string;
  created_at: string;
}
