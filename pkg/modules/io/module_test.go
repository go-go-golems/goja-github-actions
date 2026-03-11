package iomodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestIOModuleSupportsFileOperations(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	script := `
const io = require("@actions/io");

module.exports = function () {
  io.mkdirP("nested/folder");
  io.writeFile("nested/folder/output.txt", "world");
  io.cp("source", "copied");
  const beforeMove = io.readdir("copied");
  io.mv("nested/folder/output.txt", "nested/folder/renamed.txt");
  const content = io.readFile("nested/folder/renamed.txt");
  const goBinary = io.which("go", true);
  io.rmRF("copied");

  return {
    beforeMove,
    content,
    copiedExistsAfterRemove: io.readdir(".").includes("copied"),
    goBinary
  };
};
`
	if err := os.WriteFile(entryPath, []byte(script), 0o644); err != nil {
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
		Spec(&Dependencies{Settings: settings}),
	)
	if err != nil {
		t.Fatalf("run io script: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result.Export())
	}

	entries, ok := exported["beforeMove"].([]string)
	if !ok || len(entries) != 1 || entries[0] != "a.txt" {
		t.Fatalf("beforeMove = %#v, want [a.txt]", exported["beforeMove"])
	}
	if got, want := exported["content"], "world"; got != want {
		t.Fatalf("content = %v, want %v", got, want)
	}
	if got, want := exported["copiedExistsAfterRemove"], false; got != want {
		t.Fatalf("copiedExistsAfterRemove = %v, want %v", got, want)
	}
	if got := exported["goBinary"]; got == "" {
		t.Fatalf("goBinary = %v, want non-empty", got)
	}
}
