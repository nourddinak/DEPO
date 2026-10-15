package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL    string `json:"server_url"`
	ServerID     string `json:"server_id"`
	AuthToken    string `json:"auth_token"`
	AgentVersion string `json:"agent_version"`
	DataDir      string `json:"data_dir"`
	CaddyConfig  string `json:"caddy_config_path"`
}

func DefaultConfig() *Config {
	return &Config{
		AgentVersion: "1.0.0",
		DataDir:      "/var/lib/depo-agent",
		CaddyConfig:  "/etc/caddy/Caddyfile",
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config json: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// 0600 permissions to protect agent auth token
	return os.WriteFile(path, data, 0600)
}
