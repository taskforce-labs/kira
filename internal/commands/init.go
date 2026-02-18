package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kira/internal/config"
	"kira/internal/templates"

	"github.com/spf13/cobra"
)

// docsSubdirs are the standard subdirectories under the docs folder (relative paths).
var docsSubdirs = []string{
	"agents", "architecture", "product", "reports", "guides", "api",
	"guides/security",
}

var initCmd = &cobra.Command{
	Use:   "init [folder]",
	Short: "Initialize a kira workspace",
	Long:  `Creates the files and folders used by kira in the specified directory.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		cfg, err := config.LoadConfigFromDir(targetDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		fillMissing, _ := cmd.Flags().GetBool("fill-missing")
		workPath := filepath.Join(targetDir, config.GetWorkFolderPath(cfg))
		docsPath := filepath.Join(targetDir, config.GetDocsFolderPath(cfg))
		if err := ensureWorkspaceDecision(workPath, docsPath, force, fillMissing); err != nil {
			return err
		}

		return initializeWorkspace(targetDir, cfg)
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "Overwrite existing work folder if present")
	initCmd.Flags().Bool("fill-missing", false, "Create any missing files/folders without overwriting existing ones")
}

func initializeWorkspace(targetDir string, cfg *config.Config) error {
	// Create work directory (configured work folder)
	workDir := filepath.Join(targetDir, config.GetWorkFolderPath(cfg))
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create status folders and .gitkeep files
	for _, folder := range cfg.StatusFolders {
		folderPath := filepath.Join(workDir, folder)
		if err := os.MkdirAll(folderPath, 0o700); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
		if err := os.WriteFile(filepath.Join(folderPath, ".gitkeep"), []byte(""), 0o600); err != nil {
			return fmt.Errorf("failed to create .gitkeep in %s: %w", folder, err)
		}
	}

	// Create templates directory and default templates and .gitkeep
	if err := templates.CreateDefaultTemplates(workDir); err != nil {
		return fmt.Errorf("failed to create default templates: %w", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "templates", ".gitkeep"), []byte(""), 0o600); err != nil {
		return fmt.Errorf("failed to create .gitkeep in templates: %w", err)
	}

	// Create or preserve IDEAS.md file (prepend header if missing)
	ideasPath := filepath.Join(workDir, "IDEAS.md")
	header := `# Ideas

This file is for capturing quick ideas and thoughts that don't fit into formal work items yet.

## How to use
- Add ideas with timestamps using ` + "`kira idea add \"your idea here\"`" + `
- Or manually add entries below

## List

`
	if _, err := os.Stat(ideasPath); os.IsNotExist(err) {
		if err := os.WriteFile(ideasPath, []byte(header), 0o600); err != nil {
			return fmt.Errorf("failed to create IDEAS.md: %w", err)
		}
	} else {
		content, readErr := safeReadFile(ideasPath, cfg)
		if readErr != nil {
			return fmt.Errorf("failed to read IDEAS.md: %w", readErr)
		}
		if !strings.HasPrefix(string(content), "# Ideas") {
			newContent := header + string(content)
			if err := os.WriteFile(ideasPath, []byte(newContent), 0o600); err != nil {
				return fmt.Errorf("failed to update IDEAS.md: %w", err)
			}
		}
	}

	// Create docs folder and standard subdirs
	if err := initializeDocsFolder(targetDir, cfg); err != nil {
		return err
	}

	// Create GitHub workflow file if git_platform is github
	if err := initializeGitHubWorkflow(targetDir, cfg); err != nil {
		return err
	}

	// Create kira.yml config file under the target directory
	if err := config.SaveConfigToDir(cfg, targetDir); err != nil {
		return fmt.Errorf("failed to create kira.yml: %w", err)
	}

	fmt.Printf("Initialized kira workspace in %s\n", targetDir)
	return nil
}

func initializeDocsFolder(targetDir string, cfg *config.Config) error {
	docsRoot := filepath.Join(targetDir, config.GetDocsFolderPath(cfg))
	if err := os.MkdirAll(docsRoot, 0o700); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}
	for _, sub := range docsSubdirs {
		subPath := filepath.Join(docsRoot, sub)
		if err := os.MkdirAll(subPath, 0o700); err != nil {
			return fmt.Errorf("failed to create docs subfolder %s: %w", sub, err)
		}
	}
	return writeDocsIndexFiles(docsRoot)
}

// docsIndexEntries defines relative path (under docs root) and README content. Empty path = docs root.
var docsIndexEntries = []struct {
	path    string
	content string
}{
	{"", `# Documentation

Overview of project documentation. Use this folder for long-lived reference material (ADRs, guides, product docs, reports). Work items and specs live in .work instead.

## Sections

- [Agents](agents/) – Agent-specific documentation (e.g. using kira)
- [Architecture](architecture/) – Architecture Decision Records and diagrams
- [Product](product/) – Product vision, roadmap, personas, glossary, feature briefs
- [Reports](reports/) – Release reports, metrics, audits, retrospectives
- [API](api/) – API reference
- [Guides](guides/) – Development and usage guides (including security)
`},
	{"agents", `# Agent documentation

Documentation for agents and tooling (e.g. [using-kira](using-kira.md)).
`},
	{"architecture", `# Architecture

Architecture Decision Records (ADRs) and system design documents.
`},
	{"product", `# Product

Product vision, roadmap, personas, glossary, feature briefs, and commercials.
`},
	{"reports", `# Reports

Release reports, metrics summaries, audits, and retrospectives.
`},
	{"api", `# API

API reference documentation.
`},
	{"guides", `# Guides

Development and usage guides. See [security/](security/) for security guidelines.
`},
	{"guides/security", `# Security

Security guidelines (e.g. [golang-secure-coding](golang-secure-coding.md)).
`},
}

func writeDocsIndexFiles(docsRoot string) error {
	for _, e := range docsIndexEntries {
		dir := docsRoot
		if e.path != "" {
			dir = filepath.Join(docsRoot, e.path)
		}
		readmePath := filepath.Join(dir, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			continue
		}
		if err := os.WriteFile(readmePath, []byte(e.content), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", readmePath, err)
		}
	}
	return nil
}

func removePathIfExists(path, kind string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove existing %s: %w", kind, err)
	}
	return nil
}

func ensureWorkspaceDecision(workPath, docsPath string, force, fillMissing bool) error {
	workExists := pathExists(workPath)
	docsExists := pathExists(docsPath)
	if !workExists && !docsExists {
		return nil
	}
	if force {
		_ = removePathIfExists(workPath, "work folder")
		_ = removePathIfExists(docsPath, "docs folder")
		return nil
	}
	if fillMissing {
		return nil
	}

	fmt.Printf("Workspace (.work and docs) already exists. Choose an option: [c]ancel, [o]verwrite, [f]ill-missing\n")
	fmt.Print("Enter choice (c/o/f): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	choice := strings.ToLower(strings.TrimSpace(input))
	if choice == "f" || choice == "fill-missing" {
		return nil
	}
	if choice == "o" || choice == "overwrite" {
		_ = removePathIfExists(workPath, "work folder")
		_ = removePathIfExists(docsPath, "docs folder")
		return nil
	}
	return fmt.Errorf("init cancelled")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// initializeGitHubWorkflow creates the GitHub Actions workflow file if git_platform is github.
func initializeGitHubWorkflow(targetDir string, cfg *config.Config) error {
	// Check if git_platform is github
	gitPlatform := ""
	if cfg.Workspace != nil {
		gitPlatform = cfg.Workspace.GitPlatform
	}
	// Default to "auto" if not set, but we only create workflow for explicit "github"
	if gitPlatform != "github" {
		return nil
	}

	// Create .github/workflows directory if it doesn't exist
	workflowsDir := filepath.Join(targetDir, ".github", "workflows")
	// #nosec G301 - directory permissions 0o755 are required for GitHub Actions to read workflow files
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Check if workflow file already exists
	workflowPath := filepath.Join(workflowsDir, "update-pr-details.yml")
	if _, err := os.Stat(workflowPath); err == nil {
		// File already exists, skip creation
		return nil
	}

	// Create workflow file from embedded template
	workflowContent := getUpdatePRDetailsWorkflowTemplate()
	// #nosec G306 - file permissions 0o644 are required for GitHub Actions to read workflow files
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
		return fmt.Errorf("failed to create update-pr-details.yml: %w", err)
	}

	return nil
}

// getUpdatePRDetailsWorkflowTemplate returns the GitHub Actions workflow template for updating PR details.
func getUpdatePRDetailsWorkflowTemplate() string {
	return `name: Update PR Details

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  update-pr-details:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - name: Checkout PR head ref
        uses: actions/checkout@v4
        with:
          ref: ${{ github.head_ref }}

      - name: Read trunk branch from kira.yml
        id: trunk-branch
        run: |
          if [ ! -f kira.yml ]; then
            echo "Error: kira.yml not found"
            exit 1
          fi
          # Try to read trunk_branch using yq if available, otherwise use grep/sed
          if command -v yq &> /dev/null; then
            TRUNK=$(yq eval '.git.trunk_branch' kira.yml 2>/dev/null || echo "")
          else
            # Fallback: use grep and sed to extract trunk_branch value
            TRUNK=$(grep -E "^\s*trunk_branch\s*:" kira.yml | sed 's/.*trunk_branch\s*:\s*["'\'']*\([^"'\'']*\)["'\'']*/\1/' | tr -d ' ' || echo "")
          fi
          # If trunk_branch is empty or not set, default to master/main
          if [ -z "$TRUNK" ] || [ "$TRUNK" = "null" ]; then
            # Try to detect default branch
            if git show-ref --verify --quiet refs/heads/master; then
              TRUNK="master"
            else
              TRUNK="main"
            fi
          fi
          echo "trunk=$TRUNK" >> $GITHUB_OUTPUT
          echo "Trunk branch: $TRUNK"

      - name: Skip if head branch is trunk
        if: github.head_ref == steps.trunk-branch.outputs.trunk
        run: |
          echo "Skipping: PR head branch (${{ github.head_ref }}) is the trunk branch (${{ steps.trunk-branch.outputs.trunk }})"
          exit 0

      - name: Verify base branch matches trunk
        if: github.base_ref != steps.trunk-branch.outputs.trunk
        run: |
          echo "Error: PR base branch (${{ github.base_ref }}) does not match trunk branch (${{ steps.trunk-branch.outputs.trunk }})"
          exit 1

      - name: Set up Go
        if: github.head_ref != steps.trunk-branch.outputs.trunk
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Cache Go modules
        if: github.head_ref != steps.trunk-branch.outputs.trunk
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Build kira
        if: github.head_ref != steps.trunk-branch.outputs.trunk
        run: make build

      - name: Install jq
        if: github.head_ref != steps.trunk-branch.outputs.trunk
        run: sudo apt-get update && sudo apt-get install -y jq

      - name: Validate branch name format
        if: github.head_ref != steps.trunk-branch.outputs.trunk
        id: validate-branch
        continue-on-error: true
        run: |
          BRANCH=$(./kira current --slug 2>&1)
          EXIT_CODE=$?
          if [ $EXIT_CODE -ne 0 ]; then
            echo "Branch name does not match kira format, skipping PR update"
            echo "valid=false" >> $GITHUB_OUTPUT
            exit 0
          fi
          echo "Branch: $BRANCH"
          echo "branch=$BRANCH" >> $GITHUB_OUTPUT
          echo "valid=true" >> $GITHUB_OUTPUT

      - name: Get PR title
        if: steps.validate-branch.outputs.valid == 'true'
        id: pr-title
        continue-on-error: true
        run: |
          TITLE=$(./kira current --title 2>&1)
          EXIT_CODE=$?
          if [ $EXIT_CODE -ne 0 ]; then
            echo "Work item not found or invalid branch name, skipping PR update"
            echo "title=" >> $GITHUB_OUTPUT
            echo "skip=true" >> $GITHUB_OUTPUT
            exit 0
          fi
          DELIMITER=$(openssl rand -hex 16)
          echo "title<<$DELIMITER" >> $GITHUB_OUTPUT
          echo "$TITLE" >> $GITHUB_OUTPUT
          echo "$DELIMITER" >> $GITHUB_OUTPUT
          echo "skip=false" >> $GITHUB_OUTPUT

      - name: Get PR body
        id: pr-body
        if: steps.validate-branch.outputs.valid == 'true' && steps.pr-title.outputs.skip != 'true'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        continue-on-error: true
        run: |
          BODY=$(./kira current --body 2>&1)
          EXIT_CODE=$?
          if [ $EXIT_CODE -ne 0 ]; then
            echo "Work item not found or invalid branch name, skipping PR update"
            echo "body=" >> $GITHUB_OUTPUT
            echo "skip=true" >> $GITHUB_OUTPUT
            exit 0
          fi
          # Check PR body size limit (~65KB)
          BODY_SIZE=$(echo "$BODY" | wc -c)
          MAX_SIZE=66560  # 65KB in bytes
          if [ $BODY_SIZE -gt $MAX_SIZE ]; then
            echo "Error: Work item file exceeds GitHub PR body size limit (~65KB). Please reduce the work item size or split it into multiple work items."
            exit 1
          fi
          DELIMITER=$(openssl rand -hex 16)
          echo "body<<$DELIMITER" >> $GITHUB_OUTPUT
          echo "$BODY" >> $GITHUB_OUTPUT
          echo "$DELIMITER" >> $GITHUB_OUTPUT
          echo "skip=false" >> $GITHUB_OUTPUT

      - name: Get related PRs
        id: prs
        if: steps.validate-branch.outputs.valid == 'true' && steps.pr-title.outputs.skip != 'true' && steps.pr-body.outputs.skip != 'true'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        continue-on-error: true
        run: |
          PRS=$(./kira current prs 2>&1)
          EXIT_CODE=$?
          if [ $EXIT_CODE -ne 0 ] || [ -z "$PRS" ] || [ "$PRS" = "[]" ]; then
            echo "No PRs found or error getting PR list, skipping PR update"
            echo "prs=[]" >> $GITHUB_OUTPUT
            echo "skip=true" >> $GITHUB_OUTPUT
            exit 0
          fi
          # Validate JSON structure
          if ! echo "$PRS" | jq empty 2>/dev/null; then
            echo "Error: Invalid JSON from kira current prs: $PRS"
            exit 1
          fi
          DELIMITER=$(openssl rand -hex 16)
          echo "prs<<$DELIMITER" >> $GITHUB_OUTPUT
          echo "$PRS" >> $GITHUB_OUTPUT
          echo "$DELIMITER" >> $GITHUB_OUTPUT
          echo "skip=false" >> $GITHUB_OUTPUT

      - name: Update PRs
        if: steps.validate-branch.outputs.valid == 'true' && steps.pr-title.outputs.skip != 'true' && steps.pr-body.outputs.skip != 'true' && steps.prs.outputs.skip != 'true'
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          PRS='${{ steps.prs.outputs.prs }}'
          TITLE='${{ steps.pr-title.outputs.title }}'
          BODY='${{ steps.pr-body.outputs.body }}'
          
          # Parse PRs JSON array
          PR_COUNT=$(echo "$PRS" | jq '. | length')
          
          MAIN_REPO_UPDATED=false
          
          for i in $(seq 0 $((PR_COUNT - 1))); do
            OWNER=$(echo "$PRS" | jq -r ".[$i].owner")
            REPO=$(echo "$PRS" | jq -r ".[$i].repo")
            PR_NUMBER=$(echo "$PRS" | jq -r ".[$i].pr_number")
            BRANCH=$(echo "$PRS" | jq -r ".[$i].branch")
            
            # Check if this is the main repo (where workflow runs)
            IS_MAIN_REPO=false
            if [ "$OWNER/$REPO" = "${{ github.repository }}" ]; then
              IS_MAIN_REPO=true
            fi
            
            echo "Updating PR #$PR_NUMBER in $OWNER/$REPO..."
            
            # Update PR with retry logic for rate limiting
            MAX_RETRIES=3
            RETRY_DELAY=2
            RETRY_COUNT=0
            SUCCESS=false
            
            while [ $RETRY_COUNT -lt $MAX_RETRIES ] && [ "$SUCCESS" = "false" ]; do
              RESPONSE=$(curl -s -w "\n%{http_code}" \
                -X PATCH \
                -H "Authorization: token $GITHUB_TOKEN" \
                -H "Accept: application/vnd.github.v3+json" \
                -H "Content-Type: application/json" \
                -d "{\"title\":$(echo "$TITLE" | jq -Rs .),\"body\":$(echo "$BODY" | jq -Rs .)}" \
                "https://api.github.com/repos/$OWNER/$REPO/pulls/$PR_NUMBER")
              
              HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
              RESPONSE_BODY=$(echo "$RESPONSE" | sed '$d')
              
              if [ "$HTTP_CODE" = "200" ]; then
                echo "Successfully updated PR #$PR_NUMBER in $OWNER/$REPO"
                SUCCESS=true
                if [ "$IS_MAIN_REPO" = "true" ]; then
                  MAIN_REPO_UPDATED=true
                fi
              elif [ "$HTTP_CODE" = "403" ]; then
                if [ "$IS_MAIN_REPO" = "true" ]; then
                  echo "Error: Failed to update PR in main repo ($OWNER/$REPO). Ensure the workflow has pull-requests: write permission. See https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions for details."
                  exit 1
                else
                  echo "Warning: Could not update PR in $OWNER/$REPO: GITHUB_TOKEN does not have write access to this repository. To enable cross-repo PR updates, ensure the workflow runs with a token that has access to all repos, or configure repository permissions. Skipping this repo."
                  SUCCESS=true  # Mark as "success" to continue with other repos
                fi
              elif [ "$HTTP_CODE" = "404" ]; then
                if [ "$IS_MAIN_REPO" = "true" ]; then
                  echo "Error: PR #$PR_NUMBER not found in main repo ($OWNER/$REPO)"
                  exit 1
                else
                  echo "Info: No open PR found for branch $BRANCH in $OWNER/$REPO. This is expected if the PR hasn't been created yet or was closed. Skipping this repo."
                  SUCCESS=true  # Mark as "success" to continue with other repos
                fi
              elif [ "$HTTP_CODE" = "429" ]; then
                RETRY_COUNT=$((RETRY_COUNT + 1))
                if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                  DELAY=$((RETRY_DELAY * (2 ** RETRY_COUNT)))
                  echo "Rate limited (429). Retrying in $DELAY seconds... (attempt $RETRY_COUNT/$MAX_RETRIES)"
                  sleep $DELAY
                else
                  if [ "$IS_MAIN_REPO" = "true" ]; then
                    echo "Error: Rate limited (429) after $MAX_RETRIES retries. Failed to update PR in main repo ($OWNER/$REPO)"
                    exit 1
                  else
                    echo "Warning: Rate limited (429) after $MAX_RETRIES retries. Skipping PR update in $OWNER/$REPO"
                    SUCCESS=true  # Mark as "success" to continue with other repos
                  fi
                fi
              else
                if [ "$IS_MAIN_REPO" = "true" ]; then
                  echo "Error: Failed to update PR in main repo ($OWNER/$REPO). HTTP $HTTP_CODE: $RESPONSE_BODY"
                  exit 1
                else
                  echo "Warning: Failed to update PR in $OWNER/$REPO. HTTP $HTTP_CODE: $RESPONSE_BODY. Skipping this repo."
                  SUCCESS=true  # Mark as "success" to continue with other repos
                fi
              fi
            done
          done
          
          if [ "$MAIN_REPO_UPDATED" = "false" ]; then
            echo "Warning: Main repo PR was not updated. This may indicate an issue with the workflow configuration."
          fi
`
}
