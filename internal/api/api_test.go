package api

import (
	"testing"
	"time"

	"depo/internal/db"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, 100*time.Millisecond)

	ip := "192.168.1.1"
	if !rl.Allow(ip) {
		t.Fatal("expected attempt 1 to be allowed")
	}
	if !rl.Allow(ip) {
		t.Fatal("expected attempt 2 to be allowed")
	}
	if !rl.Allow(ip) {
		t.Fatal("expected attempt 3 to be allowed")
	}
	// 4th should be blocked
	if rl.Allow(ip) {
		t.Fatal("expected attempt 4 to be blocked by rate limiter")
	}

	// After window expires
	time.Sleep(120 * time.Millisecond)
	if !rl.Allow(ip) {
		t.Fatal("expected attempt after window expiry to be allowed")
	}
}

func TestFrameworkAutoDetection(t *testing.T) {
	vitePkg := `{
		"dependencies": {
			"react": "^18.2.0",
			"react-dom": "^18.2.0"
		},
		"devDependencies": {
			"vite": "^5.0.0"
		}
	}`
	res := detectFramework(vitePkg)
	if res.Framework != "Vite (React/Vue/Svelte)" || res.BuildCmd != "npm run build" || res.OutputDir != "dist/" {
		t.Fatalf("Unexpected framework detection for Vite: %+v", res)
	}

	astroPkg := `{"dependencies": {"astro": "^4.0.0"}}`
	astroRes := detectFramework(astroPkg)
	if astroRes.Framework != "Astro" {
		t.Fatalf("Unexpected framework detection for Astro: %+v", astroRes)
	}
}

func TestParseEnvExample(t *testing.T) {
	sample := `
# Base API URL
VITE_API_URL=https://api.example.com

# Analytics tracking token
VITE_ANALYTICS_ID="UA-123456"

# Port
PORT=3000
`
	items := parseEnvExample(sample)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Key != "VITE_API_URL" || items[0].Value != "https://api.example.com" {
		t.Fatalf("Unexpected item 0: %+v", items[0])
	}
	if items[1].Key != "VITE_ANALYTICS_ID" || items[1].Value != "UA-123456" {
		t.Fatalf("Unexpected item 1: %+v", items[1])
	}
}

func TestProjectDeployDirResolution(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string defaults to /var/www/depo", "", "/var/www/depo"},
		{"whitespace defaults to /var/www/depo", "   ", "/var/www/depo"},
		{"legacy /var/www upgraded to /var/www/depo", "/var/www", "/var/www/depo"},
		{"legacy /var/www/ trailing slash upgraded to /var/www/depo", "/var/www/", "/var/www/depo"},
		{"multiple trailing slashes /var/www/// upgraded", "/var/www///", "/var/www/depo"},
		{"windows backslash \\var\\www upgraded to /var/www/depo", `\var\www`, "/var/www/depo"},
		{"windows backslash with trailing slash upgraded", `\var\www\`, "/var/www/depo"},
		{"standard /var/www/depo preserved", "/var/www/depo", "/var/www/depo"},
		{"trailing slash on /var/www/depo/ cleaned", "/var/www/depo/", "/var/www/depo"},
		{"subpath under /var/www/depo preserved", "/var/www/depo/apps", "/var/www/depo/apps"},
		{"custom path preserved and cleaned", "/opt/custom/path/", "/opt/custom/path"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := db.NormalizeDeployDir(c.input)
			if actual != c.expected {
				t.Errorf("NormalizeDeployDir(%q) = %q, expected %q", c.input, actual, c.expected)
			}

			// Verify CreateProjectRequest handling matches
			req := CreateProjectRequest{DeployDir: c.input}
			normalized := db.NormalizeDeployDir(req.DeployDir)
			if normalized != c.expected {
				t.Errorf("CreateProjectRequest normalized = %q, expected %q", normalized, c.expected)
			}
		})
	}
}

