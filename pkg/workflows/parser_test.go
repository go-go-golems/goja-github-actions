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
        with:
          persist-credentials: false
          ref: ${{ github.event.pull_request.head.sha }}
          repository: ${{ github.event.pull_request.head.repo.full_name }}
      - name: Test
        run: go test ./...
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
	if doc.WorkflowRun != nil {
		t.Fatalf("WorkflowRun = %#v, want nil", doc.WorkflowRun)
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
	if doc.CheckoutSteps[0].PersistCredentials == nil || *doc.CheckoutSteps[0].PersistCredentials != "false" {
		t.Fatalf("PersistCredentials = %#v, want false", doc.CheckoutSteps[0].PersistCredentials)
	}
	if doc.CheckoutSteps[0].Ref == nil || *doc.CheckoutSteps[0].Ref != "${{ github.event.pull_request.head.sha }}" {
		t.Fatalf("Ref = %#v, want github.event.pull_request.head.sha", doc.CheckoutSteps[0].Ref)
	}
	if doc.CheckoutSteps[0].Repository == nil || *doc.CheckoutSteps[0].Repository != "${{ github.event.pull_request.head.repo.full_name }}" {
		t.Fatalf("Repository = %#v, want github.event.pull_request.head.repo.full_name", doc.CheckoutSteps[0].Repository)
	}
	if len(doc.RunSteps) != 1 {
		t.Fatalf("len(RunSteps) = %d, want 1", len(doc.RunSteps))
	}
	if doc.RunSteps[0].JobID != "build" || doc.RunSteps[0].Run != "go test ./..." {
		t.Fatalf("RunSteps[0] = %#v, want build/go test ./...", doc.RunSteps[0])
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

func TestParseWorkflowRunTriggerDetails(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	workflowDir := filepath.Join(workspace, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	content := `name: Follow-up
on:
  workflow_run:
    workflows:
      - CI
      - Lint
    types:
      - completed
    branches:
      - main
    branches-ignore:
      - release/*
`
	if err := os.WriteFile(filepath.Join(workflowDir, "follow-up.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	doc, err := ParseFile(workspace, "follow-up.yml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if doc.WorkflowRun == nil {
		t.Fatalf("WorkflowRun = nil, want details")
	}
	if len(doc.WorkflowRun.Workflows) != 2 || doc.WorkflowRun.Workflows[0] != "CI" || doc.WorkflowRun.Workflows[1] != "Lint" {
		t.Fatalf("WorkflowRun.Workflows = %#v, want [CI Lint]", doc.WorkflowRun.Workflows)
	}
	if len(doc.WorkflowRun.Types) != 1 || doc.WorkflowRun.Types[0] != "completed" {
		t.Fatalf("WorkflowRun.Types = %#v, want [completed]", doc.WorkflowRun.Types)
	}
	if len(doc.WorkflowRun.Branches) != 1 || doc.WorkflowRun.Branches[0] != "main" {
		t.Fatalf("WorkflowRun.Branches = %#v, want [main]", doc.WorkflowRun.Branches)
	}
	if len(doc.WorkflowRun.BranchesIgnore) != 1 || doc.WorkflowRun.BranchesIgnore[0] != "release/*" {
		t.Fatalf("WorkflowRun.BranchesIgnore = %#v, want [release/*]", doc.WorkflowRun.BranchesIgnore)
	}
}
