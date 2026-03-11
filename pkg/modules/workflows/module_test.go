package workflowmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestWorkflowsModuleParsesWorkflowData(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	scriptDir := filepath.Join(root, "scripts")
	workflowDir := filepath.Join(workspace, ".github", "workflows")

	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}

	workflowContent := `name: Example
on: [push]
permissions: write-all
jobs:
  build:
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          persist-credentials: false
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflowContent), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	scriptPath := filepath.Join(scriptDir, "entry.js")
	script := `const workflows = require("@goja-gha/workflows");
module.exports = function () {
  return workflows.parseAll();
};`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	settings := &runtime.Settings{
		ScriptPath:         scriptPath,
		Workspace:          workspace,
		AmbientEnvironment: map[string]string{},
		State:              &runtime.State{Environment: map[string]string{}},
	}

	value, err := runtime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings}),
	)
	if err != nil {
		t.Fatalf("RunScriptWithModules: %v", err)
	}

	result, ok := value.Export().([]map[string]interface{})
	if !ok {
		t.Fatalf("result = %#v, want []map[string]interface{}", value.Export())
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	doc := result[0]
	if doc["path"] != ".github/workflows/ci.yml" {
		t.Fatalf("doc.path = %#v, want .github/workflows/ci.yml", doc["path"])
	}
	uses, ok := doc["uses"].([]map[string]interface{})
	if !ok || len(uses) != 1 {
		t.Fatalf("uses = %#v, want one entry", doc["uses"])
	}
	checkoutSteps, ok := doc["checkoutSteps"].([]map[string]interface{})
	if !ok || len(checkoutSteps) != 1 {
		t.Fatalf("checkoutSteps = %#v, want one entry", doc["checkoutSteps"])
	}
}
