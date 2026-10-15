package db

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		sql: `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			totp_secret TEXT NOT NULL DEFAULT '',
			totp_enabled BOOLEAN NOT NULL DEFAULT 0,
			recovery_codes TEXT NOT NULL DEFAULT '[]',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			ip_address TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
		CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

		CREATE TABLE IF NOT EXISTS invitations (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL DEFAULT 'member',
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMP NOT NULL,
			accepted_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations(token_hash);

		CREATE TABLE IF NOT EXISTS servers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			host TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			os TEXT NOT NULL DEFAULT 'linux',
			agent_version TEXT NOT NULL DEFAULT '',
			enrollment_token_hash TEXT NOT NULL DEFAULT '',
			auth_token_hash TEXT NOT NULL DEFAULT '',
			last_heartbeat_at TIMESTAMP,
			cpu_pct REAL NOT NULL DEFAULT 0.0,
			ram_pct REAL NOT NULL DEFAULT 0.0,
			disk_pct REAL NOT NULL DEFAULT 0.0,
			disk_free_bytes INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_servers_auth_token_hash ON servers(auth_token_hash);
		CREATE INDEX IF NOT EXISTS idx_servers_enrollment_token_hash ON servers(enrollment_token_hash);

		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			repo_url TEXT NOT NULL DEFAULT '',
			branch TEXT NOT NULL DEFAULT 'main',
			frontend_dir TEXT NOT NULL DEFAULT 'frontend/',
			build_cmd TEXT NOT NULL DEFAULT 'npm run build',
			output_dir TEXT NOT NULL DEFAULT 'dist/',
			server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			deploy_dir TEXT NOT NULL DEFAULT '/var/www/depo',
			run_user TEXT NOT NULL DEFAULT 'www-data',
			caddyfile_path TEXT NOT NULL DEFAULT 'frontend/Caddyfile',
			domain TEXT NOT NULL DEFAULT '',
			deploy_token_hash TEXT NOT NULL DEFAULT '',
			auto_deploy BOOLEAN NOT NULL DEFAULT 1,
			preview_enabled BOOLEAN NOT NULL DEFAULT 0,
			preview_base_domain TEXT NOT NULL DEFAULT '',
			log_retention_days INTEGER NOT NULL DEFAULT 7,
			active_release_id TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_projects_server_id ON projects(server_id);
		CREATE INDEX IF NOT EXISTS idx_projects_deploy_token ON projects(deploy_token_hash);

		CREATE TABLE IF NOT EXISTS env_vars (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			key TEXT NOT NULL,
			encrypted_value TEXT NOT NULL,
			is_build_time BOOLEAN NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, key)
		);
		CREATE INDEX IF NOT EXISTS idx_env_vars_project_id ON env_vars(project_id);

		CREATE TABLE IF NOT EXISTS releases (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			commit_hash TEXT NOT NULL DEFAULT '',
			artifact_checksum TEXT NOT NULL,
			artifact_path TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT 0,
			deployed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_releases_project_id ON releases(project_id);

		CREATE TABLE IF NOT EXISTS deployments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			release_id TEXT REFERENCES releases(id) ON DELETE SET NULL,
			commit_hash TEXT NOT NULL DEFAULT '',
			commit_msg TEXT NOT NULL DEFAULT '',
			branch TEXT NOT NULL DEFAULT 'main',
			trigger TEXT NOT NULL DEFAULT 'manual',
			status TEXT NOT NULL DEFAULT 'queued',
			error_message TEXT NOT NULL DEFAULT '',
			artifact_id TEXT NOT NULL DEFAULT '',
			started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id);
		CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);

		CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'pending',
			lease_token TEXT NOT NULL DEFAULT '',
			leased_at TIMESTAMP,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			error_message TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_server_status ON jobs(server_id, status);

		CREATE TABLE IF NOT EXISTS previews (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			pr_number INTEGER NOT NULL DEFAULT 0,
			branch TEXT NOT NULL DEFAULT '',
			commit_hash TEXT NOT NULL DEFAULT '',
			domain TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'active',
			release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_previews_project_id ON previews(project_id);
		CREATE INDEX IF NOT EXISTS idx_previews_expires_at ON previews(expires_at);

		CREATE TABLE IF NOT EXISTS webhook_events (
			id TEXT PRIMARY KEY,
			event_id TEXT NOT NULL UNIQUE,
			source TEXT NOT NULL DEFAULT 'github',
			event_type TEXT NOT NULL DEFAULT '',
			payload_hash TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'processed',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_events_event_id ON webhook_events(event_id);

		CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT '',
			user_email TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL DEFAULT '',
			details_json TEXT NOT NULL DEFAULT '{}',
			ip_address TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

		CREATE TABLE IF NOT EXISTS deployment_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
			level TEXT NOT NULL DEFAULT 'info',
			stage TEXT NOT NULL DEFAULT 'build',
			message TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id ON deployment_logs(deployment_id);

		CREATE TABLE IF NOT EXISTS notification_settings (
			id TEXT PRIMARY KEY,
			ntfy_url TEXT NOT NULL DEFAULT 'https://ntfy.sh',
			topic TEXT NOT NULL DEFAULT '',
			auth_token TEXT NOT NULL DEFAULT '',
			events_json TEXT NOT NULL DEFAULT '["deploy_success","deploy_failed","server_offline","disk_warning"]',
			enabled BOOLEAN NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		`,
	},
	{
		version: 2,
		name:    "update_default_deploy_dir_to_var_www_depo",
		sql: `
		UPDATE projects SET deploy_dir = '/var/www/depo' 
		WHERE deploy_dir = '/var/www/' 
		   OR deploy_dir = '/var/www' 
		   OR deploy_dir = '' 
		   OR deploy_dir = '\var\www' 
		   OR deploy_dir = '\var\www\';
		`,
	},
}

// Migrate applies all pending migrations in order.
func (d *DB) Migrate() error {
	return d.Transact(func(tx *sql.Tx) error {
		// Ensure migration tracking table exists
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("create schema_migrations: %w", err)
		}

		for _, m := range migrations {
			var exists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", m.version).Scan(&exists)
			if err != nil {
				return fmt.Errorf("checking migration %d: %w", m.version, err)
			}

			if !exists {
				if _, err := tx.Exec(m.sql); err != nil {
					return fmt.Errorf("executing migration %d (%s): %w", m.version, m.name, err)
				}
				if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.version, m.name); err != nil {
					return fmt.Errorf("recording migration %d: %w", m.version, err)
				}
			}
		}

		return nil
	})
}
