package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	serverURL  string
	authToken  string
	httpClient *http.Client
}

type JobClaimResponse struct {
	JobID      string         `json:"job_id"`
	Type       string         `json:"type"`
	LeaseToken string         `json:"lease_token"`
	Payload    map[string]any `json:"payload"`
}

type APIResponseWrapper[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}

func NewClient(serverURL, authToken string) *Client {
	return &Client{
		serverURL: strings.TrimRight(serverURL, "/"),
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Accommodate long polling
		},
	}
}

func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

func (c *Client) Enroll(enrollmentToken, hostname, osName, version string) (serverID string, authToken string, err error) {
	url := fmt.Sprintf("%s/api/v1/agent/enroll", c.serverURL)
	payload := map[string]string{
		"enrollment_token": enrollmentToken,
		"hostname":        hostname,
		"os":              osName,
		"agent_version":    version,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("enroll request: %w", err)
	}
	defer resp.Body.Close()

	var res APIResponseWrapper[map[string]string]
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", fmt.Errorf("decoding enroll response: %w", err)
	}

	if !res.Success {
		return "", "", fmt.Errorf("enrollment failed: %s", res.Error)
	}

	return res.Data["server_id"], res.Data["auth_token"], nil
}

func (c *Client) Heartbeat(cpuPct, ramPct, diskPct float64, diskFreeBytes int64) error {
	url := fmt.Sprintf("%s/api/v1/agent/heartbeat", c.serverURL)
	payload := map[string]any{
		"cpu_pct":         cpuPct,
		"ram_pct":         ramPct,
		"disk_pct":        diskPct,
		"disk_free_bytes": diskFreeBytes,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) PollJobs(ctx context.Context) (*JobClaimResponse, error) {
	url := fmt.Sprintf("%s/api/v1/agent/jobs/poll", c.serverURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized agent credentials")
	}

	var res APIResponseWrapper[JobClaimResponse]
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if !res.Success {
		return nil, fmt.Errorf("job poll error: %s", res.Error)
	}

	if res.Data.JobID == "" {
		return nil, nil // No jobs waiting
	}

	return &res.Data, nil
}

func (c *Client) ReportProgress(jobID, deploymentID, stage, level, message string) error {
	url := fmt.Sprintf("%s/api/v1/agent/jobs/%s/progress", c.serverURL, jobID)
	payload := map[string]string{
		"deployment_id": deploymentID,
		"stage":         stage,
		"level":         level,
		"message":       message,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) CompleteJob(jobID, leaseToken, deploymentID, releaseID string) error {
	url := fmt.Sprintf("%s/api/v1/agent/jobs/%s/complete", c.serverURL, jobID)
	payload := map[string]string{
		"lease_token":   leaseToken,
		"deployment_id": deploymentID,
		"release_id":    releaseID,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) FailJob(jobID, leaseToken, deploymentID, errorMessage string) error {
	url := fmt.Sprintf("%s/api/v1/agent/jobs/%s/fail", c.serverURL, jobID)
	payload := map[string]string{
		"lease_token":   leaseToken,
		"deployment_id": deploymentID,
		"error_message": errorMessage,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) DownloadArtifact(artifactID, destPath, expectedSHA256 string) error {
	url := fmt.Sprintf("%s/api/v1/artifacts/%s/download", c.serverURL, artifactID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("artifact download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artifact download returned status: %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating destination artifact file: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("streaming artifact content: %w", err)
	}

	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if expectedSHA256 != "" && actualSHA256 != expectedSHA256 {
		_ = os.Remove(destPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}
