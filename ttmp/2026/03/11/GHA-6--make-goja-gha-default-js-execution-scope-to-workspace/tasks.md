# Tasks

## Implementation

- [x] Create `GHA-6` ticket workspace and docs.
- [x] Identify every runtime surface that currently depends on `WorkingDirectory`.
- [x] Add a shared execution-root helper that prefers workspace.
- [x] Update `process.cwd()` to use the execution root.
- [x] Update `@actions/io` relative path resolution to use the execution root.
- [x] Update default `@actions/exec` cwd to use the execution root.
- [x] Simplify `permissions-audit.js` to rely on relative IO again.
- [x] Update public docs to explain workspace-first semantics.

## Validation

- [x] Add runtime tests for workspace-first `process.cwd()`.
- [x] Add IO module tests for workspace-first relative paths.
- [x] Add exec module tests for workspace-first command execution.
- [x] Run targeted Go tests for runtime, IO, exec, and integration.
- [x] Run a real `/tmp/geppetto` smoke after the semantic change.

## Follow-up

- [ ] Decide whether to expose raw CLI `--cwd` separately to JavaScript in the future.
