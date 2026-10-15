package api

import (
	"encoding/json"
	"net/http"

	"depo/internal/db"
)

func (s *Server) handleListPreviews(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	previews, err := s.db.ListPreviews(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list previews")
		return
	}
	writeJSON(w, http.StatusOK, previews)
}

func (s *Server) handleGetPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prev, err := s.db.GetPreviewByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Preview not found")
		return
	}
	writeJSON(w, http.StatusOK, prev)
}

func (s *Server) handleDestroyPreview(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	id := r.PathValue("id")

	prev, err := s.db.GetPreviewByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Preview not found")
		return
	}

	project, _ := s.db.GetProjectByID(prev.ProjectID)
	deployDir := "/var/www"
	if project != nil && project.DeployDir != "" {
		deployDir = project.DeployDir
	}

	// Enqueue cleanup job for the destination agent
	payload := map[string]any{
		"project_id": prev.ProjectID,
		"preview_id": prev.ID,
		"domain":     prev.Domain,
		"deploy_dir": deployDir,
	}
	payloadBytes, _ := json.Marshal(payload)
	_, _ = s.db.CreateJob(prev.ServerID, db.JobCleanupPreview, string(payloadBytes))

	_ = s.db.UpdatePreviewStatus(prev.ID, db.PreviewDestroyed)
	_ = s.db.RecordAuditLog(user.ID, user.Email, "destroy_preview", "preview", prev.ID, "", getClientIP(r))

	s.sseHub.Broadcast("preview_updated", map[string]any{
		"preview_id": prev.ID,
		"status":     "destroyed",
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Preview cleanup initiated"})
}
