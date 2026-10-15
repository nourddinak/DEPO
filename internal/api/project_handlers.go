package api

import (
	"fmt"
	"net/http"
	"strings"

	"depo/internal/crypto"
	"depo/internal/db"
	"depo/internal/github"
)

type DetectFrameworkRequest struct {
	PackageJSON string `json:"package_json"`
}

type FrameworkDetectionResult struct {
	Framework   string `json:"framework"`
	FrontendDir string `json:"frontend_dir"`
	BuildCmd    string `json:"build_cmd"`
	OutputDir   string `json:"output_dir"`
}

type CreateProjectRequest struct {
	Name              string `json:"name"`
	RepoURL           string `json:"repo_url"`
	Branch            string `json:"branch"`
	FrontendDir       string `json:"frontend_dir"`
	BuildCmd          string `json:"build_cmd"`
	OutputDir         string `json:"output_dir"`
	ServerID          string `json:"server_id"`
	DeployDir         string `json:"deploy_dir"`
	RunUser           string `json:"run_user"`
	CaddyfilePath     string `json:"caddyfile_path"`
	Domain            string `json:"domain"`
	AutoDeploy        bool   `json:"auto_deploy"`
	PreviewEnabled    bool   `json:"preview_enabled"`
	PreviewBaseDomain string `json:"preview_base_domain"`
	LogRetentionDays  int    `json:"log_retention_days"`
}

// Auto-detect framework from package.json content
func detectFramework(pkgJSON string) FrameworkDetectionResult {
	lower := strings.ToLower(pkgJSON)

	res := FrameworkDetectionResult{
		Framework:   "Static HTML/JS",
		FrontendDir: "frontend/",
		BuildCmd:    "echo 'No build required'",
		OutputDir:   "dist/",
	}

	if strings.Contains(lower, `"astro"`) {
		res.Framework = "Astro"
		res.BuildCmd = "npm run build"
		res.OutputDir = "dist/"
	} else if strings.Contains(lower, `"vite"`) {
		res.Framework = "Vite (React/Vue/Svelte)"
		res.BuildCmd = "npm run build"
		res.OutputDir = "dist/"
	} else if strings.Contains(lower, `"next"`) {
		res.Framework = "Next.js (Static Export)"
		res.BuildCmd = "npx next build"
		res.OutputDir = "out/"
	} else if strings.Contains(lower, `"nuxt"`) {
		res.Framework = "Nuxt (Static Generate)"
		res.BuildCmd = "npx nuxt generate"
		res.OutputDir = ".output/public/"
	} else if strings.Contains(lower, `"svelte"`) {
		res.Framework = "Svelte"
		res.BuildCmd = "npm run build"
		res.OutputDir = "build/"
	} else if strings.Contains(lower, `"react-scripts"`) {
		res.Framework = "Create React App"
		res.BuildCmd = "npm run build"
		res.OutputDir = "build/"
	} else if strings.Contains(lower, `"@angular/core"`) {
		res.Framework = "Angular"
		res.BuildCmd = "npx ng build --configuration production"
		res.OutputDir = "dist/"
	}

	return res
}

func (s *Server) handleDetectFramework(w http.ResponseWriter, r *http.Request) {
	var req DetectFrameworkRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	result := detectFramework(req.PackageJSON)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list projects")
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	var req CreateProjectRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "Target server is required")
		return
	}

	// Verify server exists
	srv, err := s.db.GetServerByID(req.ServerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Specified target server does not exist")
		return
	}

	// Generate project deploy token
	rawDeployToken, tokenHash, err := crypto.GenerateDeployToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate deploy token")
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.FrontendDir == "" {
		req.FrontendDir = "frontend/"
	}
	if req.BuildCmd == "" {
		req.BuildCmd = "npm run build"
	}
	if req.OutputDir == "" {
		req.OutputDir = "dist/"
	}
	req.DeployDir = db.NormalizeDeployDir(req.DeployDir)
	if req.RunUser == "" {
		req.RunUser = "www-data"
	}
	if req.CaddyfilePath == "" {
		req.CaddyfilePath = "frontend/Caddyfile"
	}
	if req.LogRetentionDays <= 0 {
		req.LogRetentionDays = 7
	}

	p := &db.Project{
		Name:              name,
		RepoURL:           req.RepoURL,
		Branch:            req.Branch,
		FrontendDir:       req.FrontendDir,
		BuildCmd:          req.BuildCmd,
		OutputDir:         req.OutputDir,
		ServerID:          srv.ID,
		DeployDir:         req.DeployDir,
		RunUser:           req.RunUser,
		CaddyfilePath:     req.CaddyfilePath,
		Domain:            req.Domain,
		DeployTokenHash:   tokenHash,
		AutoDeploy:        req.AutoDeploy,
		PreviewEnabled:    req.PreviewEnabled,
		PreviewBaseDomain: req.PreviewBaseDomain,
		LogRetentionDays:  req.LogRetentionDays,
	}

	if err := s.db.CreateProject(p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to create project: %v", err))
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "create_project", "project", p.ID, fmt.Sprintf(`{"name":"%s"}`, p.Name), getClientIP(r))

	workflowYAML := github.GenerateWorkflowYAML(github.WorkflowConfig{
		ProjectName: p.Name,
		ServerURL:   s.externalURL,
		Branch:      p.Branch,
		FrontendDir: p.FrontendDir,
		BuildCmd:    p.BuildCmd,
		OutputDir:   p.OutputDir,
		UseOIDC:     true,
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"project":       p,
		"deploy_token":  rawDeployToken,
		"workflow_yaml": workflowYAML,
	})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.db.GetProjectByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	id := r.PathValue("id")

	p, err := s.db.GetProjectByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	var req CreateProjectRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Name != "" {
		p.Name = req.Name
	}
	if req.RepoURL != "" {
		p.RepoURL = req.RepoURL
	}
	if req.Branch != "" {
		p.Branch = req.Branch
	}
	if req.FrontendDir != "" {
		p.FrontendDir = req.FrontendDir
	}
	if req.BuildCmd != "" {
		p.BuildCmd = req.BuildCmd
	}
	if req.OutputDir != "" {
		p.OutputDir = req.OutputDir
	}
	if req.Domain != "" {
		p.Domain = req.Domain
	}
	if req.ServerID != "" {
		p.ServerID = req.ServerID
	}
	if req.DeployDir != "" {
		p.DeployDir = db.NormalizeDeployDir(req.DeployDir)
	}
	p.AutoDeploy = req.AutoDeploy
	p.PreviewEnabled = req.PreviewEnabled
	p.PreviewBaseDomain = req.PreviewBaseDomain

	if err := s.db.UpdateProject(p); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update project")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "update_project", "project", p.ID, fmt.Sprintf(`{"name":"%s"}`, p.Name), getClientIP(r))
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r.Context())
	id := r.PathValue("id")

	p, err := s.db.GetProjectByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	if err := s.db.DeleteProject(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete project")
		return
	}

	_ = s.db.RecordAuditLog(user.ID, user.Email, "delete_project", "project", id, fmt.Sprintf(`{"name":"%s"}`, p.Name), getClientIP(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Project deleted successfully"})
}

func (s *Server) handleGetProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.db.GetProjectByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}

	workflowYAML := github.GenerateWorkflowYAML(github.WorkflowConfig{
		ProjectName: p.Name,
		ServerURL:   s.externalURL,
		Branch:      p.Branch,
		FrontendDir: p.FrontendDir,
		BuildCmd:    p.BuildCmd,
		OutputDir:   p.OutputDir,
		UseOIDC:     true,
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"workflow_yaml": workflowYAML,
		"target_path":   ".github/workflows/depo-deploy.yml",
	})
}
