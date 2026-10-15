package notify

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type EventType string

const (
	EventDeploySuccess   EventType = "deploy_success"
	EventDeployFailed    EventType = "deploy_failed"
	EventServerOffline   EventType = "server_offline"
	EventServerRecovered EventType = "server_recovered"
	EventDiskWarning     EventType = "disk_warning"
	EventTLSProblem      EventType = "tls_problem"
	EventPreviewCleanup  EventType = "preview_cleanup"
)

type Dispatcher struct {
	client    *http.Client
	mu        sync.Mutex
	lastAlert map[string]time.Time
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastAlert: make(map[string]time.Time),
	}
}

// RedactSecrets replaces sensitive patterns (tokens, keys, passwords) with [REDACTED]
func RedactSecrets(msg string) string {
	redacted := msg
	prefixes := []string{"depo_sess_", "depo_enr_", "depo_agt_", "depo_dpl_", "depo_inv_", "ghp_", "github_pat_"}
	for _, p := range prefixes {
		idx := strings.Index(redacted, p)
		for idx != -1 {
			end := idx + len(p)
			for end < len(redacted) && (redacted[end] != ' ' && redacted[end] != '\n' && redacted[end] != '"' && redacted[end] != ',') {
				end++
			}
			redacted = redacted[:idx] + "[REDACTED]" + redacted[end:]
			idx = strings.Index(redacted, p)
		}
	}
	return redacted
}

// Dispatch sends a notification to ntfy with deduplication and secret redaction
func (d *Dispatcher) Dispatch(ntfyURL, topic, authToken string, event EventType, title, message string, priority int) error {
	if topic == "" {
		return nil // Notification topic not configured
	}

	dedupKey := fmt.Sprintf("%s:%s:%s", topic, event, title)

	d.mu.Lock()
	lastTime, exists := d.lastAlert[dedupKey]
	// Deduplicate identical alerts within 5 minutes
	if exists && time.Since(lastTime) < 5*time.Minute {
		d.mu.Unlock()
		return nil
	}
	d.lastAlert[dedupKey] = time.Now()
	d.mu.Unlock()

	safeMessage := RedactSecrets(message)
	safeTitle := RedactSecrets(title)

	url := strings.TrimRight(ntfyURL, "/") + "/" + topic

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(safeMessage))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}

	req.Header.Set("Title", safeTitle)
	req.Header.Set("Priority", fmt.Sprintf("%d", priority))
	req.Header.Set("Tags", string(event))

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("dispatching ntfy alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
