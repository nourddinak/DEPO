package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"depo/internal/crypto"
	"depo/internal/db"
	"depo/internal/notify"
)

type CreateServerRequest struct {
	Name string `json:"name"`
	Host string `json:"host,omitempty"`
}

type AgentEnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	AgentVersion    string `json:"agent_version"`
}

type AgentHeartbeatRequest struct {
	CPUPct        float64 `json:"cpu_pct"`
	RAMPct        float64 `json:"ram_pct"`
	DiskPct       float64 `json:"disk_pct"`
	DiskFreeBytes int64   `json:"disk_free_bytes"`
}

type JobProgressRequest struct {
	DeploymentID string `json:"deployment_id"`
	Stage        string `json:"stage"`
	Level        string `json:"level"`
	Message      string `json:"message"`
}

type JobCompleteRequest struct {
	LeaseToken   string `json:"lease_token"`
	DeploymentID string `json:"deployment_id"`
	ReleaseID    string `json:"release_id"`
}

type JobFailRequest struct {
	LeaseToken   string `json:"lease_token"`
	DeploymentID string `json:"deployment_id"`
	ErrorMessage string `json:"error_message"`
}

// --- Dashboard Server Handlers ---

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.db.ListServers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list servers")
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	var req CreateServerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Server name is required")
		return
	}

	rawToken, tokenHash, err := crypto.GenerateEnrollmentToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate enrollment token")
		return
	}

	srv, err := s.db.CreateServer(name, req.Host, "linux", tokenHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Server with this name already exists")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "create_server", "server", srv.ID, fmt.Sprintf(`{"name":"%s"}`, name), getClientIP(r))

	installCmd := fmt.Sprintf("curl -sSL %s/install-agent.sh | sudo bash -s -- --server %s --token %s",
		s.externalURL, s.externalURL, rawToken)

	writeJSON(w, http.StatusCreated, map[string]any{
		"server":           srv,
		"enrollment_token": rawToken,
		"install_command":  installCmd,
	})
}

func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	srv, err := s.db.GetServerByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	writeJSON(w, http.StatusOK, srv)
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	id := r.PathValue("id")

	srv, err := s.db.GetServerByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}

	if err := s.db.DeleteServer(id); err != nil {
		writeError(w, http.StatusBadRequest, "Cannot delete server: ensure no projects or deployments are associated with it")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "delete_server", "server", id, fmt.Sprintf(`{"name":"%s"}`, srv.Name), getClientIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Server deleted successfully"})
}

// --- Agent-Facing Handlers ---

func (s *Server) handleAgentEnroll(w http.ResponseWriter, r *http.Request) {
	var req AgentEnrollRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid enrollment payload")
		return
	}

	tokenHash := crypto.SHA256String(strings.TrimSpace(req.EnrollmentToken))
	srv, err := s.db.GetServerByEnrollmentTokenHash(tokenHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid or already used enrollment token")
		return
	}

	rawAuthToken, authHash, err := crypto.GenerateAgentAuthToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate agent auth token")
		return
	}

	if err := s.db.CompleteServerEnrollment(srv.ID, authHash, req.AgentVersion); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to complete server enrollment")
		return
	}

	s.sseHub.Broadcast("server_enrolled", map[string]any{
		"server_id": srv.ID,
		"name":      srv.Name,
		"status":    "online",
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"server_id":  srv.ID,
		"auth_token": rawAuthToken,
		"message":    "Enrollment successful",
	})
}

// AuthenticateAgent resolves the server from Bearer depo_agt_...
func (s *Server) authenticateAgent(r *http.Request) (*db.Server, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing or invalid authorization header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	tokenHash := crypto.SHA256String(token)
	return s.db.GetServerByAuthTokenHash(tokenHash)
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	srv, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent")
		return
	}

	var req AgentHeartbeatRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid heartbeat payload")
		return
	}

	if err := s.db.UpdateServerHeartbeat(srv.ID, req.CPUPct, req.RAMPct, req.DiskPct, req.DiskFreeBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to record heartbeat")
		return
	}

	// High disk warning trigger
	if req.DiskPct > 90.0 {
		go func() {
			ns, err := s.db.GetNotificationSettings()
			if err == nil && ns.Enabled {
				_ = s.notifier.Dispatch(ns.NtfyURL, ns.Topic, ns.AuthToken, notify.EventDiskWarning,
					fmt.Sprintf("Disk Alert: %s (%.1f%% used)", srv.Name, req.DiskPct),
					fmt.Sprintf("Server %s is running low on disk space: %.1f%% used with only %d MB free.",
						srv.Name, req.DiskPct, req.DiskFreeBytes/(1024*1024)), 4)
			}
		}()
	}

	s.sseHub.Broadcast("server_metrics", map[string]any{
		"server_id":       srv.ID,
		"status":          "online",
		"cpu_pct":         req.CPUPct,
		"ram_pct":         req.RAMPct,
		"disk_pct":        req.DiskPct,
		"disk_free_bytes": req.DiskFreeBytes,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAgentPollJobs implements long-polling for jobs
func (s *Server) handleAgentPollJobs(w http.ResponseWriter, r *http.Request) {
	srv, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent")
		return
	}

	timeout := 25 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		// Attempt to claim job with a 90s lease
		job, err := s.db.LeaseJob(srv.ID, 90*time.Second)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to query job queue")
			return
		}

		if job != nil {
			// Found job!
			var payload map[string]any
			_ = json.Unmarshal([]byte(job.PayloadJSON), &payload)

			writeJSON(w, http.StatusOK, map[string]any{
				"job_id":      job.ID,
				"type":        job.Type,
				"lease_token": job.LeaseToken,
				"payload":     payload,
			})
			return
		}

		// Check if request deadline or client canceled
		if time.Now().After(deadline) || r.Context().Err() != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"job_id": nil,
			})
			return
		}

		// Sleep briefly before polling again in this long-poll request
		time.Sleep(1 * time.Second)
	}
}

func (s *Server) handleAgentJobProgress(w http.ResponseWriter, r *http.Request) {
	_, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent")
		return
	}

	var req JobProgressRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid progress payload")
		return
	}

	if req.DeploymentID != "" {
		_ = s.db.AppendDeploymentLog(req.DeploymentID, req.Level, req.Stage, req.Message)
		s.sseHub.Broadcast("deployment_log", map[string]any{
			"deployment_id": req.DeploymentID,
			"stage":         req.Stage,
			"level":         req.Level,
			"message":       req.Message,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAgentJobComplete(w http.ResponseWriter, r *http.Request) {
	_, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent")
		return
	}

	jobID := r.PathValue("id")
	var req JobCompleteRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := s.db.CompleteJob(jobID, req.LeaseToken); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.DeploymentID != "" {
		_ = s.db.UpdateDeploymentStatus(req.DeploymentID, db.DeploySuccess, &req.ReleaseID, "")
		dep, _ := s.db.GetDeploymentByID(req.DeploymentID)
		if dep != nil {
			_ = s.db.SetActiveRelease(dep.ProjectID, req.ReleaseID)
		}

		s.sseHub.Broadcast("deployment_updated", map[string]any{
			"deployment_id": req.DeploymentID,
			"status":        "success",
			"release_id":    req.ReleaseID,
		})

		// Send ntfy notification
		go func() {
			ns, err := s.db.GetNotificationSettings()
			if err == nil && ns.Enabled && dep != nil {
				_ = s.notifier.Dispatch(ns.NtfyURL, ns.Topic, ns.AuthToken, notify.EventDeploySuccess,
					fmt.Sprintf("Deployment Successful: %s", dep.ProjectName),
					fmt.Sprintf("Release %s for project %s (%s) was deployed successfully.", crypto.Short(req.ReleaseID, 8), dep.ProjectName, dep.Branch), 2)
			}
		}()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (s *Server) handleAgentJobFail(w http.ResponseWriter, r *http.Request) {
	_, err := s.authenticateAgent(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized agent")
		return
	}

	jobID := r.PathValue("id")
	var req JobFailRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := s.db.FailJob(jobID, req.LeaseToken, req.ErrorMessage); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.DeploymentID != "" {
		_ = s.db.UpdateDeploymentStatus(req.DeploymentID, db.DeployFailed, nil, req.ErrorMessage)
		_ = s.db.AppendDeploymentLog(req.DeploymentID, "error", "execution", req.ErrorMessage)

		s.sseHub.Broadcast("deployment_updated", map[string]any{
			"deployment_id": req.DeploymentID,
			"status":        "failed",
			"error":         req.ErrorMessage,
		})

		// Send ntfy notification
		go func() {
			dep, _ := s.db.GetDeploymentByID(req.DeploymentID)
			ns, err := s.db.GetNotificationSettings()
			if err == nil && ns.Enabled && dep != nil {
				_ = s.notifier.Dispatch(ns.NtfyURL, ns.Topic, ns.AuthToken, notify.EventDeployFailed,
					fmt.Sprintf("Deployment Failed: %s", dep.ProjectName),
					fmt.Sprintf("Deployment for project %s (%s) failed: %s", dep.ProjectName, dep.Branch, req.ErrorMessage), 4)
			}
		}()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}
