package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunScriptExecutesEntrypointAndLocalRequire(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	helperPath := filepath.Join(tempDir, "helper.js")

	if err := os.WriteFile(helperPath, []byte("module.exports = { answer: 42 };"), 0o644); err != nil {
		t.Fatalf("write helper.js: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte(`
const helper = require("./helper.js");

module.exports = function () {
  return {
    answer: helper.answer,
    workspace: process.env.GITHUB_WORKSPACE,
    cwd: process.cwd()
  };
};
`), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	settings := &Settings{
		ScriptPath:       entryPath,
		WorkingDirectory: tempDir,
		Workspace:        "/workspace",
		AmbientEnvironment: map[string]string{
			"PATH": os.Getenv("PATH"),
		},
	}

	result, err := RunScript(context.Background(), settings)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("exported result type = %T, want map[string]interface{}", result.Export())
	}

	if got, want := exported["answer"], int64(42); got != want {
		t.Fatalf("answer = %v, want %v", got, want)
	}
	if got, want := exported["workspace"], "/workspace"; got != want {
		t.Fatalf("workspace = %v, want %v", got, want)
	}
	if got, want := exported["cwd"], tempDir; got != want {
		t.Fatalf("cwd = %v, want %v", got, want)
	}
}

func TestRunScriptAwaitsAsyncEntrypoint(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")

	if err := os.WriteFile(entryPath, []byte(`
module.exports = async function () {
  await Promise.resolve();
  return {
    ok: true,
    cwd: process.cwd()
  };
};
`), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	settings := &Settings{
		ScriptPath:         entryPath,
		WorkingDirectory:   tempDir,
		AmbientEnvironment: map[string]string{},
	}

	result, err := RunScript(context.Background(), settings)
	if err != nil {
		t.Fatalf("run async script: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("exported result type = %T, want map[string]interface{}", result.Export())
	}

	if got, want := exported["ok"], true; got != want {
		t.Fatalf("ok = %v, want %v", got, want)
	}
	if got, want := exported["cwd"], tempDir; got != want {
		t.Fatalf("cwd = %v, want %v", got, want)
	}
}
