package execmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestExecModuleRunsCommandsAndAwaitsPromise(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	script := `
const ghaExec = require("@actions/exec");

module.exports = async function () {
  const chunks = [];
  const result = await ghaExec.exec("go", ["env", "GOOS"], {
    silent: true,
    listeners: {
      stdout: (chunk) => chunks.push(chunk)
    }
  });

  return {
    exitCode: result.exitCode,
    stdout: result.stdout.trim(),
    chunks: chunks.join("").trim()
  };
};
`
	if err := os.WriteFile(entryPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write entry.js: %v", err)
	}

	settings := &gharuntime.Settings{
		ScriptPath:       entryPath,
		WorkingDirectory: tempDir,
		AmbientEnvironment: map[string]string{
			"PATH": os.Getenv("PATH"),
		},
	}
	settings.State = &gharuntime.State{Environment: settings.ProcessEnv()}

	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings}),
	)
	if err != nil {
		t.Fatalf("run exec script: %v", err)
	}

	exported, ok := result.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result.Export())
	}

	if got, want := exported["exitCode"], int64(0); got != want {
		t.Fatalf("exitCode = %v, want %v", got, want)
	}
	if got := exported["stdout"]; got == "" {
		t.Fatalf("stdout = %v, want non-empty", got)
	}
	if got, want := exported["chunks"], exported["stdout"]; got != want {
		t.Fatalf("chunks = %v, want %v", got, want)
	}
}
