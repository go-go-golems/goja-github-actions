package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(wd)
}

func TestPermissionsAuditExampleViaCLI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/actions/permissions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"enabled":         true,
				"allowed_actions": "selected",
			})
		case "/repos/acme/widgets/actions/permissions/selected-actions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"patterns_allowed": []string{"acme/*"},
			})
		case "/repos/acme/widgets/actions/permissions/workflow":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"default_workflow_permissions":     "read",
				"can_approve_pull_request_reviews": false,
			})
		case "/repos/acme/widgets/actions/workflows":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 2,
				"workflows": []map[string]interface{}{
					{"id": 1001, "name": "CI", "path": ".github/workflows/ci.yml"},
					{"id": 1002, "name": "Lint", "path": ".github/workflows/lint.yml"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".github", "workflows", "ci.yml"), []byte("name: CI\n"), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	outputFile := filepath.Join(tempDir, "github-output.txt")
	summaryFile := filepath.Join(tempDir, "summary.md")

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/permissions-audit.js",
		"--cwd", workspace,
		"--event-path", "./testdata/events/workflow_dispatch.json",
		"--runner-output-file", outputFile,
		"--runner-summary-file", summaryFile,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_API_URL="+server.URL,
		"GITHUB_ACTOR=manuel",
		"GITHUB_EVENT_NAME=workflow_dispatch",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
		"GITHUB_TOKEN=secret-token",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha permissions-audit failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["repository"], "acme/widgets"; got != want {
		t.Fatalf("repository = %v, want %v", got, want)
	}
	if got, want := result["workspace"], workspace; got != want {
		t.Fatalf("workspace = %v, want %v", got, want)
	}
	if got, want := result["workflowCount"], float64(2); got != want {
		t.Fatalf("workflowCount = %v, want %v", got, want)
	}
	if got := result["localWorkflowFiles"]; got == nil {
		t.Fatalf("localWorkflowFiles = nil, want non-nil")
	}
	if got, want := result["selectedActionsStatus"], "fetched"; got != want {
		t.Fatalf("selectedActionsStatus = %v, want %v", got, want)
	}

	outputBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(outputBytes), "audit=") {
		t.Fatalf("runner output file missing audit payload: %s", string(outputBytes))
	}

	summaryBytes, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	if !strings.Contains(string(summaryBytes), "GitHub Actions Audit") {
		t.Fatalf("summary missing heading: %s", string(summaryBytes))
	}
}

func TestPermissionsAuditSkipsSelectedActionsWhenPolicyIsNotSelected(t *testing.T) {
	t.Parallel()

	selectedActionsCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/actions/permissions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"enabled":         true,
				"allowed_actions": "all",
			})
		case "/repos/acme/widgets/actions/permissions/selected-actions":
			selectedActionsCalls++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Conflict",
			})
		case "/repos/acme/widgets/actions/permissions/workflow":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"default_workflow_permissions":     "read",
				"can_approve_pull_request_reviews": false,
			})
		case "/repos/acme/widgets/actions/workflows":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"workflows": []map[string]interface{}{
					{"id": 1001, "name": "CI", "path": ".github/workflows/ci.yml"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".github", "workflows", "ci.yml"), []byte("name: CI\n"), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/permissions-audit.js",
		"--cwd", workspace,
		"--event-path", "./testdata/events/workflow_dispatch.json",
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_API_URL="+server.URL,
		"GITHUB_ACTOR=manuel",
		"GITHUB_EVENT_NAME=workflow_dispatch",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
		"GITHUB_TOKEN=secret-token",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha permissions-audit failed: %v\n%s", err, string(output))
	}

	if selectedActionsCalls != 0 {
		t.Fatalf("selected-actions endpoint called %d times, want 0", selectedActionsCalls)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["selectedActionsStatus"], "skipped-not-selected-policy"; got != want {
		t.Fatalf("selectedActionsStatus = %v, want %v", got, want)
	}
	if got := result["selectedActions"]; got != nil {
		t.Fatalf("selectedActions = %v, want nil", got)
	}
	files, ok := result["localWorkflowFiles"].([]interface{})
	if !ok || len(files) != 1 || files[0] != "ci.yml" {
		t.Fatalf("localWorkflowFiles = %#v, want [ci.yml]", result["localWorkflowFiles"])
	}
}

func TestPermissionsAuditExamplePrintsHumanReportWithoutJSONResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/actions/permissions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"enabled":         true,
				"allowed_actions": "all",
			})
		case "/repos/acme/widgets/actions/permissions/workflow":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"default_workflow_permissions":     "read",
				"can_approve_pull_request_reviews": false,
			})
		case "/repos/acme/widgets/actions/workflows":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"workflows": []map[string]interface{}{
					{"id": 1001, "name": "CI", "path": ".github/workflows/ci.yml"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".github", "workflows", "ci.yml"), []byte("name: CI\n"), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/permissions-audit.js",
		"--cwd", workspace,
		"--event-path", "./testdata/events/workflow_dispatch.json",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_API_URL="+server.URL,
		"GITHUB_ACTOR=manuel",
		"GITHUB_EVENT_NAME=workflow_dispatch",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
		"GITHUB_TOKEN=secret-token",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha permissions-audit human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"GitHub Actions Audit",
		"Inspected acme/widgets",
		"Workspace",
		workspace,
		"Allowed actions",
		"selected-actions only applies when allowed_actions == \"selected\"",
		"Workflows",
		"CI",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
	if strings.Contains(rendered, "\"repository\"") {
		t.Fatalf("human report unexpectedly contained raw JSON payload:\n%s", rendered)
	}
}

func TestListWorkflowsExampleViaCLI(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/list-workflows.js",
		"--cwd", root,
		"--json-result",
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_WORKSPACE="+root,
		"GITHUB_REPOSITORY=go-go-golems/goja-github-actions",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha list-workflows failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got := strings.TrimSpace(result["gitRoot"].(string)); got != root {
		t.Fatalf("gitRoot = %q, want %q", got, root)
	}
	files, ok := result["workflowFiles"].([]interface{})
	if !ok || len(files) == 0 {
		t.Fatalf("workflowFiles = %#v, want non-empty", result["workflowFiles"])
	}
}
