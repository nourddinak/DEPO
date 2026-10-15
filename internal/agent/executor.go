package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Executor struct {
	client       *Client
	caddyManager *CaddyManager
	healthCheck  *HealthChecker
	dataDir      string
	projectLocks map[string]*sync.Mutex
	locksMu      sync.Mutex
}

func NewExecutor(client *Client, caddyManager *CaddyManager, dataDir string) *Executor {
	return &Executor{
		client:       client,
		caddyManager: caddyManager,
		healthCheck:  NewHealthChecker(),
		dataDir:      dataDir,
		projectLocks: make(map[string]*sync.Mutex),
	}
}

func (e *Executor) getProjectLock(projectID string) *sync.Mutex {
	e.locksMu.Lock()
	defer e.locksMu.Unlock()
	lock, exists := e.projectLocks[projectID]
	if !exists {
		lock = &sync.Mutex{}
		e.projectLocks[projectID] = lock
	}
	return lock
}

type DeployPayload struct {
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name"`
	DeploymentID     string `json:"deployment_id"`
	ReleaseID        string `json:"release_id"`
	ArtifactID       string `json:"artifact_id"`
	ArtifactChecksum string `json:"artifact_checksum"`
	DeployDir        string `json:"deploy_dir"`
	Domain           string `json:"domain"`
	CaddyTemplate    string `json:"caddy_template"`
	IsPreview        bool   `json:"is_preview"`
	PreviewDomain    string `json:"preview_domain"`
}

type RollbackPayload struct {
	ProjectID     string `json:"project_id"`
	DeploymentID  string `json:"deployment_id"`
	ReleaseID     string `json:"release_id"`
	DeployDir     string `json:"deploy_dir"`
	Domain        string `json:"domain"`
	CaddyTemplate string `json:"caddy_template"`
}

type CleanupPreviewPayload struct {
	ProjectID string `json:"project_id"`
	PreviewID string `json:"preview_id"`
	Domain    string `json:"domain"`
	DeployDir string `json:"deploy_dir"`
}

func (e *Executor) ExecuteJob(claim *JobClaimResponse) error {
	payloadBytes, err := json.Marshal(claim.Payload)
	if err != nil {
		return err
	}

	switch claim.Type {
	case "deploy_release":
		var p DeployPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return err
		}
		return e.executeDeploy(claim.JobID, claim.LeaseToken, &p)

	case "rollback":
		var p RollbackPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return err
		}
		return e.executeRollback(claim.JobID, claim.LeaseToken, &p)

	case "cleanup_preview":
		var p CleanupPreviewPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return err
		}
		return e.executeCleanupPreview(claim.JobID, claim.LeaseToken, &p)

	default:
		return fmt.Errorf("unknown job type: %s", claim.Type)
	}
}

func (e *Executor) executeDeploy(jobID, leaseToken string, p *DeployPayload) error {
	lock := e.getProjectLock(p.ProjectID)
	lock.Lock()
	defer lock.Unlock()

	domain := p.Domain
	if p.IsPreview && p.PreviewDomain != "" {
		domain = p.PreviewDomain
	}

	baseDir := p.DeployDir
	if baseDir == "" {
		baseDir = "/var/www"
	}
	projectRoot := filepath.Join(baseDir, p.ProjectID)
	releasesDir := filepath.Join(projectRoot, "releases")
	targetReleaseDir := filepath.Join(releasesDir, p.ReleaseID)
	currentSymlink := filepath.Join(projectRoot, "current")
	projectCaddyConf := filepath.Join(projectRoot, "caddy.conf")

	// Check if an existing active release exists before deployment (for safe rollback)
	var prevTarget string
	if target, err := os.Readlink(currentSymlink); err == nil {
		prevTarget = target
	}

	// 1. Download artifact
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "download", "info", "Downloading release artifact from central server...")
	tempArtifactFile := filepath.Join(e.dataDir, fmt.Sprintf("artifact-%s.tar.gz", p.ReleaseID))
	_ = os.MkdirAll(e.dataDir, 0755)
	defer os.Remove(tempArtifactFile)

	if err := e.client.DownloadArtifact(p.ArtifactID, tempArtifactFile, p.ArtifactChecksum); err != nil {
		errMsg := fmt.Sprintf("Failed to download artifact: %v", err)
		_ = e.client.ReportProgress(jobID, p.DeploymentID, "download", "error", errMsg)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 2. Extract into isolated release directory
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "extract", "info", fmt.Sprintf("Extracting to %s...", targetReleaseDir))
	if err := os.MkdirAll(targetReleaseDir, 0755); err != nil {
		errMsg := fmt.Sprintf("Failed to create release directory: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	if err := ExtractArchive(tempArtifactFile, targetReleaseDir); err != nil {
		_ = os.RemoveAll(targetReleaseDir)
		errMsg := fmt.Sprintf("Failed to extract artifact: %v", err)
		_ = e.client.ReportProgress(jobID, p.DeploymentID, "extract", "error", errMsg)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 3. Configure Caddy
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "caddy_config", "info", "Configuring Caddy virtual host and directory bindings...")
	if err := e.caddyManager.GenerateProjectCaddyfile(p.CaddyTemplate, domain, currentSymlink, projectCaddyConf); err != nil {
		_ = os.RemoveAll(targetReleaseDir)
		errMsg := fmt.Sprintf("Failed to generate project Caddyfile: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	if err := e.caddyManager.EnsureMainImport(projectCaddyConf); err != nil {
		_ = os.RemoveAll(targetReleaseDir)
		errMsg := fmt.Sprintf("Failed to register import in main Caddyfile: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 4. Atomic symlink replacement
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "activate", "info", "Executing atomic same-filesystem symlink swap...")
	tmpSymlink := filepath.Join(projectRoot, fmt.Sprintf("current_tmp_%d", time.Now().UnixNano()))

	// Create new symlink pointing to target release
	relativeTarget := filepath.Join("releases", p.ReleaseID)
	if err := os.Symlink(relativeTarget, tmpSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		errMsg := fmt.Sprintf("Failed to create temporary symlink: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// Atomic rename replaces current symlink instantly
	if err := swapSymlink(tmpSymlink, currentSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		errMsg := fmt.Sprintf("Failed to atomically swap symlink: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 5. Validate Caddy syntax
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "caddy_validate", "info", "Validating effective Caddy configuration...")
	if err := e.caddyManager.ValidateConfig(); err != nil {
		// Rollback symlink and Caddy config immediately
		_ = e.caddyManager.Revert()
		if prevTarget != "" {
			_ = os.Symlink(prevTarget, tmpSymlink)
			_ = os.Rename(tmpSymlink, currentSymlink)
		}
		errMsg := fmt.Sprintf("Caddy configuration validation failed: %v", err)
		_ = e.client.ReportProgress(jobID, p.DeploymentID, "caddy_validate", "error", errMsg)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 6. Reload Caddy
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "caddy_reload", "info", "Gracefully reloading Caddy...")
	if err := e.caddyManager.Reload(); err != nil {
		// Rollback
		_ = e.caddyManager.Revert()
		if prevTarget != "" {
			_ = os.Symlink(prevTarget, tmpSymlink)
			_ = os.Rename(tmpSymlink, currentSymlink)
		}
		errMsg := fmt.Sprintf("Caddy reload failed: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 7. Post-activation health check
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "health_check", "info", "Performing local health check...")
	if err := e.healthCheck.CheckLocalDeployment(domain, 80); err != nil {
		// Health check failed! Rollback to prior release
		_ = e.caddyManager.Revert()
		if prevTarget != "" {
			_ = os.Symlink(prevTarget, tmpSymlink)
			_ = os.Rename(tmpSymlink, currentSymlink)
			_ = e.caddyManager.Reload()
		}
		errMsg := fmt.Sprintf("Deployment failed health check: %v. Restored previous active release.", err)
		_ = e.client.ReportProgress(jobID, p.DeploymentID, "health_check", "error", errMsg)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// 8. Completed successfully!
	_ = e.client.ReportProgress(jobID, p.DeploymentID, "active", "info", fmt.Sprintf("Release %s is live on %s!", safeShort(p.ReleaseID, 8), domain))
	return e.client.CompleteJob(jobID, leaseToken, p.DeploymentID, p.ReleaseID)
}

func (e *Executor) executeRollback(jobID, leaseToken string, p *RollbackPayload) error {
	lock := e.getProjectLock(p.ProjectID)
	lock.Lock()
	defer lock.Unlock()

	baseDir := p.DeployDir
	if baseDir == "" {
		baseDir = "/var/www"
	}
	projectRoot := filepath.Join(baseDir, p.ProjectID)
	targetReleaseDir := filepath.Join(projectRoot, "releases", p.ReleaseID)
	currentSymlink := filepath.Join(projectRoot, "current")

	if _, err := os.Stat(targetReleaseDir); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("Target rollback release %s does not exist on disk", p.ReleaseID)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// Swap symlink to target release
	tmpSymlink := filepath.Join(projectRoot, fmt.Sprintf("current_tmp_%d", time.Now().UnixNano()))
	relativeTarget := filepath.Join("releases", p.ReleaseID)

	if err := os.Symlink(relativeTarget, tmpSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		errMsg := fmt.Sprintf("Failed to create rollback symlink: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	if err := swapSymlink(tmpSymlink, currentSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		errMsg := fmt.Sprintf("Failed to swap rollback symlink: %v", err)
		_ = e.client.FailJob(jobID, leaseToken, p.DeploymentID, errMsg)
		return err
	}

	// Reload Caddy
	_ = e.caddyManager.Reload()

	_ = e.client.ReportProgress(jobID, p.DeploymentID, "rollback", "info", fmt.Sprintf("Rolled back to release %s", safeShort(p.ReleaseID, 8)))
	return e.client.CompleteJob(jobID, leaseToken, p.DeploymentID, p.ReleaseID)
}

func safeShort(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// swapSymlink atomically replaces currentSymlink with tmpSymlink.
// On POSIX/Linux, rename over an existing symlink is atomic.
// On Windows, if destination exists, we remove it before renaming to avoid "Access is denied".
func swapSymlink(tmpSymlink, currentSymlink string) error {
	if runtime.GOOS == "windows" {
		_ = os.Remove(currentSymlink)
	}
	if err := os.Rename(tmpSymlink, currentSymlink); err != nil {
		_ = os.Remove(currentSymlink)
		return os.Rename(tmpSymlink, currentSymlink)
	}
	return nil
}

func (e *Executor) executeCleanupPreview(jobID, leaseToken string, p *CleanupPreviewPayload) error {
	baseDir := p.DeployDir
	if baseDir == "" {
		baseDir = "/var/www"
	}
	previewRoot := filepath.Join(baseDir, p.ProjectID, "previews", p.PreviewID)
	previewCaddyConf := filepath.Join(previewRoot, "caddy.conf")

	// Remove import and reload Caddy
	_ = e.caddyManager.RemoveMainImport(previewCaddyConf)
	_ = e.caddyManager.Reload()

	// Delete preview directory
	_ = os.RemoveAll(previewRoot)

	return e.client.CompleteJob(jobID, leaseToken, "", "")
}
