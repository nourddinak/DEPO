package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed scripts/install-agent.sh
var embeddedInstallAgentScript []byte

//go:embed scripts/install.sh
var embeddedInstallServerScript []byte

//go:embed scripts/uninstall-agent.sh
var embeddedUninstallAgentScript []byte

//go:embed scripts/uninstall.sh
var embeddedUninstallServerScript []byte

func (s *Server) handleInstallAgentScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	// Check on-disk scripts first for live development overrides
	if data, err := os.ReadFile("scripts/install-agent.sh"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embeddedInstallAgentScript)
}

func (s *Server) handleInstallServerScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	if data, err := os.ReadFile("scripts/install.sh"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embeddedInstallServerScript)
}

func (s *Server) handleUninstallAgentScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	if data, err := os.ReadFile("scripts/uninstall-agent.sh"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embeddedUninstallAgentScript)
}

func (s *Server) handleUninstallServerScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	if data, err := os.ReadFile("scripts/uninstall.sh"); err == nil && len(data) > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(embeddedUninstallServerScript)
}

func (s *Server) handleAgentDownload(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	if name == "." || name == "/" || name == "" {
		http.NotFound(w, r)
		return
	}

	// 1. Check dataDir/downloads/<name>
	localDataPath := filepath.Join(s.dataDir, "downloads", name)
	if fi, err := os.Stat(localDataPath); err == nil && !fi.IsDir() {
		http.ServeFile(w, r, localDataPath)
		return
	}

	// 2. Check bin/<name>
	localBinPath := filepath.Join("bin", name)
	if fi, err := os.Stat(localBinPath); err == nil && !fi.IsDir() {
		http.ServeFile(w, r, localBinPath)
		return
	}

	// 3. Fallback: redirect to GitHub release asset
	githubReleaseURL := fmt.Sprintf("https://github.com/depo-platform/depo/releases/latest/download/%s", name)
	http.Redirect(w, r, githubReleaseURL, http.StatusFound)
}
