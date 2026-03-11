package workflows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAllExtractsWorkflowData(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}

	content := `name: CI
on:
  push:
  pull_request:
permissions: read-all
jobs:
  build:
    permissions:
      contents: read
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Lint
        uses: golangci/golangci-lint-action@v8
  reusable:
    uses: acme/reusable/.github/workflows/build.yml@main
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	docs, err := ParseAll(workspace)
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(ParseAll()) = %d, want 1", len(docs))
	}

	doc := docs[0]
	if doc.FileName != "ci.yml" {
		t.Fatalf("FileName = %q, want ci.yml", doc.FileName)
	}
	if doc.Path != ".github/workflows/ci.yml" {
		t.Fatalf("Path = %q, want .github/workflows/ci.yml", doc.Path)
	}
	if doc.Name != "CI" {
		t.Fatalf("Name = %q, want CI", doc.Name)
	}
	if len(doc.TriggerNames) != 2 || doc.TriggerNames[0] != "push" || doc.TriggerNames[1] != "pull_request" {
		t.Fatalf("TriggerNames = %#v, want [push pull_request]", doc.TriggerNames)
	}
	if len(doc.Uses) != 3 {
		t.Fatalf("len(Uses) = %d, want 3", len(doc.Uses))
	}
	if doc.Uses[0].Uses != "actions/checkout@v4" || doc.Uses[0].Kind != "step" {
		t.Fatalf("Uses[0] = %#v, want checkout step", doc.Uses[0])
	}
	if doc.Uses[2].Uses != "acme/reusable/.github/workflows/build.yml@main" || doc.Uses[2].Kind != "job" {
		t.Fatalf("Uses[2] = %#v, want reusable workflow ref", doc.Uses[2])
	}
	if len(doc.CheckoutSteps) != 1 {
		t.Fatalf("len(CheckoutSteps) = %d, want 1", len(doc.CheckoutSteps))
	}
	if doc.CheckoutSteps[0].PersistCredentials != nil {
		t.Fatalf("PersistCredentials = %#v, want nil", doc.CheckoutSteps[0].PersistCredentials)
	}
	if len(doc.Permissions) != 2 {
		t.Fatalf("len(Permissions) = %d, want 2", len(doc.Permissions))
	}
	if doc.Permissions[0].Scope != "workflow" || doc.Permissions[0].Kind != "scalar" || doc.Permissions[0].Value != "read-all" {
		t.Fatalf("Permissions[0] = %#v, want workflow read-all", doc.Permissions[0])
	}
	values, ok := doc.Permissions[1].Value.(map[string]string)
	if !ok {
		t.Fatalf("Permissions[1].Value = %#v, want map[string]string", doc.Permissions[1].Value)
	}
	if doc.Permissions[1].Scope != "job" || doc.Permissions[1].JobID != "build" || values["contents"] != "read" {
		t.Fatalf("Permissions[1] = %#v, want job contents:read", doc.Permissions[1])
	}
}

func TestParseFileAcceptsBareFileName(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "lint.yaml"), []byte("name: Lint\n"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	doc, err := ParseFile(workspace, "lint.yaml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if doc.Path != ".github/workflows/lint.yaml" {
		t.Fatalf("Path = %q, want .github/workflows/lint.yaml", doc.Path)
	}
}
