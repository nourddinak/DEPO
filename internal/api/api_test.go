package api

import (
	"testing"
	"time"
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
