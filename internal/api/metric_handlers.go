package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"depo/internal/notify"
)

type UpdateNotificationRequest struct {
	NtfyURL    string   `json:"ntfy_url"`
	Topic      string   `json:"topic"`
	AuthToken  string   `json:"auth_token,omitempty"`
	Events     []string `json:"events"`
	Enabled    bool     `json:"enabled"`
}

type CheckDomainRequest struct {
	Domain string `json:"domain"`
}

func (s *Server) handleGetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetNotificationSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load notification settings")
		return
	}

	var events []string
	_ = json.Unmarshal([]byte(settings.EventsJSON), &events)
	if events == nil {
		events = make([]string, 0)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":        settings.ID,
		"ntfy_url":  settings.NtfyURL,
		"topic":     settings.Topic,
		"has_token": settings.AuthToken != "",
		"events":    events,
		"enabled":   settings.Enabled,
	})
}

func (s *Server) handleUpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateNotificationRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	settings, err := s.db.GetNotificationSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load settings")
		return
	}

	settings.NtfyURL = req.NtfyURL
	settings.Topic = req.Topic
	if req.AuthToken != "" {
		settings.AuthToken = req.AuthToken
	}
	eventsBytes, _ := json.Marshal(req.Events)
	settings.EventsJSON = string(eventsBytes)
	settings.Enabled = req.Enabled

	if err := s.db.UpdateNotificationSettings(settings); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save notification settings")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Notification settings updated successfully"})
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetNotificationSettings()
	if err != nil || settings.Topic == "" {
		writeError(w, http.StatusBadRequest, "Notification topic is not configured")
		return
	}

	err = s.notifier.Dispatch(settings.NtfyURL, settings.Topic, settings.AuthToken,
		notify.EventDeploySuccess, "DEPO Test Notification", "This is a test notification from your self-hosted DEPO control panel.", 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to dispatch test notification: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Test notification dispatched successfully"})
}

func (s *Server) handleCheckDomain(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain query parameter is required")
		return
	}

	ips, err := net.LookupHost(domain)
	resolvedIPs := make([]string, 0)
	if err == nil {
		resolvedIPs = ips
	}

	cnames, _ := net.LookupCNAME(domain)

	writeJSON(w, http.StatusOK, map[string]any{
		"domain":       domain,
		"resolved_ips": resolvedIPs,
		"cname":        cnames,
		"is_routed":    len(resolvedIPs) > 0,
		"instructions": fmt.Sprintf("Configure a DNS 'A' record pointing '%s' to your target VPS public IP, or a 'CNAME' record to your preview host.", domain),
	})
}
