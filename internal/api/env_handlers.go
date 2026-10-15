package api

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"depo/internal/crypto"
)

type EnvItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsBuildTime bool   `json:"is_build_time"`
	Comment     string `json:"comment,omitempty"`
}

type UpdateEnvRequest struct {
	Variables     []EnvItem `json:"variables,omitempty"`
	RawContent    string    `json:"raw_content,omitempty"`
	TriggerDeploy bool      `json:"trigger_deploy"`
}

type ParseExampleRequest struct {
	Content string `json:"content"`
}

func parseEnvExample(content string) []EnvItem {
	items := make([]EnvItem, 0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	currentComment := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			currentComment = ""
			continue
		}
		if strings.HasPrefix(line, "#") {
			comment := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if currentComment != "" {
				currentComment += " " + comment
			} else {
				currentComment = comment
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			items = append(items, EnvItem{
				Key:         key,
				Value:       val,
				IsBuildTime: true,
				Comment:     currentComment,
			})
			currentComment = ""
		}
	}
	return items
}

func (s *Server) handleParseEnvExample(w http.ResponseWriter, r *http.Request) {
	var req ParseExampleRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	items := parseEnvExample(req.Content)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetProjectEnv(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	envVars, err := s.db.GetEnvVars(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load environment variables")
		return
	}

	items := make([]EnvItem, 0)
	rawLines := make([]string, 0)

	for _, ev := range envVars {
		decrypted, err := crypto.Decrypt(ev.EncryptedValue, s.masterKey)
		val := ""
		if err == nil {
			val = string(decrypted)
		}

		items = append(items, EnvItem{
			Key:         ev.Key,
			Value:       val,
			IsBuildTime: ev.IsBuildTime,
		})
		rawLines = append(rawLines, fmt.Sprintf("%s=%s", ev.Key, val))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"variables":   items,
		"raw_content": strings.Join(rawLines, "\n"),
		"warning":     "NOTE: All variables embedded into static JavaScript frontends at build time are publicly inspectable in user browsers. Do not put private backend secrets here.",
	})
}

func (s *Server) handleUpdateProjectEnv(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	projectID := r.PathValue("id")

	project, err := s.db.GetProjectByID(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	var req UpdateEnvRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var toSave []EnvItem
	if req.RawContent != "" {
		// Parse from multiline editor
		scanner := bufio.NewScanner(strings.NewReader(req.RawContent))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				toSave = append(toSave, EnvItem{
					Key:         strings.TrimSpace(parts[0]),
					Value:       strings.Trim(strings.TrimSpace(parts[1]), `"'`),
					IsBuildTime: true,
				})
			}
		}
	} else {
		toSave = req.Variables
	}

	var updatedKeys []string
	for _, item := range toSave {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}

		encryptedVal, err := crypto.Encrypt([]byte(item.Value), s.masterKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to encrypt variable")
			return
		}

		if err := s.db.SetEnvVar(projectID, key, encryptedVal, item.IsBuildTime); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to store variable")
			return
		}
		updatedKeys = append(updatedKeys, key)
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "update_env_vars", "project", project.ID,
		fmt.Sprintf(`{"keys":%q}`, updatedKeys), getClientIP(r))

	writeJSON(w, http.StatusOK, map[string]any{
		"message":      "Environment variables updated and encrypted successfully",
		"updated_keys": updatedKeys,
	})
}
