package db

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("record not found")
	ErrDuplicateEntry = errors.New("duplicate entry")
)

// --- User & Auth Repository ---

func (d *DB) CreateUser(email, passwordHash string, role UserRole) (*User, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	err := d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO users (id, email, password_hash, role, created_at, updated_at) 
		          VALUES (?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, id, email, passwordHash, string(role), now, now)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return d.GetUserByID(id)
}

func (d *DB) CountUsers() (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (d *DB) GetUserByID(id string) (*User, error) {
	u := &User{}
	query := `SELECT id, email, password_hash, role, totp_secret, totp_enabled, recovery_codes, created_at, updated_at 
	          FROM users WHERE id = ?`
	err := d.QueryRow(query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.RecoveryCodes, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	query := `SELECT id, email, password_hash, role, totp_secret, totp_enabled, recovery_codes, created_at, updated_at 
	          FROM users WHERE email = ?`
	err := d.QueryRow(query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.RecoveryCodes, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (d *DB) UpdateUserTOTP(userID, secret, recoveryCodes string, enabled bool) error {
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE users SET totp_secret = ?, totp_enabled = ?, recovery_codes = ?, updated_at = ? WHERE id = ?`
		_, err := tx.Exec(query, secret, enabled, recoveryCodes, time.Now().UTC(), userID)
		return err
	})
}

func (d *DB) UpdateUserPassword(userID, passwordHash string) error {
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
		_, err := tx.Exec(query, passwordHash, time.Now().UTC(), userID)
		return err
	})
}

func (d *DB) ListUsers() ([]*User, error) {
	query := `SELECT id, email, role, totp_enabled, created_at, updated_at FROM users ORDER BY created_at ASC`
	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*User, 0)
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Session Repository ---

func (d *DB) CreateSession(userID, tokenHash, ipAddress, userAgent string, expiresAt time.Time) (*Session, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	err := d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, id, userID, tokenHash, ipAddress, userAgent, expiresAt, now)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (d *DB) GetSessionByTokenHash(tokenHash string) (*Session, *User, error) {
	query := `SELECT s.id, s.user_id, s.token_hash, s.ip_address, s.user_agent, s.expires_at, s.created_at,
	                 u.id, u.email, u.password_hash, u.role, u.totp_secret, u.totp_enabled, u.recovery_codes, u.created_at, u.updated_at
	          FROM sessions s
	          JOIN users u ON s.user_id = u.id
	          WHERE s.token_hash = ? AND s.expires_at > ?`
	now := time.Now().UTC()

	s := &Session{}
	u := &User{}
	err := d.QueryRow(query, tokenHash, now).Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt,
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.RecoveryCodes, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return s, u, nil
}

func (d *DB) DeleteSession(id string) error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM sessions WHERE id = ?", id)
		return err
	})
}

func (d *DB) DeleteExpiredSessions() error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().UTC())
		return err
	})
}

// --- Invitation Repository ---

func (d *DB) CreateInvitation(email string, role UserRole, tokenHash string, expiresAt time.Time) (*Invitation, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	err := d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO invitations (id, email, role, token_hash, expires_at, created_at)
		          VALUES (?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, id, email, string(role), tokenHash, expiresAt, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Invitation{
		ID:        id,
		Email:     email,
		Role:      role,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (d *DB) GetInvitationByTokenHash(tokenHash string) (*Invitation, error) {
	query := `SELECT id, email, role, token_hash, expires_at, accepted_at, created_at
	          FROM invitations WHERE token_hash = ? AND expires_at > ? AND accepted_at IS NULL`
	now := time.Now().UTC()
	inv := &Invitation{}
	err := d.QueryRow(query, tokenHash, now).Scan(
		&inv.ID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (d *DB) MarkInvitationAccepted(id string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE invitations SET accepted_at = ? WHERE id = ?", now, id)
		return err
	})
}

func (d *DB) ListInvitations() ([]*Invitation, error) {
	query := `SELECT id, email, role, expires_at, accepted_at, created_at FROM invitations ORDER BY created_at DESC`
	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Invitation, 0)
	for rows.Next() {
		inv := &Invitation{}
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

// --- Server Repository ---

func (d *DB) CreateServer(name, host, osName, enrollmentTokenHash string) (*Server, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	err := d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO servers (id, name, host, status, os, enrollment_token_hash, created_at, updated_at)
		          VALUES (?, ?, ?, 'pending', ?, ?, ?, ?)`
		_, err := tx.Exec(query, id, name, host, osName, enrollmentTokenHash, now, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return d.GetServerByID(id)
}

func (d *DB) GetServerByID(id string) (*Server, error) {
	s := &Server{}
	query := `SELECT id, name, host, status, os, agent_version, enrollment_token_hash, auth_token_hash,
	                 last_heartbeat_at, cpu_pct, ram_pct, disk_pct, disk_free_bytes, created_at, updated_at
	          FROM servers WHERE id = ?`
	err := d.QueryRow(query, id).Scan(
		&s.ID, &s.Name, &s.Host, &s.Status, &s.OS, &s.AgentVersion, &s.EnrollmentTokenHash, &s.AuthTokenHash,
		&s.LastHeartbeatAt, &s.CPUPct, &s.RAMPct, &s.DiskPct, &s.DiskFreeBytes, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) GetServerByEnrollmentTokenHash(hash string) (*Server, error) {
	s := &Server{}
	query := `SELECT id, name, host, status, os, agent_version, enrollment_token_hash, auth_token_hash,
	                 last_heartbeat_at, cpu_pct, ram_pct, disk_pct, disk_free_bytes, created_at, updated_at
	          FROM servers WHERE enrollment_token_hash = ? AND status = 'pending'`
	err := d.QueryRow(query, hash).Scan(
		&s.ID, &s.Name, &s.Host, &s.Status, &s.OS, &s.AgentVersion, &s.EnrollmentTokenHash, &s.AuthTokenHash,
		&s.LastHeartbeatAt, &s.CPUPct, &s.RAMPct, &s.DiskPct, &s.DiskFreeBytes, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) GetServerByAuthTokenHash(hash string) (*Server, error) {
	s := &Server{}
	query := `SELECT id, name, host, status, os, agent_version, enrollment_token_hash, auth_token_hash,
	                 last_heartbeat_at, cpu_pct, ram_pct, disk_pct, disk_free_bytes, created_at, updated_at
	          FROM servers WHERE auth_token_hash = ?`
	err := d.QueryRow(query, hash).Scan(
		&s.ID, &s.Name, &s.Host, &s.Status, &s.OS, &s.AgentVersion, &s.EnrollmentTokenHash, &s.AuthTokenHash,
		&s.LastHeartbeatAt, &s.CPUPct, &s.RAMPct, &s.DiskPct, &s.DiskFreeBytes, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) CompleteServerEnrollment(serverID, authTokenHash, agentVersion string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE servers SET status = 'online', auth_token_hash = ?, enrollment_token_hash = '',
		          agent_version = ?, last_heartbeat_at = ?, updated_at = ? WHERE id = ?`
		_, err := tx.Exec(query, authTokenHash, agentVersion, now, now, serverID)
		return err
	})
}

func (d *DB) ResetServerEnrollmentToken(serverID, tokenHash string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE servers SET enrollment_token_hash = ?, status = 'pending', updated_at = ? WHERE id = ?`
		_, err := tx.Exec(query, tokenHash, now, serverID)
		return err
	})
}

func (d *DB) UpdateServerHeartbeat(serverID string, cpuPct, ramPct, diskPct float64, diskFreeBytes int64) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE servers SET status = 'online', last_heartbeat_at = ?, cpu_pct = ?, ram_pct = ?,
		          disk_pct = ?, disk_free_bytes = ?, updated_at = ? WHERE id = ?`
		_, err := tx.Exec(query, now, cpuPct, ramPct, diskPct, diskFreeBytes, now, serverID)
		return err
	})
}

func (d *DB) MarkInactiveServersOffline(timeoutDuration time.Duration) ([]string, error) {
	threshold := time.Now().UTC().Add(-timeoutDuration)
	offlineIDs := make([]string, 0)

	err := d.Transact(func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT id FROM servers WHERE status = 'online' AND last_heartbeat_at < ?", threshold)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			offlineIDs = append(offlineIDs, id)
		}

		if len(offlineIDs) > 0 {
			_, err = tx.Exec("UPDATE servers SET status = 'offline', updated_at = ? WHERE status = 'online' AND last_heartbeat_at < ?", time.Now().UTC(), threshold)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return offlineIDs, err
}

func (d *DB) ListServers() ([]*Server, error) {
	query := `SELECT id, name, host, status, os, agent_version, enrollment_token_hash, auth_token_hash,
	                 last_heartbeat_at, cpu_pct, ram_pct, disk_pct, disk_free_bytes, created_at, updated_at
	          FROM servers ORDER BY name ASC`
	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]*Server, 0)
	for rows.Next() {
		s := &Server{}
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Host, &s.Status, &s.OS, &s.AgentVersion, &s.EnrollmentTokenHash, &s.AuthTokenHash,
			&s.LastHeartbeatAt, &s.CPUPct, &s.RAMPct, &s.DiskPct, &s.DiskFreeBytes, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

func (d *DB) DeleteServer(id string) error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM servers WHERE id = ?", id)
		return err
	})
}

// --- Project Repository ---

// NormalizeDeployDir normalizes deployment directory paths across DEPO.
// It maps empty paths, whitespace, legacy "/var/www", and variations with trailing slashes
// or Windows backslashes to the standard base directory "/var/www/depo".
func NormalizeDeployDir(deployDir string) string {
	deployDir = strings.TrimSpace(deployDir)
	if deployDir == "" {
		return "/var/www/depo"
	}
	slashPath := filepath.ToSlash(deployDir)
	cleaned := strings.TrimRight(slashPath, "/")
	if cleaned == "" || cleaned == "/var/www" || cleaned == "." {
		return "/var/www/depo"
	}
	return cleaned
}

func (d *DB) CreateProject(p *Project) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	p.DeployDir = NormalizeDeployDir(p.DeployDir)
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO projects (
			id, name, repo_url, branch, frontend_dir, build_cmd, output_dir, server_id,
			deploy_dir, run_user, caddyfile_path, domain, deploy_token_hash, auto_deploy,
			preview_enabled, preview_base_domain, log_retention_days, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query,
			p.ID, p.Name, p.RepoURL, p.Branch, p.FrontendDir, p.BuildCmd, p.OutputDir, p.ServerID,
			p.DeployDir, p.RunUser, p.CaddyfilePath, p.Domain, p.DeployTokenHash, p.AutoDeploy,
			p.PreviewEnabled, p.PreviewBaseDomain, p.LogRetentionDays, p.CreatedAt, p.UpdatedAt,
		)
		return err
	})
}

func (d *DB) GetProjectByID(id string) (*Project, error) {
	p := &Project{}
	query := `SELECT p.id, p.name, p.repo_url, p.branch, p.frontend_dir, p.build_cmd, p.output_dir,
	                 p.server_id, s.name as server_name, p.deploy_dir, p.run_user, p.caddyfile_path,
	                 p.domain, p.deploy_token_hash, p.auto_deploy, p.preview_enabled, p.preview_base_domain,
	                 p.log_retention_days, p.active_release_id, p.created_at, p.updated_at
	          FROM projects p
	          JOIN servers s ON p.server_id = s.id
	          WHERE p.id = ?`
	err := d.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.FrontendDir, &p.BuildCmd, &p.OutputDir,
		&p.ServerID, &p.ServerName, &p.DeployDir, &p.RunUser, &p.CaddyfilePath,
		&p.Domain, &p.DeployTokenHash, &p.AutoDeploy, &p.PreviewEnabled, &p.PreviewBaseDomain,
		&p.LogRetentionDays, &p.ActiveReleaseID, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (d *DB) GetProjectByRepoAndBranch(repoURL, branch string) (*Project, error) {
	p := &Project{}
	query := `SELECT p.id, p.name, p.repo_url, p.branch, p.frontend_dir, p.build_cmd, p.output_dir,
	                 p.server_id, s.name as server_name, p.deploy_dir, p.run_user, p.caddyfile_path,
	                 p.domain, p.deploy_token_hash, p.auto_deploy, p.preview_enabled, p.preview_base_domain,
	                 p.log_retention_days, p.active_release_id, p.created_at, p.updated_at
	          FROM projects p
	          JOIN servers s ON p.server_id = s.id
	          WHERE p.repo_url = ? AND (p.branch = ? OR p.preview_enabled = 1)`
	err := d.QueryRow(query, repoURL, branch).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.FrontendDir, &p.BuildCmd, &p.OutputDir,
		&p.ServerID, &p.ServerName, &p.DeployDir, &p.RunUser, &p.CaddyfilePath,
		&p.Domain, &p.DeployTokenHash, &p.AutoDeploy, &p.PreviewEnabled, &p.PreviewBaseDomain,
		&p.LogRetentionDays, &p.ActiveReleaseID, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (d *DB) GetProjectByDeployTokenHash(tokenHash string) (*Project, error) {
	p := &Project{}
	query := `SELECT p.id, p.name, p.repo_url, p.branch, p.frontend_dir, p.build_cmd, p.output_dir,
	                 p.server_id, s.name as server_name, p.deploy_dir, p.run_user, p.caddyfile_path,
	                 p.domain, p.deploy_token_hash, p.auto_deploy, p.preview_enabled, p.preview_base_domain,
	                 p.log_retention_days, p.active_release_id, p.created_at, p.updated_at
	          FROM projects p
	          JOIN servers s ON p.server_id = s.id
	          WHERE p.deploy_token_hash = ?`
	err := d.QueryRow(query, tokenHash).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.FrontendDir, &p.BuildCmd, &p.OutputDir,
		&p.ServerID, &p.ServerName, &p.DeployDir, &p.RunUser, &p.CaddyfilePath,
		&p.Domain, &p.DeployTokenHash, &p.AutoDeploy, &p.PreviewEnabled, &p.PreviewBaseDomain,
		&p.LogRetentionDays, &p.ActiveReleaseID, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (d *DB) ListProjects() ([]*Project, error) {
	query := `SELECT p.id, p.name, p.repo_url, p.branch, p.frontend_dir, p.build_cmd, p.output_dir,
	                 p.server_id, s.name as server_name, p.deploy_dir, p.run_user, p.caddyfile_path,
	                 p.domain, p.deploy_token_hash, p.auto_deploy, p.preview_enabled, p.preview_base_domain,
	                 p.log_retention_days, p.active_release_id, p.created_at, p.updated_at
	          FROM projects p
	          JOIN servers s ON p.server_id = s.id
	          ORDER BY p.name ASC`
	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]*Project, 0)
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.RepoURL, &p.Branch, &p.FrontendDir, &p.BuildCmd, &p.OutputDir,
			&p.ServerID, &p.ServerName, &p.DeployDir, &p.RunUser, &p.CaddyfilePath,
			&p.Domain, &p.DeployTokenHash, &p.AutoDeploy, &p.PreviewEnabled, &p.PreviewBaseDomain,
			&p.LogRetentionDays, &p.ActiveReleaseID, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) UpdateProject(p *Project) error {
	p.DeployDir = NormalizeDeployDir(p.DeployDir)
	p.UpdatedAt = time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE projects SET
			name = ?, repo_url = ?, branch = ?, frontend_dir = ?, build_cmd = ?, output_dir = ?,
			server_id = ?, deploy_dir = ?, run_user = ?, caddyfile_path = ?, domain = ?,
			auto_deploy = ?, preview_enabled = ?, preview_base_domain = ?, log_retention_days = ?,
			active_release_id = ?, updated_at = ?
			WHERE id = ?`
		_, err := tx.Exec(query,
			p.Name, p.RepoURL, p.Branch, p.FrontendDir, p.BuildCmd, p.OutputDir,
			p.ServerID, p.DeployDir, p.RunUser, p.CaddyfilePath, p.Domain,
			p.AutoDeploy, p.PreviewEnabled, p.PreviewBaseDomain, p.LogRetentionDays,
			p.ActiveReleaseID, p.UpdatedAt, p.ID,
		)
		return err
	})
}

func (d *DB) DeleteProject(id string) error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM projects WHERE id = ?", id)
		return err
	})
}

// --- Environment Variables Repository ---

func (d *DB) SetEnvVar(projectID, key, encryptedVal string, isBuildTime bool) error {
	id := uuid.New().String()
	now := time.Now().UTC()

	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO env_vars (id, project_id, key, encrypted_value, is_build_time, created_at, updated_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?)
		          ON CONFLICT(project_id, key) DO UPDATE SET
		            encrypted_value = excluded.encrypted_value,
		            is_build_time = excluded.is_build_time,
		            updated_at = excluded.updated_at`
		_, err := tx.Exec(query, id, projectID, key, encryptedVal, isBuildTime, now, now)
		return err
	})
}

func (d *DB) GetEnvVars(projectID string) ([]*EnvVar, error) {
	query := `SELECT id, project_id, key, encrypted_value, is_build_time, created_at, updated_at
	          FROM env_vars WHERE project_id = ? ORDER BY key ASC`
	rows, err := d.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*EnvVar, 0)
	for rows.Next() {
		ev := &EnvVar{}
		if err := rows.Scan(&ev.ID, &ev.ProjectID, &ev.Key, &ev.EncryptedValue, &ev.IsBuildTime, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, ev)
	}
	return list, rows.Err()
}

func (d *DB) DeleteEnvVar(projectID, key string) error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM env_vars WHERE project_id = ? AND key = ?", projectID, key)
		return err
	})
}

// --- Deployments & Releases Repository ---

func (d *DB) CreateDeployment(dep *Deployment) error {
	if dep.ID == "" {
		dep.ID = uuid.New().String()
	}
	dep.StartedAt = time.Now().UTC()

	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO deployments (
			id, project_id, server_id, release_id, commit_hash, commit_msg, branch, trigger,
			status, error_message, artifact_id, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query,
			dep.ID, dep.ProjectID, dep.ServerID, dep.ReleaseID, dep.CommitHash, dep.CommitMsg, dep.Branch, dep.Trigger,
			string(dep.Status), dep.ErrorMessage, dep.ArtifactID, dep.StartedAt,
		)
		return err
	})
}

func (d *DB) GetDeploymentByID(id string) (*Deployment, error) {
	dep := &Deployment{}
	query := `SELECT d.id, d.project_id, p.name as project_name, d.server_id, s.name as server_name,
	                 d.release_id, d.commit_hash, d.commit_msg, d.branch, d.trigger, d.status,
	                 d.error_message, d.artifact_id, d.started_at, d.completed_at
	          FROM deployments d
	          JOIN projects p ON d.project_id = p.id
	          JOIN servers s ON d.server_id = s.id
	          WHERE d.id = ?`
	err := d.QueryRow(query, id).Scan(
		&dep.ID, &dep.ProjectID, &dep.ProjectName, &dep.ServerID, &dep.ServerName,
		&dep.ReleaseID, &dep.CommitHash, &dep.CommitMsg, &dep.Branch, &dep.Trigger, &dep.Status,
		&dep.ErrorMessage, &dep.ArtifactID, &dep.StartedAt, &dep.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return dep, nil
}

func (d *DB) UpdateDeploymentStatus(id string, status DeploymentStatus, releaseID *string, errMsg string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		var err error
		if status == DeploySuccess || status == DeployFailed || status == DeployRolledBack {
			query := `UPDATE deployments SET status = ?, release_id = COALESCE(?, release_id),
			          error_message = ?, completed_at = ? WHERE id = ?`
			_, err = tx.Exec(query, string(status), releaseID, errMsg, now, id)
		} else {
			query := `UPDATE deployments SET status = ?, release_id = COALESCE(?, release_id),
			          error_message = ? WHERE id = ?`
			_, err = tx.Exec(query, string(status), releaseID, errMsg, id)
		}
		return err
	})
}

func (d *DB) ListDeployments(projectID string, limit, offset int) ([]*Deployment, error) {
	var query string
	var args []any

	if projectID != "" {
		query = `SELECT d.id, d.project_id, p.name as project_name, d.server_id, s.name as server_name,
		                d.release_id, d.commit_hash, d.commit_msg, d.branch, d.trigger, d.status,
		                d.error_message, d.artifact_id, d.started_at, d.completed_at
		         FROM deployments d
		         JOIN projects p ON d.project_id = p.id
		         JOIN servers s ON d.server_id = s.id
		         WHERE d.project_id = ?
		         ORDER BY d.started_at DESC LIMIT ? OFFSET ?`
		args = []any{projectID, limit, offset}
	} else {
		query = `SELECT d.id, d.project_id, p.name as project_name, d.server_id, s.name as server_name,
		                d.release_id, d.commit_hash, d.commit_msg, d.branch, d.trigger, d.status,
		                d.error_message, d.artifact_id, d.started_at, d.completed_at
		         FROM deployments d
		         JOIN projects p ON d.project_id = p.id
		         JOIN servers s ON d.server_id = s.id
		         ORDER BY d.started_at DESC LIMIT ? OFFSET ?`
		args = []any{limit, offset}
	}

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deps := make([]*Deployment, 0)
	for rows.Next() {
		dep := &Deployment{}
		if err := rows.Scan(
			&dep.ID, &dep.ProjectID, &dep.ProjectName, &dep.ServerID, &dep.ServerName,
			&dep.ReleaseID, &dep.CommitHash, &dep.CommitMsg, &dep.Branch, &dep.Trigger, &dep.Status,
			&dep.ErrorMessage, &dep.ArtifactID, &dep.StartedAt, &dep.CompletedAt,
		); err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	return deps, rows.Err()
}

func (d *DB) CreateRelease(rel *Release) error {
	if rel.ID == "" {
		rel.ID = uuid.New().String()
	}
	rel.DeployedAt = time.Now().UTC()

	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO releases (id, project_id, server_id, commit_hash, artifact_checksum, artifact_path, is_active, deployed_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query,
			rel.ID, rel.ProjectID, rel.ServerID, rel.CommitHash, rel.ArtifactChecksum, rel.ArtifactPath, rel.IsActive, rel.DeployedAt,
		)
		return err
	})
}

func (d *DB) SetActiveRelease(projectID, releaseID string) error {
	return d.Transact(func(tx *sql.Tx) error {
		// Set all releases for this project to not active
		if _, err := tx.Exec("UPDATE releases SET is_active = 0 WHERE project_id = ?", projectID); err != nil {
			return err
		}
		// Set selected release to active
		if _, err := tx.Exec("UPDATE releases SET is_active = 1 WHERE id = ?", releaseID); err != nil {
			return err
		}
		// Update project active release pointer
		if _, err := tx.Exec("UPDATE projects SET active_release_id = ?, updated_at = ? WHERE id = ?", releaseID, time.Now().UTC(), projectID); err != nil {
			return err
		}
		return nil
	})
}

func (d *DB) GetReleaseByID(id string) (*Release, error) {
	rel := &Release{}
	query := `SELECT id, project_id, server_id, commit_hash, artifact_checksum, artifact_path, is_active, deployed_at
	          FROM releases WHERE id = ?`
	err := d.QueryRow(query, id).Scan(
		&rel.ID, &rel.ProjectID, &rel.ServerID, &rel.CommitHash, &rel.ArtifactChecksum, &rel.ArtifactPath, &rel.IsActive, &rel.DeployedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rel, nil
}

func (d *DB) ListReleases(projectID string) ([]*Release, error) {
	query := `SELECT id, project_id, server_id, commit_hash, artifact_checksum, artifact_path, is_active, deployed_at
	          FROM releases WHERE project_id = ? ORDER BY deployed_at DESC`
	rows, err := d.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	releases := make([]*Release, 0)
	for rows.Next() {
		rel := &Release{}
		if err := rows.Scan(
			&rel.ID, &rel.ProjectID, &rel.ServerID, &rel.CommitHash, &rel.ArtifactChecksum, &rel.ArtifactPath, &rel.IsActive, &rel.DeployedAt,
		); err != nil {
			return nil, err
		}
		releases = append(releases, rel)
	}
	return releases, rows.Err()
}

// --- Deployment Logs Repository ---

func (d *DB) AppendDeploymentLog(deploymentID, level, stage, message string) error {
	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO deployment_logs (deployment_id, level, stage, message, created_at)
		          VALUES (?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, deploymentID, level, stage, message, time.Now().UTC())
		return err
	})
}

func (d *DB) GetDeploymentLogs(deploymentID string) ([]*DeploymentLog, error) {
	query := `SELECT id, deployment_id, level, stage, message, created_at
	          FROM deployment_logs WHERE deployment_id = ? ORDER BY id ASC`
	rows, err := d.Query(query, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*DeploymentLog, 0)
	for rows.Next() {
		l := &DeploymentLog{}
		if err := rows.Scan(&l.ID, &l.DeploymentID, &l.Level, &l.Stage, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// --- Central Job Queue Repository ---

func (d *DB) CreateJob(serverID string, jobType JobType, payloadJSON string) (*Job, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	err := d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO jobs (id, server_id, type, payload_json, status, created_at, updated_at)
		          VALUES (?, ?, ?, ?, 'pending', ?, ?)`
		_, err := tx.Exec(query, id, serverID, string(jobType), payloadJSON, now, now)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Job{
		ID:          id,
		ServerID:    serverID,
		Type:        jobType,
		PayloadJSON: payloadJSON,
		Status:      JobPending,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// LeaseJob attempts to atomically claim the next pending job for a given server.
func (d *DB) LeaseJob(serverID string, leaseDuration time.Duration) (*Job, error) {
	var job *Job
	now := time.Now().UTC()
	leaseToken := uuid.New().String()

	err := d.Transact(func(tx *sql.Tx) error {
		// First recover any expired leases for this server
		expireThreshold := now.Add(-leaseDuration)
		_, err := tx.Exec(`UPDATE jobs SET status = 'pending', lease_token = '', leased_at = NULL, updated_at = ?
		                   WHERE server_id = ? AND status = 'leased' AND leased_at < ?`, now, serverID, expireThreshold)
		if err != nil {
			return fmt.Errorf("recovering expired leases: %w", err)
		}

		// Find the oldest pending job with attempts < max_attempts
		query := `SELECT id, server_id, type, payload_json, status, attempts, max_attempts, created_at, updated_at
		          FROM jobs WHERE server_id = ? AND status = 'pending' AND attempts < max_attempts
		          ORDER BY created_at ASC LIMIT 1`
		row := tx.QueryRow(query, serverID)

		j := &Job{}
		err = row.Scan(&j.ID, &j.ServerID, &j.Type, &j.PayloadJSON, &j.Status, &j.Attempts, &j.MaxAttempts, &j.CreatedAt, &j.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // No pending jobs available
		}
		if err != nil {
			return err
		}

		// Lease this job
		j.Attempts++
		j.Status = JobLeased
		j.LeaseToken = leaseToken
		j.LeasedAt = &now
		j.UpdatedAt = now

		updateQuery := `UPDATE jobs SET status = 'leased', lease_token = ?, leased_at = ?, attempts = ?, updated_at = ?
		                WHERE id = ?`
		_, err = tx.Exec(updateQuery, j.LeaseToken, j.LeasedAt, j.Attempts, j.UpdatedAt, j.ID)
		if err != nil {
			return err
		}

		job = j
		return nil
	})

	return job, err
}

func (d *DB) CompleteJob(jobID, leaseToken string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		res, err := tx.Exec("UPDATE jobs SET status = 'completed', updated_at = ? WHERE id = ? AND lease_token = ?", now, jobID, leaseToken)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("job not found or invalid lease token")
		}
		return nil
	})
}

func (d *DB) FailJob(jobID, leaseToken, errorMessage string) error {
	now := time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE jobs SET status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'pending' END,
		          error_message = ?, updated_at = ?, lease_token = '' WHERE id = ? AND lease_token = ?`
		_, err := tx.Exec(query, errorMessage, now, jobID, leaseToken)
		return err
	})
}

// --- Previews Repository ---

func (d *DB) CreatePreview(prev *Preview) error {
	if prev.ID == "" {
		prev.ID = uuid.New().String()
	}
	prev.CreatedAt = time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO previews (id, project_id, server_id, pr_number, branch, commit_hash, domain, status, release_id, expires_at, created_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query,
			prev.ID, prev.ProjectID, prev.ServerID, prev.PRNumber, prev.Branch, prev.CommitHash,
			prev.Domain, string(prev.Status), prev.ReleaseID, prev.ExpiresAt, prev.CreatedAt,
		)
		return err
	})
}

func (d *DB) GetPreviewByID(id string) (*Preview, error) {
	prev := &Preview{}
	query := `SELECT p.id, p.project_id, prj.name as project_name, p.server_id, p.pr_number, p.branch,
	                 p.commit_hash, p.domain, p.status, p.release_id, p.expires_at, p.created_at
	          FROM previews p
	          JOIN projects prj ON p.project_id = prj.id
	          WHERE p.id = ?`
	err := d.QueryRow(query, id).Scan(
		&prev.ID, &prev.ProjectID, &prev.ProjectName, &prev.ServerID, &prev.PRNumber, &prev.Branch,
		&prev.CommitHash, &prev.Domain, &prev.Status, &prev.ReleaseID, &prev.ExpiresAt, &prev.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return prev, nil
}

func (d *DB) ListPreviews(projectID string) ([]*Preview, error) {
	var query string
	var args []any

	if projectID != "" {
		query = `SELECT p.id, p.project_id, prj.name as project_name, p.server_id, p.pr_number, p.branch,
		                p.commit_hash, p.domain, p.status, p.release_id, p.expires_at, p.created_at
		         FROM previews p
		         JOIN projects prj ON p.project_id = prj.id
		         WHERE p.project_id = ?
		         ORDER BY p.created_at DESC`
		args = []any{projectID}
	} else {
		query = `SELECT p.id, p.project_id, prj.name as project_name, p.server_id, p.pr_number, p.branch,
		                p.commit_hash, p.domain, p.status, p.release_id, p.expires_at, p.created_at
		         FROM previews p
		         JOIN projects prj ON p.project_id = prj.id
		         ORDER BY p.created_at DESC`
	}

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	previews := make([]*Preview, 0)
	for rows.Next() {
		prev := &Preview{}
		if err := rows.Scan(
			&prev.ID, &prev.ProjectID, &prev.ProjectName, &prev.ServerID, &prev.PRNumber, &prev.Branch,
			&prev.CommitHash, &prev.Domain, &prev.Status, &prev.ReleaseID, &prev.ExpiresAt, &prev.CreatedAt,
		); err != nil {
			return nil, err
		}
		previews = append(previews, prev)
	}
	return previews, rows.Err()
}

func (d *DB) GetExpiredActivePreviews() ([]*Preview, error) {
	query := `SELECT p.id, p.project_id, prj.name as project_name, p.server_id, p.pr_number, p.branch,
	                 p.commit_hash, p.domain, p.status, p.release_id, p.expires_at, p.created_at
	          FROM previews p
	          JOIN projects prj ON p.project_id = prj.id
	          WHERE p.status = 'active' AND p.expires_at <= ?`
	rows, err := d.Query(query, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expired := make([]*Preview, 0)
	for rows.Next() {
		prev := &Preview{}
		if err := rows.Scan(
			&prev.ID, &prev.ProjectID, &prev.ProjectName, &prev.ServerID, &prev.PRNumber, &prev.Branch,
			&prev.CommitHash, &prev.Domain, &prev.Status, &prev.ReleaseID, &prev.ExpiresAt, &prev.CreatedAt,
		); err != nil {
			return nil, err
		}
		expired = append(expired, prev)
	}
	return expired, rows.Err()
}

func (d *DB) UpdatePreviewStatus(id string, status PreviewStatus) error {
	return d.Transact(func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE previews SET status = ? WHERE id = ?", string(status), id)
		return err
	})
}

// --- Webhook Deduplication Repository ---

func (d *DB) RecordWebhookEvent(eventID, source, eventType, payloadHash string) (bool, error) {
	var isNew bool
	err := d.Transact(func(tx *sql.Tx) error {
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM webhook_events WHERE event_id = ?)", eventID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			isNew = false
			return nil
		}

		id := uuid.New().String()
		_, err = tx.Exec(`INSERT INTO webhook_events (id, event_id, source, event_type, payload_hash, status, created_at)
		                  VALUES (?, ?, ?, ?, ?, 'processed', ?)`, id, eventID, source, eventType, payloadHash, time.Now().UTC())
		if err != nil {
			return err
		}
		isNew = true
		return nil
	})
	return isNew, err
}

// --- Audit Log Repository ---

func (d *DB) RecordAuditLog(userID, userEmail, action, resourceType, resourceID, detailsJSON, ipAddress string) error {
	id := uuid.New().String()
	return d.Transact(func(tx *sql.Tx) error {
		query := `INSERT INTO audit_logs (id, user_id, user_email, action, resource_type, resource_id, details_json, ip_address, created_at)
		          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, id, userID, userEmail, action, resourceType, resourceID, detailsJSON, ipAddress, time.Now().UTC())
		return err
	})
}

func (d *DB) ListAuditLogs(limit, offset int) ([]*AuditLog, error) {
	query := `SELECT id, user_id, user_email, action, resource_type, resource_id, details_json, ip_address, created_at
	          FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := d.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*AuditLog, 0)
	for rows.Next() {
		l := &AuditLog{}
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.UserEmail, &l.Action, &l.ResourceType, &l.ResourceID, &l.DetailsJSON, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// --- Notification Settings Repository ---

func (d *DB) GetNotificationSettings() (*NotificationSettings, error) {
	ns := &NotificationSettings{}
	query := `SELECT id, ntfy_url, topic, auth_token, events_json, enabled, created_at, updated_at
	          FROM notification_settings LIMIT 1`
	err := d.QueryRow(query).Scan(&ns.ID, &ns.NtfyURL, &ns.Topic, &ns.AuthToken, &ns.EventsJSON, &ns.Enabled, &ns.CreatedAt, &ns.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Initialize default record
		id := uuid.New().String()
		now := time.Now().UTC()
		err = d.Transact(func(tx *sql.Tx) error {
			_, err := tx.Exec(`INSERT INTO notification_settings (id, ntfy_url, topic, auth_token, events_json, enabled, created_at, updated_at)
			                   VALUES (?, 'https://ntfy.sh', '', '', '["deploy_success","deploy_failed","server_offline","disk_warning"]', 0, ?, ?)`, id, now, now)
			return err
		})
		if err != nil {
			return nil, err
		}
		return d.GetNotificationSettings()
	}
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func (d *DB) UpdateNotificationSettings(ns *NotificationSettings) error {
	ns.UpdatedAt = time.Now().UTC()
	return d.Transact(func(tx *sql.Tx) error {
		query := `UPDATE notification_settings SET
		          ntfy_url = ?, topic = ?, auth_token = ?, events_json = ?, enabled = ?, updated_at = ?
		          WHERE id = ?`
		_, err := tx.Exec(query, ns.NtfyURL, ns.Topic, ns.AuthToken, ns.EventsJSON, ns.Enabled, ns.UpdatedAt, ns.ID)
		return err
	})
}
