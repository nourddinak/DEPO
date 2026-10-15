package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"depo/internal/crypto"
	"depo/internal/db"
	"github.com/google/uuid"
)

func (s *Server) handleArtifactUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Dual-mode authentication: Deploy Token OR GitHub OIDC JWT
	var project *db.Project
	var err error

	deployToken := r.Header.Get("X-Deploy-Token")
	authHeader := r.Header.Get("Authorization")

	if deployToken == "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenCandidate := strings.TrimPrefix(authHeader, "Bearer ")
		if strings.HasPrefix(tokenCandidate, "depo_dpl_") {
			deployToken = tokenCandidate
		} else if strings.Count(tokenCandidate, ".") == 2 {
			// GitHub OIDC JWT token
			claims, oidcErr := s.oidcValidator.ValidateOIDCToken(tokenCandidate, "")
			if oidcErr != nil {
				writeError(w, http.StatusUnauthorized, fmt.Sprintf("Invalid GitHub OIDC Token: %v", oidcErr))
				return
			}
			projectName := r.Header.Get("X-Project-Name")
			if projectName != "" {
				projects, _ := s.db.ListProjects()
				for _, p := range projects {
					if p.Name == projectName {
						project = p
						break
					}
				}
			}
			if project == nil {
				writeError(w, http.StatusNotFound, fmt.Sprintf("Project %s not found for OIDC claims", projectName))
				return
			}
			_ = claims // validated
		}
	}

	if project == nil && deployToken != "" {
		tokenHash := crypto.SHA256String(deployToken)
		project, err = s.db.GetProjectByDeployTokenHash(tokenHash)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid deploy token")
			return
		}
	}

	if project == nil {
		writeError(w, http.StatusUnauthorized, "Missing or invalid deployment credentials")
		return
	}

	// 2. Parse Multipart File
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil { // 100MB max memory
		writeError(w, http.StatusBadRequest, "Failed to parse multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing 'file' in multipart form data")
		return
	}
	defer file.Close()

	expectedChecksum := r.Header.Get("X-Artifact-Checksum")
	commitHash := r.Header.Get("X-Commit-Hash")
	commitMsg := r.Header.Get("X-Commit-Msg")
	branch := r.Header.Get("X-Branch")
	isPRStr := r.Header.Get("X-Is-PR")
	isPR := isPRStr == "true" || isPRStr == "1"
	prNumberStr := r.Header.Get("X-PR-Number")
	prNumber, _ := strconv.Atoi(prNumberStr)

	if commitHash == "" {
		commitHash = "unknown"
	}
	if branch == "" {
		branch = project.Branch
	}

	// 3. Save to disk outside web roots
	artifactID := uuid.New().String()
	artifactsDir := filepath.Join(s.dataDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0750); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize artifact storage")
		return
	}

	destPath := filepath.Join(artifactsDir, fmt.Sprintf("%s.tar.gz", artifactID))
	out, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create artifact file on server")
		return
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	if _, err := io.Copy(writer, file); err != nil {
		_ = os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "Failed to stream artifact to disk")
		return
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if expectedChecksum != "" && actualChecksum != expectedChecksum {
		_ = os.Remove(destPath)
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Checksum verification failed: expected %s, got %s", expectedChecksum, actualChecksum))
		return
	}

	// 4. Create Release Record
	releaseID := uuid.New().String()
	release := &db.Release{
		ID:               releaseID,
		ProjectID:        project.ID,
		ServerID:         project.ServerID,
		CommitHash:       commitHash,
		ArtifactChecksum: actualChecksum,
		ArtifactPath:     destPath,
		IsActive:         false,
	}
	if err := s.db.CreateRelease(release); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create release record")
		return
	}

	// 5. Handle Preview vs Production
	previewDomain := ""
	if isPR && project.PreviewEnabled {
		baseDomain := project.PreviewBaseDomain
		if baseDomain == "" {
			baseDomain = "preview." + project.Domain
		}
		previewDomain = fmt.Sprintf("%s-pr%d.%s", project.Name, prNumber, baseDomain)
		preview := &db.Preview{
			ProjectID:  project.ID,
			ServerID:   project.ServerID,
			PRNumber:   prNumber,
			Branch:     branch,
			CommitHash: commitHash,
			Domain:     previewDomain,
			Status:     db.PreviewActive,
			ReleaseID:  releaseID,
			ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour),
		}
		_ = s.db.CreatePreview(preview)
	}

	// 6. Create Deployment Record
	deploymentID := uuid.New().String()
	trigger := "ci_build"
	if isPR {
		trigger = "pr_preview"
	}
	deployment := &db.Deployment{
		ID:           deploymentID,
		ProjectID:    project.ID,
		ServerID:     project.ServerID,
		ReleaseID:    &releaseID,
		CommitHash:   commitHash,
		CommitMsg:    commitMsg,
		Branch:       branch,
		Trigger:      trigger,
		Status:       db.DeployQueued,
		ArtifactID:   artifactID,
	}
	if err := s.db.CreateDeployment(deployment); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create deployment record")
		return
	}

	_ = s.db.AppendDeploymentLog(deploymentID, "info", "upload",
		fmt.Sprintf("Artifact uploaded (%d bytes, sha256: %s). Queued release %s for delivery to server.",
			header.Size, crypto.Short(actualChecksum, 12), crypto.Short(releaseID, 8)))

	// 7. Enqueue Job for the VPS Agent
	jobPayload := map[string]any{
		"project_id":        project.ID,
		"project_name":      project.Name,
		"deployment_id":     deploymentID,
		"release_id":        releaseID,
		"artifact_id":       artifactID,
		"artifact_checksum": actualChecksum,
		"deploy_dir":        project.DeployDir,
		"domain":            project.Domain,
		"caddy_template":    "", // Use default or project Caddyfile
		"is_preview":        isPR,
		"preview_domain":    previewDomain,
	}
	payloadBytes, _ := json.Marshal(jobPayload)
	_, err = s.db.CreateJob(project.ServerID, db.JobDeployRelease, string(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to enqueue deployment job")
		return
	}

	s.sseHub.Broadcast("deployment_created", map[string]any{
		"deployment_id": deploymentID,
		"project_id":    project.ID,
		"project_name":  project.Name,
		"release_id":    releaseID,
		"status":        "queued",
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"message":        "Artifact verified and deployment job queued",
		"artifact_id":    artifactID,
		"release_id":     releaseID,
		"deployment_id":  deploymentID,
		"preview_domain": previewDomain,
	})
}

func (s *Server) handleArtifactDownload(w http.ResponseWriter, r *http.Request) {
	_, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent credentials")
		return
	}

	id := r.PathValue("id")
	filePath := filepath.Join(s.dataDir, "artifacts", fmt.Sprintf("%s.tar.gz", id))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Artifact not found")
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=artifact-%s.tar.gz", id))
	http.ServeFile(w, r, filePath)
}
