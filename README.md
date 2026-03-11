# goja-github-actions

`goja-github-actions` is the home of `goja-gha`, a Go CLI for running GitHub Actions-oriented JavaScript on top of Goja.

The current implementation provides:

- a Glazed/Cobra CLI with `run` and `doctor`,
- decoded runner settings plus a shared `github-actions` section,
- a Goja runtime with `process`, `console`, and script-relative `require(...)`,
- `@actions/core`, `@actions/github`, `@actions/io`, and `@actions/exec`,
- example scripts for outputs, permissions auditing, and workflow inspection,
- a local composite action wrapper in [`action.yml`](./action.yml),
- CI coverage for unit tests, CLI integration tests, and local-action smoke runs.

## Current Status

`goja-gha run` executes CommonJS entrypoints and supports both synchronous and `async` exports. The first shipped module surface is intentionally narrow and centered on the initial permissions-audit use case:

- `@actions/core`: inputs, outputs, env/path mutation, logging, `setFailed`, and step summaries,
- `@actions/github`: `github.context`, generic `request/paginate`, and the first `rest.actions.*` helpers,
- `@actions/io`: common file and path helpers for local workflow inspection,
- `@actions/exec`: promise-based command execution with stdout/stderr capture.

## Development

Show the root help:

```bash
go run ./cmd/goja-gha --help
```

Inspect resolved settings:

```bash
go run ./cmd/goja-gha doctor --script ./examples/permissions-audit.js --output json
```

Run the trivial smoke example:

```bash
go run ./cmd/goja-gha run --script ./examples/trivial.js --json-result
```

Run the set-output example with local runner files:

```bash
tmpdir=$(mktemp -d)
INPUT_NAME=Manuel \
GITHUB_OUTPUT="$tmpdir/output.txt" \
GITHUB_STEP_SUMMARY="$tmpdir/summary.md" \
go run ./cmd/goja-gha run --script ./examples/set-output.js --json-result
```

Run the permissions-audit example against a fake or real GitHub API:

```bash
GITHUB_TOKEN=... \
GITHUB_REPOSITORY=owner/repo \
GITHUB_WORKSPACE="$PWD" \
go run ./cmd/goja-gha run \
  --script ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

When you run `permissions-audit.js` locally without `GITHUB_OUTPUT` or `GITHUB_STEP_SUMMARY`, the script now returns JSON and records best-effort runner-file status in `runnerOutput` and `stepSummary` instead of failing the whole run.

Debug the permissions-audit example with structured logs:

```bash
GITHUB_TOKEN=... \
GITHUB_REPOSITORY=owner/repo \
GITHUB_WORKSPACE="$PWD" \
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

For `examples/permissions-audit.js`, a fine-grained PAT usually needs these repository permissions:

- `Actions: Read`
- `Administration: Read`

Why: the script does not just list workflows. It also calls the repository Actions permissions endpoints, which are stricter than the plain workflow-list endpoint. If your logs show `403 Resource not accessible by personal access token`, the token is valid but under-scoped for those admin-style endpoints.

If the repository does not use `allowed_actions = "selected"`, the example now skips the `selected-actions` endpoint and returns:

- `selectedActions: null`
- `selectedActionsStatus: "skipped-not-selected-policy"`
- `selectedActionsReason: ...`

## Local Action Wrapper

The repo root also exposes a local composite action:

```yaml
- uses: ./
  with:
    script: ./examples/trivial.js
    cwd: .
    json-result: "true"
```

This wrapper builds `goja-gha` from source on the runner and then executes the requested script.

## Roadmap

The design packet and detailed ticket backlog live in:

- `ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/design-doc/01-goja-github-actions-design-and-implementation-guide.md`
- `ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/tasks.md`
