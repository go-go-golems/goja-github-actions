package cmds

import (
	"bytes"
	"testing"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

func TestMaybePrintScriptResultSuppressesNil(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, nil, true, nil); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buffer = %q, want empty", got)
	}
}

func TestMaybePrintScriptResultPrintsWhenForced(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, map[string]interface{}{"ok": true}, true, nil); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got == "" {
		t.Fatal("buffer is empty, want JSON output")
	}
}

func TestMaybePrintScriptResultSuppressesNonInteractiveByDefault(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, map[string]interface{}{"ok": true}, false, nil); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buffer = %q, want empty", got)
	}
}

func TestMaybePrintScriptResultSuppressesWhenHumanOutputAlreadyRendered(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	state := &gharuntime.State{HumanOutputRendered: true}
	if err := maybePrintScriptResult(&buf, map[string]interface{}{"ok": true}, false, state); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buffer = %q, want empty", got)
	}
}
