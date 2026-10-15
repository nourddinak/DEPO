package db

import (
	"time"
)

type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
)

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	Role           UserRole  `json:"role"`
	TOTPSecret     string    `json:"-"`
	TOTPEnabled    bool      `json:"totp_enabled"`
	RecoveryCodes  string    `json:"-"` // JSON array of hashed recovery codes
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Invitation struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Role       UserRole   `json:"role"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ServerStatus string

const (
	ServerStatusPending  ServerStatus = "pending"
	ServerStatusOnline   ServerStatus = "online"
	ServerStatusOffline  ServerStatus = "offline"
	ServerStatusDegraded ServerStatus = "degraded"
)

type Server struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Host                string       `json:"host"`
	Status              ServerStatus `json:"status"`
	OS                  string       `json:"os"`
	AgentVersion        string       `json:"agent_version"`
	EnrollmentTokenHash string       `json:"-"`
	AuthTokenHash       string       `json:"-"`
	LastHeartbeatAt     *time.Time   `json:"last_heartbeat_at,omitempty"`
	CPUPct              float64      `json:"cpu_pct"`
	RAMPct              float64      `json:"ram_pct"`
	DiskPct             float64      `json:"disk_pct"`
	DiskFreeBytes       int64        `json:"disk_free_bytes"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

type Project struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	RepoURL           string    `json:"repo_url"`
	Branch            string    `json:"branch"`
	FrontendDir       string    `json:"frontend_dir"`
	BuildCmd          string    `json:"build_cmd"`
	OutputDir         string    `json:"output_dir"`
	ServerID          string    `json:"server_id"`
	ServerName        string    `json:"server_name,omitempty"`
	DeployDir         string    `json:"deploy_dir"`
	RunUser           string    `json:"run_user"`
	CaddyfilePath     string    `json:"caddyfile_path"`
	Domain            string    `json:"domain"`
	DeployTokenHash   string    `json:"-"`
	AutoDeploy        bool      `json:"auto_deploy"`
	PreviewEnabled    bool      `json:"preview_enabled"`
	PreviewBaseDomain string    `json:"preview_base_domain"`
	LogRetentionDays  int       `json:"log_retention_days"`
	ActiveReleaseID   *string   `json:"active_release_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type EnvVar struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	Key            string    `json:"key"`
	EncryptedValue string    `json:"-"`
	Value          string    `json:"value,omitempty"` // populated on read if authorized
	IsBuildTime    bool      `json:"is_build_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DeploymentStatus string

const (
	DeployQueued     DeploymentStatus = "queued"
	DeployBuilding   DeploymentStatus = "building"
	DeployUploading  DeploymentStatus = "uploading"
	DeployDelivering DeploymentStatus = "delivering"
	DeployActivating DeploymentStatus = "activating"
	DeploySuccess    DeploymentStatus = "success"
	DeployFailed     DeploymentStatus = "failed"
	DeployRolledBack DeploymentStatus = "rolled_back"
)

type Deployment struct {
	ID           string           `json:"id"`
	ProjectID    string           `json:"project_id"`
	ProjectName  string           `json:"project_name,omitempty"`
	ServerID     string           `json:"server_id"`
	ServerName   string           `json:"server_name,omitempty"`
	ReleaseID    *string          `json:"release_id,omitempty"`
	CommitHash   string           `json:"commit_hash"`
	CommitMsg    string           `json:"commit_msg"`
	Branch       string           `json:"branch"`
	Trigger      string           `json:"trigger"` // "webhook", "manual", "env_change", "rollback"
	Status       DeploymentStatus `json:"status"`
	ErrorMessage string           `json:"error_message,omitempty"`
	ArtifactID   string           `json:"artifact_id,omitempty"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

type Release struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	ServerID         string    `json:"server_id"`
	CommitHash       string    `json:"commit_hash"`
	ArtifactChecksum string    `json:"artifact_checksum"`
	ArtifactPath     string    `json:"-"`
	IsActive         bool      `json:"is_active"`
	DeployedAt       time.Time `json:"deployed_at"`
}

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobLeased    JobStatus = "leased"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

type JobType string

const (
	JobDeployRelease  JobType = "deploy_release"
	JobRollback       JobType = "rollback"
	JobCleanupPreview JobType = "cleanup_preview"
	JobRemoveProject  JobType = "remove_project"
)

type Job struct {
	ID           string     `json:"id"`
	ServerID     string     `json:"server_id"`
	Type         JobType    `json:"type"`
	PayloadJSON  string     `json:"payload_json"`
	Status       JobStatus  `json:"status"`
	LeaseToken   string     `json:"lease_token,omitempty"`
	LeasedAt     *time.Time `json:"leased_at,omitempty"`
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"max_attempts"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PreviewStatus string

const (
	PreviewActive    PreviewStatus = "active"
	PreviewExpired   PreviewStatus = "expired"
	PreviewDestroyed PreviewStatus = "destroyed"
)

type Preview struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	ProjectName string        `json:"project_name,omitempty"`
	ServerID    string        `json:"server_id"`
	PRNumber    int           `json:"pr_number"`
	Branch      string        `json:"branch"`
	CommitHash  string        `json:"commit_hash"`
	Domain      string        `json:"domain"`
	Status      PreviewStatus `json:"status"`
	ReleaseID   string        `json:"release_id"`
	ExpiresAt   time.Time     `json:"expires_at"`
	CreatedAt   time.Time     `json:"created_at"`
}

type WebhookEvent struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	Source      string    `json:"source"`
	EventType   string    `json:"event_type"`
	PayloadHash string    `json:"payload_hash"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	DetailsJSON  string    `json:"details_json"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
}

type DeploymentLog struct {
	ID           int64     `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Level        string    `json:"level"`
	Stage        string    `json:"stage"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

type NotificationSettings struct {
	ID         string    `json:"id"`
	NtfyURL    string    `json:"ntfy_url"`
	Topic      string    `json:"topic"`
	AuthToken  string    `json:"-"`
	EventsJSON string    `json:"events_json"` // JSON array of event names
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
