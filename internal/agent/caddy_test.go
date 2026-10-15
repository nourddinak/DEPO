package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaddyManagerIdempotentImport(t *testing.T) {
	tempDir := t.TempDir()
	mainCaddyfile := filepath.Join(tempDir, "Caddyfile")
	projectCaddyfile := filepath.Join(tempDir, "project.conf")

	cm := NewCaddyManager(mainCaddyfile)

	// Initial import
	if err := cm.EnsureMainImport(projectCaddyfile); err != nil {
		t.Fatalf("EnsureMainImport failed: %v", err)
	}

	content, err := os.ReadFile(mainCaddyfile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expectedImport := "import " + filepath.ToSlash(projectCaddyfile)
	if !strings.Contains(string(content), expectedImport) {
		t.Fatalf("Expected %s in Caddyfile, got:\n%s", expectedImport, string(content))
	}

	// Second import should be completely idempotent (count should remain 1)
	if err := cm.EnsureMainImport(projectCaddyfile); err != nil {
		t.Fatalf("Second EnsureMainImport failed: %v", err)
	}

	contentSecond, _ := os.ReadFile(mainCaddyfile)
	count := strings.Count(string(contentSecond), expectedImport)
	if count != 1 {
		t.Fatalf("Expected exactly 1 import line, found %d", count)
	}
}

func TestCaddyTemplateRendering(t *testing.T) {
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "rendered.conf")

	cm := NewCaddyManager("")
	template := `{{DOMAIN}} {
    root * {{DEPLOY_DIR}}
    file_server
}
`
	err := cm.GenerateProjectCaddyfile(template, "myapp.example.com", "/var/www/my-app/current", destPath)
	if err != nil {
		t.Fatalf("GenerateProjectCaddyfile failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	rendered := string(data)
	if !strings.Contains(rendered, "myapp.example.com {") {
		t.Fatalf("Domain substitution failed: %s", rendered)
	}
	if !strings.Contains(rendered, "root * /var/www/my-app/current") {
		t.Fatalf("Deploy dir substitution failed: %s", rendered)
	}
}
