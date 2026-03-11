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
				"enabled":              true,
				"allowed_actions":      "selected",
				"sha_pinning_required": true,
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
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["status"], "passed"; got != want {
		t.Fatalf("summary.status = %v, want %v", got, want)
	}
	if got, want := summary["findingCount"], float64(0); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok {
		t.Fatalf("findings = %#v, want slice", result["findings"])
	}
	if len(findingsValue) != 0 {
		t.Fatalf("findings len = %d, want 0", len(findingsValue))
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
				"enabled":              true,
				"allowed_actions":      "all",
				"sha_pinning_required": false,
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
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["status"], "findings"; got != want {
		t.Fatalf("summary.status = %v, want %v", got, want)
	}
	if got, want := summary["highestSeverity"], "high"; got != want {
		t.Fatalf("summary.highestSeverity = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) < 2 {
		t.Fatalf("findings = %#v, want at least 2 findings", result["findings"])
	}
}

func TestPermissionsAuditExamplePrintsHumanReportWithoutJSONResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/actions/permissions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"enabled":              true,
				"allowed_actions":      "all",
				"sha_pinning_required": false,
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
		"Finding count",
		"Highest severity",
		"Findings",
		"allowed-actions-not-restricted",
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

func TestPinThirdPartyActionsExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
on:
  push:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: acme/internal-action@0123456789abcdef0123456789abcdef01234567
      - uses: docker://alpine:3.20
      - uses: ./.github/actions/local-helper
  reusable:
    uses: acme/reusable/.github/workflows/build.yml@main
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/pin-third-party-actions.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha pin-third-party-actions failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "pin-third-party-actions"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(2); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	if got, want := summary["highestSeverity"], "high"; got != want {
		t.Fatalf("summary.highestSeverity = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 2 {
		t.Fatalf("findings = %#v, want 2 findings", result["findings"])
	}

	first, ok := findingsValue[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first finding = %#v, want map", findingsValue[0])
	}
	evidence, ok := first["evidence"].(map[string]interface{})
	if !ok {
		t.Fatalf("first finding evidence = %#v, want map", first["evidence"])
	}
	if got, want := evidence["path"], ".github/workflows/ci.yml"; got != want {
		t.Fatalf("evidence.path = %v, want %v", got, want)
	}
}

func TestPinThirdPartyActionsExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/pin-third-party-actions.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha pin-third-party-actions human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"Pin Third-Party Actions",
		"Finding count",
		"pin-third-party-actions",
		".github/workflows/ci.yml",
		"actions/checkout@v5",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}

func TestCheckoutPersistCredsExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
jobs:
  safe:
    runs-on: ubuntu-latest
    steps:
      - name: safe checkout
        uses: actions/checkout@v6
        with:
          persist-credentials: false
  unsafe:
    runs-on: ubuntu-latest
    steps:
      - name: unsafe checkout
        uses: actions/checkout@v6
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/checkout-persist-creds.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha checkout-persist-creds failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "checkout-persist-creds"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(1); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 1 {
		t.Fatalf("findings = %#v, want 1 finding", result["findings"])
	}

	first, ok := findingsValue[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first finding = %#v, want map", findingsValue[0])
	}
	evidence, ok := first["evidence"].(map[string]interface{})
	if !ok {
		t.Fatalf("first finding evidence = %#v, want map", first["evidence"])
	}
	if got, want := evidence["stepName"], "unsafe checkout"; got != want {
		t.Fatalf("evidence.stepName = %v, want %v", got, want)
	}
}

func TestCheckoutPersistCredsExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/checkout-persist-creds.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha checkout-persist-creds human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"Checkout Persist Credentials",
		"checkout-persist-creds",
		".github/workflows/ci.yml",
		"actions/checkout@v6",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}

func TestNoWriteAllExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    permissions: write-all
    steps:
      - run: echo hi
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/no-write-all.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha no-write-all failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "no-write-all"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(2); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 2 {
		t.Fatalf("findings = %#v, want 2 findings", result["findings"])
	}
}

func TestNoWriteAllExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: CI
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo hi
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/no-write-all.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha no-write-all human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"No Write-All",
		"no-write-all",
		".github/workflows/ci.yml",
		"workflow",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}

func TestPullRequestTargetReviewExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: PR Target Review
on:
  pull_request_target:
jobs:
  dangerous:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout PR head
        uses: actions/checkout@v6
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          repository: ${{ github.event.pull_request.head.repo.full_name }}
      - name: Run tests
        run: make test
`
	if err := os.WriteFile(filepath.Join(workflowDir, "dangerous.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/pull-request-target-review.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha pull-request-target-review failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "pull-request-target-review"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	if got, want := result["reviewedWorkflowCount"], float64(1); got != want {
		t.Fatalf("reviewedWorkflowCount = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(2); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	if got, want := summary["highestSeverity"], "critical"; got != want {
		t.Fatalf("summary.highestSeverity = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 2 {
		t.Fatalf("findings = %#v, want 2 findings", result["findings"])
	}

	second, ok := findingsValue[1].(map[string]interface{})
	if !ok {
		t.Fatalf("second finding = %#v, want map", findingsValue[1])
	}
	evidence, ok := second["evidence"].(map[string]interface{})
	if !ok {
		t.Fatalf("second finding evidence = %#v, want map", second["evidence"])
	}
	if got, want := evidence["jobId"], "dangerous"; got != want {
		t.Fatalf("evidence.jobId = %v, want %v", got, want)
	}
	if got, want := evidence["runStepCount"], float64(1); got != want {
		t.Fatalf("evidence.runStepCount = %v, want %v", got, want)
	}
}

func TestPullRequestTargetReviewExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: Metadata only
on:
  pull_request_target:
jobs:
  label:
    runs-on: ubuntu-latest
    steps:
      - name: Inspect metadata
        run: echo labels
`
	if err := os.WriteFile(filepath.Join(workflowDir, "metadata.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/pull-request-target-review.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha pull-request-target-review human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"Pull Request Target Review",
		"Reviewed workflows",
		"pull-request-target-review",
		".github/workflows/metadata.yml",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}

func TestWorkflowRunReviewExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: Follow-up
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
jobs:
  dangerous:
    runs-on: ubuntu-latest
    steps:
      - name: Download upstream artifacts
        uses: actions/download-artifact@v5
      - name: Checkout upstream head
        uses: actions/checkout@v6
        with:
          ref: ${{ github.event.workflow_run.head_sha }}
      - name: Execute artifact content
        run: ./artifact/run.sh
`
	if err := os.WriteFile(filepath.Join(workflowDir, "follow-up.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/workflow-run-review.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha workflow-run-review failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "workflow-run-review"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	if got, want := result["reviewedWorkflowCount"], float64(1); got != want {
		t.Fatalf("reviewedWorkflowCount = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(3); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	if got, want := summary["highestSeverity"], "critical"; got != want {
		t.Fatalf("summary.highestSeverity = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 3 {
		t.Fatalf("findings = %#v, want 3 findings", result["findings"])
	}

	third, ok := findingsValue[2].(map[string]interface{})
	if !ok {
		t.Fatalf("third finding = %#v, want map", findingsValue[2])
	}
	evidence, ok := third["evidence"].(map[string]interface{})
	if !ok {
		t.Fatalf("third finding evidence = %#v, want map", third["evidence"])
	}
	if got, want := evidence["runStepCount"], float64(1); got != want {
		t.Fatalf("evidence.runStepCount = %v, want %v", got, want)
	}
}

func TestWorkflowRunReviewExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: Follow-up metadata
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
jobs:
  summarize:
    runs-on: ubuntu-latest
    steps:
      - name: Emit summary
        run: echo done
`
	if err := os.WriteFile(filepath.Join(workflowDir, "follow-up.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/workflow-run-review.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha workflow-run-review human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"Workflow Run Review",
		"Reviewed workflows",
		"workflow-run-review",
		".github/workflows/follow-up.yml",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}

func TestReusableWorkflowTrustExampleViaCLI(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: Reusable Trust
on:
  workflow_dispatch:
jobs:
  local-safe:
    uses: ./.github/workflows/internal.yml
  same-owner-unsafe:
    uses: acme/shared/.github/workflows/build.yml@main
  external-pinned:
    uses: vendor/security/.github/workflows/scan.yml@0123456789abcdef0123456789abcdef01234567
`
	if err := os.WriteFile(filepath.Join(workflowDir, "reusable.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/reusable-workflow-trust.js",
		"--cwd", tempDir,
		"--workspace", workspace,
		"--json-result",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha reusable-workflow-trust failed: %v\n%s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode json result: %v\n%s", err, string(output))
	}

	if got, want := result["scriptId"], "reusable-workflow-trust"; got != want {
		t.Fatalf("scriptId = %v, want %v", got, want)
	}
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary = %#v, want map", result["summary"])
	}
	if got, want := summary["findingCount"], float64(2); got != want {
		t.Fatalf("summary.findingCount = %v, want %v", got, want)
	}
	if got, want := summary["highestSeverity"], "high"; got != want {
		t.Fatalf("summary.highestSeverity = %v, want %v", got, want)
	}
	findingsValue, ok := result["findings"].([]interface{})
	if !ok || len(findingsValue) != 2 {
		t.Fatalf("findings = %#v, want 2 findings", result["findings"])
	}
}

func TestReusableWorkflowTrustExamplePrintsHumanReport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workspace workflows: %v", err)
	}

	workflow := `name: Reusable Trust
jobs:
  shared:
    uses: acme/shared/.github/workflows/build.yml@main
`
	if err := os.WriteFile(filepath.Join(workflowDir, "reusable.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/goja-gha", "run",
		"--script", "./examples/reusable-workflow-trust.js",
		"--cwd", tempDir,
		"--workspace", workspace,
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GITHUB_REPOSITORY=acme/widgets",
		"GITHUB_WORKSPACE="+workspace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goja-gha reusable-workflow-trust human report failed: %v\n%s", err, string(output))
	}

	rendered := string(output)
	for _, needle := range []string{
		"Reusable Workflow Trust",
		"reusable-workflow-unpinned",
		".github/workflows/reusable.yml",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("human report missing %q:\n%s", needle, rendered)
		}
	}
}
