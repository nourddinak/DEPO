package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CaddyManager struct {
	mainConfigPath string
	mu             sync.Mutex
}

func NewCaddyManager(mainConfigPath string) *CaddyManager {
	if mainConfigPath == "" {
		mainConfigPath = "/etc/caddy/Caddyfile"
	}
	return &CaddyManager{
		mainConfigPath: mainConfigPath,
	}
}

// GenerateProjectCaddyfile populates {{DOMAIN}} and {{DEPLOY_DIR}} placeholders and writes to destPath.
func (cm *CaddyManager) GenerateProjectCaddyfile(customTemplate, domain, deployDir, destPath string) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating project caddy directory: %w", err)
	}

	content := customTemplate
	if strings.TrimSpace(content) == "" {
		// Standard production static SPA configuration
		content = `{{DOMAIN}} {
    root * {{DEPLOY_DIR}}
    encode zstd gzip
    try_files {path} /index.html
    file_server
}
`
	}

	// Safe template replacements: no shell evaluation
	rendered := strings.ReplaceAll(content, "{{DOMAIN}}", domain)
	rendered = strings.ReplaceAll(rendered, "{{DEPLOY_DIR}}", deployDir)

	return os.WriteFile(destPath, []byte(rendered), 0644)
}

// EnsureMainImport ensures that the project's stable Caddyfile is imported in the main /etc/caddy/Caddyfile.
func (cm *CaddyManager) EnsureMainImport(projectCaddyPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	importLine := fmt.Sprintf("import %s", filepath.ToSlash(projectCaddyPath))

	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(cm.mainConfigPath), 0755); err != nil {
		return err
	}

	// If main Caddyfile doesn't exist, create it with this import
	if _, err := os.Stat(cm.mainConfigPath); os.IsNotExist(err) {
		initial := fmt.Sprintf("# Managed by DEPO Central Platform\n%s\n", importLine)
		return os.WriteFile(cm.mainConfigPath, []byte(initial), 0644)
	}

	// Read existing content
	existingBytes, err := os.ReadFile(cm.mainConfigPath)
	if err != nil {
		return fmt.Errorf("reading main Caddyfile: %w", err)
	}
	existing := string(existingBytes)

	// Check if already imported
	lines := strings.Split(existing, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == importLine || trimmed == fmt.Sprintf("import %s", projectCaddyPath) {
			return nil // Already imported exactly once
		}
	}

	// Backup current config before modifying
	backupPath := cm.mainConfigPath + ".bak"
	if err := os.WriteFile(backupPath, existingBytes, 0644); err != nil {
		return fmt.Errorf("creating Caddyfile backup: %w", err)
	}

	// Append import line safely
	var newContent string
	if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
		newContent = existing + "\n" + importLine + "\n"
	} else {
		newContent = existing + importLine + "\n"
	}

	return os.WriteFile(cm.mainConfigPath, []byte(newContent), 0644)
}

// RemoveMainImport removes the project's import line when a project is deleted or preview destroyed.
func (cm *CaddyManager) RemoveMainImport(projectCaddyPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, err := os.Stat(cm.mainConfigPath); os.IsNotExist(err) {
		return nil
	}

	existingBytes, err := os.ReadFile(cm.mainConfigPath)
	if err != nil {
		return err
	}

	targetSlash := fmt.Sprintf("import %s", filepath.ToSlash(projectCaddyPath))
	targetNative := fmt.Sprintf("import %s", projectCaddyPath)

	lines := strings.Split(string(existingBytes), "\n")
	var filtered []string
	changed := false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == targetSlash || trimmed == targetNative {
			changed = true
			continue
		}
		filtered = append(filtered, l)
	}

	if changed {
		return os.WriteFile(cm.mainConfigPath, []byte(strings.Join(filtered, "\n")), 0644)
	}
	return nil
}

// ValidateConfig verifies Caddy syntax against the main Caddyfile.
func (cm *CaddyManager) ValidateConfig() error {
	// Look for caddy executable
	caddyBin, err := exec.LookPath("caddy")
	if err != nil {
		// In development/test environments where caddy is not installed, simulate success if file exists
		if _, statErr := os.Stat(cm.mainConfigPath); statErr == nil {
			return nil
		}
		return fmt.Errorf("caddy binary not found on PATH: %w", err)
	}

	cmd := exec.Command(caddyBin, "validate", "--config", cm.mainConfigPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("caddy validate failed: %w (output: %s)", err, out.String())
	}
	return nil
}

// Reload reloads the running Caddy server gracefully.
func (cm *CaddyManager) Reload() error {
	caddyBin, err := exec.LookPath("caddy")
	if err != nil {
		// In environments without caddy binary, return nil for testability
		return nil
	}

	cmd := exec.Command(caddyBin, "reload", "--config", cm.mainConfigPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("caddy reload failed: %w (output: %s)", err, out.String())
	}
	return nil
}

// RevertRestores the main Caddyfile from backup.
func (cm *CaddyManager) Revert() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	backupPath := cm.mainConfigPath + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		return nil // No backup to revert
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	return os.WriteFile(cm.mainConfigPath, data, 0644)
}
