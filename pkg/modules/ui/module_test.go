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

func TestConsecutiveKVBlocksCollapsed(t *testing.T) {
	t.Parallel()
	report := &reportBuilder{
		title: "Test",
		blocks: []reportBlock{
			kvBlock{Label: "Repo", Value: "acme/widgets"},
			kvBlock{Label: "Branch", Value: "main"},
			kvBlock{Label: "Status", Value: "ok"},
		},
	}
	rendered := renderTextReport(report, false)
	// Consecutive kv blocks should NOT have blank lines between them
	if strings.Contains(rendered, "acme/widgets\n\n") {
		t.Fatalf("expected no blank line between consecutive kv blocks:\n%s", rendered)
	}
	if !strings.Contains(rendered, "acme/widgets\n") {
		t.Fatalf("expected kv value present:\n%s", rendered)
	}
}

func TestBlankLineBetweenDifferentBlockTypes(t *testing.T) {
	t.Parallel()
	report := &reportBuilder{
		title: "Test",
		blocks: []reportBlock{
			statusBlock{Kind: "ok", Text: "All good"},
			kvBlock{Label: "Repo", Value: "acme/widgets"},
		},
	}
	rendered := renderTextReport(report, false)
	// Should have blank line between status and kv
	if !strings.Contains(rendered, "All good\n\n") {
		t.Fatalf("expected blank line between status and kv blocks:\n%s", rendered)
	}
}

func TestBracketStatusLabels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind string
		want string
	}{
		{"ok", "[ OK ]"},
		{"warn", "[WARN]"},
		{"error", "[ERR ]"},
		{"skip", "[SKIP]"},
		{"info", "[INFO]"},
	}
	for _, tc := range cases {
		got := styleStatusLabel(tc.kind, false)
		if got != tc.want {
			t.Errorf("styleStatusLabel(%q, false) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestDescriptionRendersWithWordWrap(t *testing.T) {
	t.Parallel()
	report := &reportBuilder{
		title:       "Audit",
		description: "This is a long description that should be word-wrapped at approximately seventy characters to keep the terminal output readable and pleasant.",
	}
	rendered := renderTextReport(report, false)
	if !strings.Contains(rendered, "  This is a long description") {
		t.Fatalf("expected indented description:\n%s", rendered)
	}
	// Each line (after prefix removal) should be at most ~72 chars (2 indent + 70 content)
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "==") {
			if len(line) > 80 {
				t.Errorf("description line too long (%d chars): %q", len(line), line)
			}
		}
	}
}

func TestWordWrap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		width int
		want  int // expected number of lines
	}{
		{"short", 70, 1},
		{"", 70, 0},
		{"one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen", 20, 5},
	}
	for _, tc := range cases {
		lines := wordWrap(tc.input, tc.width)
		if len(lines) != tc.want {
			t.Errorf("wordWrap(%q, %d) = %d lines, want %d: %v", tc.input, tc.width, len(lines), tc.want, lines)
		}
		for _, line := range lines {
			// Allow single words longer than width
			if len(line) > tc.width && !strings.ContainsRune(line, ' ') {
				continue
			}
			if len(line) > tc.width {
				t.Errorf("line exceeds width %d: %q", tc.width, line)
			}
		}
	}
}

func TestFindingsBlockRendering(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	script := `
const ui = require("@goja-gha/ui");

module.exports = function () {
  var findings = [
    {
      ruleId: "no-write-all",
      severity: "high",
      message: "Workflow uses permissions: write-all",
      whyItMatters: "Broad write-all permissions give every permission category write access.",
      remediation: { summary: "Replace write-all with minimal permissions.", example: "permissions: { contents: read }" },
      evidence: { path: ".github/workflows/ci.yml", line: 5 }
    },
    {
      ruleId: "no-write-all",
      severity: "high",
      message: "Job build uses permissions: write-all",
      whyItMatters: "Broad write-all permissions give every permission category write access.",
      remediation: { summary: "Replace write-all with minimal permissions." },
      evidence: { path: ".github/workflows/ci.yml", line: 20 }
    },
    {
      ruleId: "unpinned-action",
      severity: "medium",
      message: "Action not pinned to SHA",
      whyItMatters: "Mutable refs can change without notice.",
      remediation: { summary: "Pin to full commit SHA." },
      evidence: { path: ".github/workflows/deploy.yml", line: 10, uses: "actions/checkout@v4" }
    }
  ];

  ui.report("Test Audit")
    .section("Findings", (s) => {
      s.findings(findings, { locationFields: ["path", "line", "uses"] });
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

	_, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings, Writer: &output}),
	)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}

	rendered := output.String()

	// Check grouped findings
	for _, needle := range []string{
		"no-write-all",
		"2 x HIGH",
		"unpinned-action",
		"1 x MEDIUM",
		"Why it matters",
		"Remediation",
		"Locations",
		".github/workflows/ci.yml",
		":5",
		":20",
		".github/workflows/deploy.yml",
		":10",
		"actions/checkout@v4",
		"permissions: { contents: read }",
	} {
		if !strings.Contains(rendered, needle) {
			t.Errorf("findings output missing %q:\n%s", needle, rendered)
		}
	}
}

func TestFindingsBlockEmptyFindings(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	entryPath := filepath.Join(tempDir, "entry.js")
	script := `
const ui = require("@goja-gha/ui");

module.exports = function () {
  ui.report("Test")
    .section("Findings", (s) => {
      s.findings([]);
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

	_, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		Spec(&Dependencies{Settings: settings, Writer: &output}),
	)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}

	rendered := output.String()
	// Should just have the title and section heading, no findings content
	if strings.Contains(rendered, "Why it matters") {
		t.Fatalf("empty findings should not render detail blocks:\n%s", rendered)
	}
}
