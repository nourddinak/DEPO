package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"depo/internal/crypto"
	"depo/internal/db"
	"github.com/google/uuid"
)

type RollbackRequest struct {
	ReleaseID string `json:"release_id"`
}

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	deps, err := s.db.ListDeployments(projectID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list deployments")
		return
	}
	writeJSON(w, http.StatusOK, deps)
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dep, err := s.db.GetDeploymentByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}
	writeJSON(w, http.StatusOK, dep)
}

func (s *Server) handleGetDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := s.db.GetDeploymentLogs(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load deployment logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	releases, err := s.db.ListReleases(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list releases")
		return
	}
	writeJSON(w, http.StatusOK, releases)
}

func (s *Server) handleRollbackProject(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	projectID := r.PathValue("id")

	project, err := s.db.GetProjectByID(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	var req RollbackRequest
	if err := readJSON(r, &req); err != nil || req.ReleaseID == "" {
		writeError(w, http.StatusBadRequest, "Target release_id is required")
		return
	}

	targetRelease, err := s.db.GetReleaseByID(req.ReleaseID)
	if err != nil || targetRelease.ProjectID != project.ID {
		writeError(w, http.StatusBadRequest, "Target release does not exist for this project")
		return
	}

	deploymentID := uuid.New().String()
	dep := &db.Deployment{
		ID:         deploymentID,
		ProjectID:  project.ID,
		ServerID:   project.ServerID,
		ReleaseID:  &targetRelease.ID,
		CommitHash: targetRelease.CommitHash,
		CommitMsg:  fmt.Sprintf("Rollback to release %s", crypto.Short(targetRelease.ID, 8)),
		Branch:     project.Branch,
		Trigger:    "rollback",
		Status:     db.DeployQueued,
	}
	if err := s.db.CreateDeployment(dep); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create rollback deployment record")
		return
	}

	_ = s.db.AppendDeploymentLog(deploymentID, "info", "rollback",
		fmt.Sprintf("Rollback triggered by %s to release %s (%s)", user.Email, crypto.Short(targetRelease.ID, 8), crypto.Short(targetRelease.CommitHash, 8)))

	// Enqueue rollback job for destination agent
	payload := map[string]any{
		"project_id":     project.ID,
		"deployment_id":  deploymentID,
		"release_id":     targetRelease.ID,
		"deploy_dir":     project.DeployDir,
		"domain":         project.Domain,
		"caddy_template": "",
	}
	payloadBytes, _ := json.Marshal(payload)
	_, err = s.db.CreateJob(project.ServerID, db.JobRollback, string(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to enqueue rollback job")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "rollback_release", "project", project.ID,
		fmt.Sprintf(`{"release_id":"%s"}`, targetRelease.ID), getClientIP(r))

	s.sseHub.Broadcast("deployment_created", map[string]any{
		"deployment_id": deploymentID,
		"project_id":    project.ID,
		"status":        "queued",
		"trigger":       "rollback",
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"message":       "Rollback initiated",
		"deployment_id": deploymentID,
		"release_id":    targetRelease.ID,
	})
}
