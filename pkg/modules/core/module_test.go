package coremodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestCoreModuleHandlesInputsAndRunnerFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	envFile := filepath.Join(tempDir, "env.txt")
	outputFile := filepath.Join(tempDir, "output.txt")
	pathFile := filepath.Join(tempDir, "path.txt")
	summaryFile := filepath.Join(tempDir, "summary.md")

	script := `
const core = require("@actions/core");

module.exports = function () {
  const input = core.getInput("name", { required: true });
  const flag = core.getBooleanInput("flag");
  const items = core.getMultilineInput("items");
  core.setOutput("result", "42");
  core.exportVariable("HELLO", "world");
  core.addPath("/tmp/bin");
  core.summary.addHeading("Heading").addRaw("Hello\n").write();
  return {
    input,
    flag,
    items,
    hello: process.env.HELLO,
    path: process.env.PATH
  };
};
`
	if err := os.WriteFile(entryPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	settings := &gharuntime.Settings{
		ScriptPath:        entryPath,
		WorkingDirectory:  tempDir,
		RunnerEnvFile:     envFile,
		RunnerOutputFile:  outputFile,
		RunnerPathFile:    pathFile,
		RunnerSummaryFile: summaryFile,
		AmbientEnvironment: map[string]string{
			"INPUT_NAME":  "  Manuel  ",
			"INPUT_FLAG":  "true",
			"INPUT_ITEMS": "one\ntwo\n",
			"PATH":        "/usr/bin",
		},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(NewDependencies(settings)),
	)
	if err != nil {
		t.Fatalf("run script with core module: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result.Export())
	}

	if got, want := exported["input"], "Manuel"; got != want {
		t.Fatalf("input = %v, want %v", got, want)
	}
	if got, want := exported["flag"], true; got != want {
		t.Fatalf("flag = %v, want %v", got, want)
	}
	items, ok := exported["items"].([]string)
	if !ok || len(items) != 2 || items[0] != "one" || items[1] != "two" {
		t.Fatalf("items = %#v, want [one two]", exported["items"])
	}
	if got, want := exported["hello"], "world"; got != want {
		t.Fatalf("hello = %v, want %v", got, want)
	}
	if got, want := exported["path"], "/tmp/bin"+string(os.PathListSeparator)+"/usr/bin"; got != want {
		t.Fatalf("path = %v, want %v", got, want)
	}

	assertFileContent(t, envFile, "HELLO=world\n")
	assertFileContent(t, outputFile, "result=42\n")
	assertFileContent(t, pathFile, "/tmp/bin\n")
	assertFileContent(t, summaryFile, "# Heading\nHello\n")
}

func TestCoreSetFailedSetsExitCode(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	if err := os.WriteFile(entryPath, []byte(`
const core = require("@actions/core");
module.exports = function () {
  core.setFailed("boom");
  return { exitCode: process.exitCode };
};
`), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	settings := &gharuntime.Settings{
		ScriptPath:         entryPath,
		WorkingDirectory:   tempDir,
		AmbientEnvironment: map[string]string{},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(NewDependencies(settings)),
	)
	if err != nil {
		t.Fatalf("run script with setFailed: %v", err)
	}

	exported := result.Export().(map[string]interface{})
	if got, want := exported["exitCode"], int64(1); got != want {
		t.Fatalf("exitCode = %v, want %v", got, want)
	}
	if got, want := settings.State.ExitCode, 1; got != want {
		t.Fatalf("state exit code = %d, want %d", got, want)
	}
	if got, want := settings.State.FailureMessage, "boom"; got != want {
		t.Fatalf("failure message = %q, want %q", got, want)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(content); got != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}
