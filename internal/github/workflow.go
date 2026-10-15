package github

import (
	"fmt"
	"strings"
)

type WorkflowConfig struct {
	ProjectName string
	ServerURL   string
	Branch      string
	FrontendDir string
	BuildCmd    string
	OutputDir   string
	NodeVersion string
	UseOIDC     bool
}

// GenerateWorkflowYAML generates a clean, secure, least-privilege GitHub Actions workflow.
func GenerateWorkflowYAML(cfg WorkflowConfig) string {
	if cfg.NodeVersion == "" {
		cfg.NodeVersion = "20"
	}
	if cfg.FrontendDir == "" {
		cfg.FrontendDir = "."
	}
	if cfg.BuildCmd == "" {
		cfg.BuildCmd = "npm run build"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "dist"
	}

	cdStep := ""
	cleanFrontendDir := strings.Trim(cfg.FrontendDir, "./")
	if cleanFrontendDir != "" && cleanFrontendDir != "." {
		cdStep = fmt.Sprintf("working-directory: %s", cleanFrontendDir)
	}

	// Least privilege permissions: id-token write only if using OIDC, contents read
	permissions := `permissions:
  contents: read
  id-token: write`
	if !cfg.UseOIDC {
		permissions = `permissions:
  contents: read`
	}

	authHeader := ""
	if cfg.UseOIDC {
		authHeader = `-H "Authorization: Bearer $ID_TOKEN"`
	} else {
		authHeader = `-H "X-Deploy-Token: ${{ secrets.DEPO_DEPLOY_TOKEN }}"`
	}

	return fmt.Sprintf(`name: Deploy to DEPO (%s)

on:
  push:
    branches: [ "%s" ]
  pull_request:
    branches: [ "%s" ]

%s

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: %s
          cache: 'npm'
          %s

      - name: Install dependencies
        %s
        run: npm ci || npm install

      - name: Build static frontend
        %s
        run: %s

      - name: Package build artifact & manifest
        run: |
          mkdir -p .depo-package
          cp -r %s/* .depo-package/
          
          # Generate artifact manifest
          cat <<EOF > .depo-package/manifest.json
          {
            "project": "%s",
            "commit": "${{ github.sha }}",
            "branch": "${{ github.ref_name }}",
            "actor": "${{ github.actor }}",
            "built_at": "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)"
          }
          EOF
          
          tar -czf artifact.tar.gz -C .depo-package .
          SHA256=$(sha256sum artifact.tar.gz | awk '{print $1}')
          echo "ARTIFACT_SHA256=$SHA256" >> $GITHUB_ENV

      - name: Upload Artifact to DEPO Central Server
        run: |
          %s
          curl -f -sSL -X POST "%s/api/v1/artifacts/upload" \
            %s \
            -H "X-Project-Name: %s" \
            -H "X-Commit-Hash: ${{ github.sha }}" \
            -H "X-Commit-Msg: ${{ github.event.head_commit.message }}" \
            -H "X-Branch: ${{ github.ref_name }}" \
            -H "X-Artifact-Checksum: $ARTIFACT_SHA256" \
            -H "X-Is-PR: ${{ github.event_name == 'pull_request' }}" \
            -H "X-PR-Number: ${{ github.event.pull_request.number }}" \
            -F "file=@artifact.tar.gz"
`,
		cfg.ProjectName,
		cfg.Branch,
		cfg.Branch,
		permissions,
		cfg.NodeVersion,
		cdStep,
		cdStep,
		cdStep,
		cfg.BuildCmd,
		cfg.OutputDir,
		cfg.ProjectName,
		getOIDCTokenFetchScript(cfg.UseOIDC),
		strings.TrimRight(cfg.ServerURL, "/"),
		authHeader,
		cfg.ProjectName,
	)
}

func getOIDCTokenFetchScript(useOIDC bool) string {
	if !useOIDC {
		return "# Authenticating using project deploy token"
	}
	return `# Request OIDC token from GitHub Actions runtime
          ID_TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=depo-deploy" | jq -r '.value')`
}
