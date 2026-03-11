package cmds

import (
	"bytes"
	"testing"
)

func TestMaybePrintScriptResultSuppressesNil(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, nil, true); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buffer = %q, want empty", got)
	}
}

func TestMaybePrintScriptResultPrintsWhenForced(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, map[string]interface{}{"ok": true}, true); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got == "" {
		t.Fatal("buffer is empty, want JSON output")
	}
}

func TestMaybePrintScriptResultSuppressesNonInteractiveByDefault(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := maybePrintScriptResult(&buf, map[string]interface{}{"ok": true}, false); err != nil {
		t.Fatalf("maybePrintScriptResult returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("buffer = %q, want empty", got)
	}
}
