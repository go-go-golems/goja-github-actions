package runnerfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileCreatesParentDirectoriesAndAppends(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "env.txt")
	writer := EnvFile{Path: path}

	if err := writer.Set("HELLO", "world"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if err := writer.SetMultiline("MULTI", "a\r\nb"); err != nil {
		t.Fatalf("set multiline env: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}

	expected := "HELLO=world\nMULTI<<__GOJA_GHA_ENV__\na\nb\n__GOJA_GHA_ENV__\n"
	if string(content) != expected {
		t.Fatalf("env file content = %q, want %q", string(content), expected)
	}
}

func TestOutputFileAndPathFileAppend(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "out.txt")
	pathPath := filepath.Join(tempDir, "path.txt")

	if err := (OutputFile{Path: outputPath}).Set("result", "42"); err != nil {
		t.Fatalf("set output: %v", err)
	}
	if err := (PathFile{Path: pathPath}).Add("/tmp/bin"); err != nil {
		t.Fatalf("add path: %v", err)
	}

	outputContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if got, want := string(outputContent), "result=42\n"; got != want {
		t.Fatalf("output file content = %q, want %q", got, want)
	}

	pathContent, err := os.ReadFile(pathPath)
	if err != nil {
		t.Fatalf("read path file: %v", err)
	}
	if got, want := string(pathContent), "/tmp/bin\n"; got != want {
		t.Fatalf("path file content = %q, want %q", got, want)
	}
}

func TestSummaryFileAppendAndClear(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "summary.md")
	writer := SummaryFile{Path: path}

	if err := writer.Append("line1\r\nline2\n"); err != nil {
		t.Fatalf("append summary: %v", err)
	}
	if err := writer.Clear(); err != nil {
		t.Fatalf("clear summary: %v", err)
	}
	if err := writer.Append("# Heading\n"); err != nil {
		t.Fatalf("append summary after clear: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read summary file: %v", err)
	}
	if got, want := string(content), "# Heading\n"; got != want {
		t.Fatalf("summary file content = %q, want %q", got, want)
	}
}
