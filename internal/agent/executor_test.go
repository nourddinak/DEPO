package agent

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDeployBaseDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty deploy dir defaults to /var/www/depo",
			input:    "",
			expected: "/var/www/depo",
		},
		{
			name:     "legacy /var/www upgraded to /var/www/depo",
			input:    "/var/www",
			expected: "/var/www/depo",
		},
		{
			name:     "legacy /var/www/ with trailing slash upgraded to /var/www/depo",
			input:    "/var/www/",
			expected: "/var/www/depo",
		},
		{
			name:     "windows backslash \\var\\www upgraded to /var/www/depo",
			input:    `\var\www`,
			expected: "/var/www/depo",
		},
		{
			name:     "windows backslash \\var\\www\\ with trailing slash upgraded to /var/www/depo",
			input:    `\var\www\`,
			expected: "/var/www/depo",
		},
		{
			name:     "already /var/www/depo remains intact",
			input:    "/var/www/depo",
			expected: "/var/www/depo",
		},
		{
			name:     "subpath under /var/www/depo preserved",
			input:    "/var/www/depo/apps",
			expected: "/var/www/depo/apps",
		},
		{
			name:     "custom absolute deploy dir preserved",
			input:    "/srv/web/my-app",
			expected: "/srv/web/my-app",
		},
		{
			name:     "whitespace deploy dir defaults to /var/www/depo",
			input:    "   \t  ",
			expected: "/var/www/depo",
		},
		{
			name:     "multiple trailing slashes /var/www/// upgraded to /var/www/depo",
			input:    "/var/www///",
			expected: "/var/www/depo",
		},
		{
			name:     "trailing slash stripped from /var/www/depo/",
			input:    "/var/www/depo/",
			expected: "/var/www/depo",
		},
		{
			name:     "trailing slash stripped from custom directory",
			input:    "/opt/apps/frontend/",
			expected: "/opt/apps/frontend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ResolveDeployBaseDir(tt.input)
			if actual != tt.expected {
				t.Errorf("ResolveDeployBaseDir(%q) = %q, expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestDeployPathCalculations(t *testing.T) {
	// Verify project paths computed using ResolveDeployBaseDir
	baseDir := ResolveDeployBaseDir("")
	projectID := "proj-frontend-xyz"
	releaseID := "rel-v123"
	previewID := "prev-pr-45"

	projectRoot := filepath.Join(baseDir, projectID)
	releasesDir := filepath.Join(projectRoot, "releases")
	targetReleaseDir := filepath.Join(releasesDir, releaseID)
	currentSymlink := filepath.Join(projectRoot, "current")
	previewRoot := filepath.Join(baseDir, projectID, "previews", previewID)

	expectedRootPrefix := filepath.Join("/var/www/depo", projectID)
	if !strings.HasPrefix(projectRoot, expectedRootPrefix) {
		t.Errorf("expected projectRoot %q to start with %q", projectRoot, expectedRootPrefix)
	}

	expectedReleaseDir := filepath.Join("/var/www/depo", projectID, "releases", releaseID)
	if targetReleaseDir != expectedReleaseDir {
		t.Errorf("targetReleaseDir = %q, expected %q", targetReleaseDir, expectedReleaseDir)
	}

	expectedSymlink := filepath.Join("/var/www/depo", projectID, "current")
	if currentSymlink != expectedSymlink {
		t.Errorf("currentSymlink = %q, expected %q", currentSymlink, expectedSymlink)
	}

	expectedPreviewRoot := filepath.Join("/var/www/depo", projectID, "previews", previewID)
	if previewRoot != expectedPreviewRoot {
		t.Errorf("previewRoot = %q, expected %q", previewRoot, expectedPreviewRoot)
	}
}
