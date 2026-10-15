package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook HMAC signature")
	ErrMissingSignature = errors.New("missing X-Hub-Signature-256 header")
)

type PushEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"` // Commit SHA
	Before     string `json:"before"`
	Repository struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	HeadCommit struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
}

type PullRequestEvent struct {
	Action      string `json:"action"` // "opened", "synchronize", "closed", "reopened"
	Number      int    `json:"number"`
	PullRequest struct {
		ID        int64  `json:"id"`
		Number    int    `json:"number"`
		State     string `json:"state"`
		Title     string `json:"title"`
		Head      struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}

// VerifyWebhookSignature verifies GitHub's X-Hub-Signature-256 HMAC
func VerifyWebhookSignature(secret string, r *http.Request) ([]byte, error) {
	sigHeader := r.Header.Get("X-Hub-Signature-256")
	if sigHeader == "" {
		return nil, ErrMissingSignature
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	if secret == "" {
		// If secret not configured in local testing, return body
		return body, nil
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	actualMAC := strings.TrimPrefix(sigHeader, "sha256=")
	if !hmac.Equal([]byte(expectedMAC), []byte(actualMAC)) {
		return nil, ErrInvalidSignature
	}

	return body, nil
}

// ParsePushPayload parses a push event payload
func ParsePushPayload(payload []byte) (*PushEvent, error) {
	var event PushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ParsePullRequestPayload parses a PR event payload
func ParsePullRequestPayload(payload []byte) (*PullRequestEvent, error) {
	var event PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
