---
Title: Investigation diary
Ticket: GHA-6
Status: active
Topics:
    - github-actions
    - goja
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Updated overview to describe workspace-first defaults
    - Path: pkg/helpdoc/01-user-guide.md
      Note: Updated user guide for workspace-first path resolution
    - Path: pkg/helpdoc/02-javascript-api.md
      Note: Updated JS API docs for workspace-first process.cwd semantics
    - Path: pkg/modules/exec/module_test.go
      Note: Tests workspace-first default exec cwd behavior
    - Path: pkg/modules/io/module_test.go
      Note: Tests workspace-first relative IO behavior
    - Path: pkg/runtime/script_runner_test.go
      Note: Tests process.cwd workspace precedence
ExternalSources: []
Summary: Chronological diary for the workspace-first execution-root change.
LastUpdated: 2026-03-11T12:35:05.354672193-04:00
WhatFor: Record why the workspace-first change was made, what files changed, and how the behavior was validated.
WhenToUse: Use when reviewing the change or continuing work on cwd/workspace semantics.
---


# Diary

## Goal

Capture the implementation of the workspace-first execution-root change so a later engineer can see what was changed, why it was changed, and how to validate the new behavior.

## Step 1: Replace split cwd/workspace defaults with an execution root

The goal of this step was to make JavaScript behave the way a GitHub Actions author expects: scripts should run relative to the checked-out repository workspace unless they explicitly opt into something else. The previous behavior made `GITHUB_WORKSPACE` and `process.cwd()` diverge by default, which forced scripts like `permissions-audit.js` to compensate manually.

I implemented a shared `ExecutionRoot()` helper on runtime settings, made `process.cwd()`, `@actions/io`, and default `@actions/exec` use it, and then simplified `permissions-audit.js` back to relative workflow discovery.

### Prompt Context

**User prompt (verbatim):** "alright, do that first before we write the scripts. Make a new ticket for this with small implementation document and tasks and then do it"

**Assistant interpretation:** Create a focused ticket for workspace-first execution semantics, implement the behavior change, document it briefly but concretely, and validate it before returning.

**Inferred user intent:** Remove a confusing path-resolution model before building more security scripts on top of it.

### What I did

- Created `GHA-6`.
- Added `Settings.ExecutionRoot()` in runtime settings.
- Updated `process.cwd()` and `process.workingDirectory` to use the execution root.
- Updated `@actions/io` relative path resolution to use the execution root.
- Updated default `@actions/exec` cwd to use the execution root.
- Simplified `examples/permissions-audit.js` back to `io.readdir(".github/workflows")`.
- Added tests covering runtime, IO, and exec behavior.
- Updated the user-facing docs and README to describe workspace-first behavior.
- Ran a real `permissions-audit.js` smoke against `/tmp/geppetto` using `--cwd /tmp --workspace /tmp/geppetto`.

### Why

- Workspace-first semantics are easier to understand and are the right default for repo-inspection scripts.
- The earlier split between workspace and working directory was a footgun for the planned security script pack.

### What worked

- The change was small and localized once the shared execution-root helper existed.
- Existing integration tests continued to pass.
- A CLI smoke test confirmed that `process.cwd()` now equals workspace when both `--cwd` and `--workspace` are provided.
- The `/tmp/geppetto` smoke confirmed that `permissions-audit.js` now finds local workflow files correctly even when `--cwd` points somewhere else.

### What didn't work

- N/A in this step.

### What I learned

- The runtime split affected more than `@actions/io`; `process.cwd()` and default `@actions/exec` had the same mismatch.
- Fixing the shared default first is cleaner than teaching every script to work around the split.

### What was tricky to build

- The subtle part was preserving an explicit override path. The right compromise was: workspace-first by default, but keep `exec(..., { cwd })` as the escape hatch for subprocesses.

### What warrants a second pair of eyes

- Whether `process.workingDirectory` should keep exposing the execution root or whether a separate raw CLI cwd field should exist later.

### What should be done in the future

- Decide whether raw `--cwd` should be surfaced separately to JS.
- Update any remaining docs that still describe the old split.

### Code review instructions

- Start with [factory.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go), [globals.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go), [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/io/module.go), and [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go).
- Then review the tests in the runtime, IO, and exec packages.
- Finally, inspect [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) to see how the new default simplifies script code.

### Technical details

- Validation commands:
  - `GOWORK=off go test ./pkg/runtime ./pkg/modules/io ./pkg/modules/exec ./integration`
  - `GOWORK=off go run ./cmd/goja-gha run --script ./examples/trivial.js --cwd /tmp --workspace /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions --json-result`
- Full validation:
  - `GOWORK=off go test ./...`
  - `GOWORK=off go run ./cmd/goja-gha help javascript-api >/dev/null`
  - `GOWORK=off go run ./cmd/goja-gha help user-guide >/dev/null`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/permissions-audit.js --cwd /tmp --workspace /tmp/geppetto --event-path ./testdata/events/workflow_dispatch.json --json-result | jq '{workspace, localWorkflowFiles, workflowCount, allowedActions: .permissions.allowed_actions}'`
- Real smoke result:

```json
{
  "workspace": "/tmp/geppetto",
  "localWorkflowFiles": [
    "codeql-analysis.yml",
    "dependency-scanning.yml",
    "lint.yml",
    "push.yml",
    "release.yml",
    "secret-scanning.yml",
    "tag-release-notes.yml"
  ],
  "workflowCount": 10,
  "allowedActions": "all"
}
```
