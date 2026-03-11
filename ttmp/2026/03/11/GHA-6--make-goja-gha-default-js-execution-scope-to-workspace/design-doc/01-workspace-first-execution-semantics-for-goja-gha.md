---
Title: Workspace-first execution semantics for goja-gha
Ticket: GHA-6
Status: active
Topics:
    - github-actions
    - goja
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/permissions-audit.js
      Note: Simplified after workspace-first relative IO became the default
    - Path: pkg/modules/exec/module.go
      Note: Changes default subprocess cwd to workspace-first execution root
    - Path: pkg/modules/io/module.go
      Note: Changes relative IO to resolve against workspace-first execution root
    - Path: pkg/runtime/factory.go
      Note: Introduces the shared ExecutionRoot helper and defines workspace-first semantics
    - Path: pkg/runtime/globals.go
      Note: Uses the execution root for process.cwd and process.workingDirectory
ExternalSources: []
Summary: Small implementation design describing the change from split cwd/workspace semantics to a workspace-first execution root for JavaScript.
LastUpdated: 2026-03-11T12:35:05.316173484-04:00
WhatFor: Explain why workspace-first semantics are more intuitive for script authors and describe the implementation changes needed across runtime globals, IO, and exec.
WhenToUse: Use when reviewing the execution-root change or extending path/cwd semantics later.
---


# Workspace-first execution semantics for goja-gha

## Executive summary

`goja-gha` previously exposed two related but different path concepts:

- `workspace`, representing the GitHub checkout root;
- `WorkingDirectory`, which powered `process.cwd()`, `@actions/io`, and default `@actions/exec` behavior.

That split made scripts harder to reason about. A script that conceptually meant “operate on the checked-out repo” could still resolve relative paths against some other `--cwd` value unless it manually stitched paths back to `process.env.GITHUB_WORKSPACE`.

This change introduces a single default JavaScript execution root:

```text
execution root = workspace if present, otherwise working directory, otherwise "."
```

That execution root is now used by:

- `process.cwd()`
- `process.workingDirectory`
- `@actions/io` relative path resolution
- default `@actions/exec` command cwd

Explicit `exec(..., { cwd })` still overrides the default for subprocesses.

## Problem statement

The old behavior forced script authors to think about two separate defaults:

```text
process.env.GITHUB_WORKSPACE  -> repository root
process.cwd()                -> working directory flag
io.readdir("...")            -> working directory flag
exec("cmd")                  -> working directory flag
```

That is not how most GitHub Actions scripts are mentally modeled. In practice, the normal questions are:

- “What files are in `.github/workflows`?”
- “What does `git rev-parse --show-toplevel` say for this checkout?”
- “What repo tree am I inspecting?”

Those are workspace-relative questions.

The mismatch already leaked into the examples. [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) had to compensate manually by building its workflow directory from `process.env.GITHUB_WORKSPACE || process.cwd()` instead of relying on relative IO.

## Design decision

Adopt workspace-first semantics for JavaScript execution.

### Rule

```text
ExecutionRoot(settings):
  if settings.Workspace is non-empty:
    return settings.Workspace
  if settings.WorkingDirectory is non-empty:
    return settings.WorkingDirectory
  return "."
```

### Why this is the right default

- It matches how GitHub Actions users think about the checked-out repository.
- It removes an easy source of accidental mis-scoping in scripts.
- It lets example scripts use relative paths naturally.
- It preserves an explicit override path for subprocesses through `exec(..., { cwd })`.

## Implementation

### Runtime settings

Add `Settings.ExecutionRoot()` in [factory.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go).

This centralizes the precedence rule so IO, exec, and process globals all use the same decision.

### Process globals

Update [globals.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go) so:

- `process.cwd()` returns `ExecutionRoot()`
- `process.workingDirectory` also reflects `ExecutionRoot()`

This keeps the JavaScript-facing runtime coherent.

### IO module

Update [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/io/module.go) so relative file operations resolve against `ExecutionRoot()` rather than `WorkingDirectory`.

### Exec module

Update [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go) so the default subprocess directory is `ExecutionRoot()`.

Important:

- keep `exec(..., { cwd })` as an explicit override;
- do not remove `WorkingDirectory` from settings yet, because it is still a useful lower-precedence fallback and remains visible at the CLI layer.

### Example simplification

After the runtime change, [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) can go back to:

```javascript
io.readdir(".github/workflows")
```

instead of rebuilding the workspace path manually.

## Tests

Add or update tests to prove:

1. `process.cwd()` prefers workspace when both workspace and working directory are set.
2. `@actions/io` relative paths resolve against workspace by default.
3. `@actions/exec` default command execution happens in workspace.
4. Existing behavior still works when workspace is absent.

These checks live in:

- [script_runner_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner_test.go)
- [module_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/io/module_test.go)
- [module_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module_test.go)

## Validation

Validated with:

```bash
GOWORK=off go test ./pkg/runtime ./pkg/modules/io ./pkg/modules/exec ./integration
GOWORK=off go run ./cmd/goja-gha run --script ./examples/trivial.js --cwd /tmp --workspace /path/to/repo --json-result
```

The `trivial.js` run now shows `cwd == workspace` when both are provided.

## Follow-up notes

- This change should land before the next wave of security scripts, because those scripts should naturally treat the repo workspace as their root.
- If we later want to expose the raw CLI `--cwd` separately to JavaScript, that should be a deliberate new field, not the default `process.cwd()` behavior.
