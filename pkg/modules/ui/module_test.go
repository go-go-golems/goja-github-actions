package uimodule

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestUIReportRendersHumanSummary(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	script := `
const ui = require("@goja-gha/ui");

module.exports = function () {
  ui.report("Audit")
    .status("ok", "Inspected repository")
    .kv("Repository", "acme/widgets")
    .section("Workflows", (s) => {
      s.table({
        columns: ["Name", "Path"],
        rows: [["CI", ".github/workflows/ci.yml"]]
      });
    })
    .render();

  return { ok: true };
};
`
	if err := os.WriteFile(entryPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	var output bytes.Buffer
	settings := &gharuntime.Settings{
		ScriptPath:         entryPath,
		WorkingDirectory:   tempDir,
		AmbientEnvironment: map[string]string{},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings, Writer: &output}),
	)
	if err != nil {
		t.Fatalf("run script with ui module: %v", err)
	}

	if got, want := result.Export().(map[string]interface{})["ok"], true; got != want {
		t.Fatalf("ok = %v, want %v", got, want)
	}
	rendered := output.String()
	for _, needle := range []string{"Audit", "Inspected repository", "Repository", "acme/widgets", "Workflows", "CI"} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("rendered output missing %q:\n%s", needle, rendered)
		}
	}
	if !settings.State.HumanOutputRendered {
		t.Fatal("expected HumanOutputRendered to be true")
	}
}

func TestUIReportIsSilentWhenJSONResultIsEnabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	if err := os.WriteFile(entryPath, []byte(`
const ui = require("@goja-gha/ui");

module.exports = function () {
  ui.report("Audit").status("ok", "done").render();
  return { ok: true };
};
`), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	var output bytes.Buffer
	settings := &gharuntime.Settings{
		ScriptPath:         entryPath,
		WorkingDirectory:   tempDir,
		JSONResult:         true,
		AmbientEnvironment: map[string]string{},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	_, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings, Writer: &output}),
	)
	if err != nil {
		t.Fatalf("run script with ui module: %v", err)
	}

	if got := output.String(); got != "" {
		t.Fatalf("output = %q, want empty", got)
	}
	if settings.State.HumanOutputRendered {
		t.Fatal("expected HumanOutputRendered to stay false")
	}
}
