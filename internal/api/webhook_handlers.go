package api

import (
	"fmt"
	"net/http"
	"strings"

	"depo/internal/crypto"
	"depo/internal/github"
)

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	deliveryID := r.Header.Get("X-GitHub-Delivery")
	if deliveryID == "" {
		writeError(w, http.StatusBadRequest, "Missing X-GitHub-Delivery header")
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		writeError(w, http.StatusBadRequest, "Missing X-GitHub-Event header")
		return
	}

	// Verify HMAC signature
	body, err := github.VerifyWebhookSignature(s.webhookSecret, r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("Webhook verification failed: %v", err))
		return
	}

	payloadHash := crypto.SHA256Bytes(body)

	// Deduplicate event
	isNew, err := s.db.RecordWebhookEvent(deliveryID, "github", eventType, payloadHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to log webhook event")
		return
	}
	if !isNew {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Event already processed (deduplicated)"})
		return
	}

	// Process event
	switch eventType {
	case "push":
		event, err := github.ParsePushPayload(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Failed to parse push payload")
			return
		}

		branch := strings.TrimPrefix(event.Ref, "refs/heads/")
		repoURL := event.Repository.CloneURL
		if repoURL == "" {
			repoURL = event.Repository.HTMLURL
		}

		// Find project watching this repo & branch
		project, err := s.db.GetProjectByRepoAndBranch(repoURL, branch)
		if err != nil || !project.AutoDeploy {
			writeJSON(w, http.StatusOK, map[string]string{
				"status": "ignored",
				"reason": "No matching auto-deploy project configured for this branch",
			})
			return
		}

		s.sseHub.Broadcast("webhook_received", map[string]any{
			"type":       "push",
			"project_id": project.ID,
			"branch":     branch,
			"commit":     event.After,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "matched",
			"project_id": project.ID,
			"branch":     branch,
		})

	case "pull_request":
		event, err := github.ParsePullRequestPayload(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Failed to parse pull request payload")
			return
		}

		s.sseHub.Broadcast("webhook_received", map[string]any{
			"type":      "pull_request",
			"pr_number": event.Number,
			"action":    event.Action,
			"branch":    event.PullRequest.Head.Ref,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "processed",
			"pr_number": event.Number,
			"action":    event.Action,
		})

	case "ping":
		writeJSON(w, http.StatusOK, map[string]string{"message": "Webhook ping acknowledged"})

	default:
		writeJSON(w, http.StatusOK, map[string]string{"message": "Event ignored"})
	}
}
