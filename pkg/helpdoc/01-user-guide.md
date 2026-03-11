---
Title: "User Guide"
Slug: "user-guide"
Short: "Learn what goja-gha is, how to run scripts, how settings resolve, and how to use the tool locally or in GitHub Actions."
Topics:
- goja
- github-actions
- javascript
Commands:
- run
- doctor
Flags:
- script
- cwd
- event-path
- github-token
- json-result
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide explains how to use `goja-gha` as a person running scripts rather than modifying the runtime itself. It covers what the tool does, how configuration flows into a script, how to run the shipped examples, and how to think about the differences between a local run and a real GitHub Actions runner.

The key idea is simple: `goja-gha` executes JavaScript with a Goja runtime and exposes a GitHub Actions-flavored API through `require("@actions/core")`, `require("@actions/github")`, `require("@actions/io")`, and `require("@actions/exec")`. That means you write JavaScript, but the runtime, packaging, and distribution model stay in Go.

## What The Tool Is

This section explains the mental model you should keep in your head when using the CLI. `goja-gha` is not a Node.js compatibility layer and it is not a general npm package runner. It is a focused JavaScript execution environment for GitHub automation.

In practice, each run looks like this:

```text
+--------------------+
| goja-gha run       |
+---------+----------+
          |
          v
+--------------------+
| Decode settings    |
| flags/config/env   |
+---------+----------+
          |
          v
+--------------------+
| Build runtime      |
| process + modules  |
+---------+----------+
          |
          v
+--------------------+
| require(entry.js)  |
| run export/main    |
+---------+----------+
          |
          v
+--------------------+
| emit JSON result   |
| and runner files   |
+--------------------+
```

Why this matters: if something goes wrong, you can usually place the failure in one of four buckets:

- settings did not resolve the way you expected,
- the runtime did not expose the object or module you expected,
- the script logic itself failed,
- a GitHub-oriented side effect such as outputs or summaries wrote to the wrong place.

## Quick Start

This section shows the smallest useful commands. Use these before reading the rest of the guide so you have a working baseline.

Run the smallest smoke example:

```bash
go run ./cmd/goja-gha run --script ./examples/trivial.js --json-result
```

If stdout is an interactive terminal, `goja-gha run` also prints a returned script value without `--json-result`. Keep `--json-result` for pipes, tests, and any automation that needs guaranteed machine-readable output.

Inspect what the CLI resolved before running anything important:

```bash
go run ./cmd/goja-gha doctor --script ./examples/trivial.js --output json
```

Run the set-output example with local runner files:

```bash
tmpdir=$(mktemp -d)
INPUT_NAME=Manuel \
GITHUB_OUTPUT="$tmpdir/output.txt" \
GITHUB_STEP_SUMMARY="$tmpdir/summary.md" \
go run ./cmd/goja-gha run --script ./examples/set-output.js --json-result
```

Use the permissions-audit example with a local event payload:

```bash
GITHUB_TOKEN=... \
GITHUB_REPOSITORY=owner/repo \
GITHUB_WORKSPACE="$PWD" \
go run ./cmd/goja-gha run \
  --script ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

For a plain local run, you do not need to provide `GITHUB_OUTPUT` or `GITHUB_STEP_SUMMARY` just to get a result anymore. The example now keeps going and returns a baseline audit object with:

- `scriptId`
- `summary`
- `findings`
- best-effort runner-file status under `runnerOutput` and `stepSummary`

If you want immediate debugging detail, use the same command with root logging enabled:

```bash
GITHUB_TOKEN=... \
GITHUB_REPOSITORY=owner/repo \
GITHUB_WORKSPACE="$PWD" \
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

## How Settings Resolve

This section explains the most important behavior for real-world usage: where values come from. If you do not understand this, you will misread both `doctor` output and script behavior.

`goja-gha` resolves settings in this order:

1. explicit Cobra flags,
2. positional/argument sources,
3. config files,
4. mapped runner environment variables,
5. app defaults,
6. field defaults.

That precedence is implemented in `pkg/cli/middleware.go`, and the runner-env mappings live in `pkg/cli/defaults.go`.

The most important mapped environment variables are:

| Environment variable | Resolved field |
|---|---|
| `GITHUB_EVENT_PATH` | `event-path` |
| `GITHUB_ACTION_PATH` | `action-path` |
| `GITHUB_ENV` | `runner-env-file` |
| `GITHUB_OUTPUT` | `runner-output-file` |
| `GITHUB_PATH` | `runner-path-file` |
| `GITHUB_STEP_SUMMARY` | `runner-summary-file` |
| `GITHUB_WORKSPACE` | `github-actions.workspace` |
| `GITHUB_TOKEN` / `GH_TOKEN` | `github-actions.github-token` |

Why this matters in practice:

- on a real runner, many values arrive automatically through env vars,
- in a local smoke run, you often need to provide runner-file paths yourself,
- `doctor` is the fastest way to confirm which source won.

## What Your Script Receives

This section covers the runtime surface visible to JavaScript. You do not need to know every internal detail, but you do need to know what is actually present.

The runtime gives your script:

- CommonJS `require(...)`,
- `console`,
- `process.env`,
- `process.cwd()`,
- `process.stdout.write(...)`,
- `process.stderr.write(...)`,
- `process.exitCode`,
- `@actions/core`,
- `@actions/github`,
- `@actions/io`,
- `@actions/exec`.

The entrypoint contract is:

- `module.exports = function () { ... }`
- `module.exports = async function () { ... }`
- `module.exports = { main() { ... } }`
- `module.exports = { default() { ... } }`

The runtime awaits returned Promises. That means `async` exports are first-class rather than a side case.

Path behavior is workspace-first by default:

- `process.cwd()` resolves to the workspace when one is set,
- relative `@actions/io` calls resolve against that same root,
- default `@actions/exec` subprocesses also start there,
- explicit `exec(..., { cwd })` still overrides the default for that command.

Pseudocode for the entrypoint behavior looks like this:

```text
load module
if exports is a function:
  call it
  if it returns a Promise:
    wait for the Promise
else if exports.main or exports.default is a function:
  call that function
  if it returns a Promise:
    wait for the Promise
else:
  return module.exports directly
```

## Running The Shipped Examples

This section explains what each example demonstrates and why you would pick it when learning the system.

### `examples/trivial.js`

Use this first. It proves that module loading, `process`, and JSON result emission all work.

### `examples/set-output.js`

Use this when you want to see the simplest runner-file interaction. It shows how `@actions/core.setOutput()` and the summary builder behave during a local run.

### `examples/core-primitives.js`

Use this when you want broad coverage of `@actions/core`. It exercises inputs, outputs, env export, path mutation, and summaries in one script.

### `examples/permissions-audit.js`

Use this when you want the first real GitHub-oriented application. It combines:

- `@actions/github` for REST calls,
- `github.context` for repo and event metadata,
- `@actions/io` for local workflow-directory inspection,
- `@actions/core` for outputs and summaries.
- `@goja-gha/ui` for the human-readable terminal report shown during normal runs.

Token requirements matter more for this example than for the simpler workflow-list examples. A fine-grained PAT that works for listing workflows can still fail here, because this script also calls repository Actions permissions endpoints. In practice, the token usually needs:

- `Actions: Read`
- `Administration: Read`

If the run fails with `403 Resource not accessible by personal access token`, the token is usually valid but under-scoped for one of those permissions endpoints.

The example also now follows the GitHub API contract for `selected-actions` more closely. It first fetches the repository Actions permissions document and only calls the `selected-actions` endpoint when `permissions.allowed_actions == "selected"`. For repos using other policy modes such as `all`, the result will contain:

- `selectedActions: null`
- `selectedActionsStatus: "skipped-not-selected-policy"`
- `selectedActionsReason: ...`

In a normal `run` without `--json-result`, the script now prints a human-oriented report. In a `--json-result` run, the UI report is suppressed and the returned object stays machine-readable.

The current baseline findings focus on repository-level policy settings first. At the moment that includes:

- unrestricted `allowed_actions`
- missing SHA pinning requirements
- non-read-only default workflow permissions
- Actions approving pull request reviews

### `examples/list-workflows.js`

Use this when you want to see `@actions/io` and `@actions/exec` together. It lists workflow files locally and then runs a helper command to inspect repository state.

## Using The Composite Action

This section covers the repo-root `action.yml` wrapper. Use it when you want GitHub Actions to run `goja-gha` from this repository without writing your own shell wrapper first.

Example:

```yaml
- uses: ./
  with:
    script: ./examples/trivial.js
    cwd: .
    json-result: "true"
```

What the wrapper does:

1. builds `goja-gha` from source on the runner,
2. constructs CLI arguments from the provided inputs,
3. executes `goja-gha run`.

Why this matters: it keeps the delivery path simple and honest. The runner is still executing the same CLI you use locally, which reduces drift between local debugging and CI behavior.

## How To Debug A Failed Run

This section explains the quickest debugging loop. The goal is not just to list commands, but to tell you what each one isolates.

Start with:

```bash
go run ./cmd/goja-gha doctor --script ./examples/permissions-audit.js --output json
```

That tells you:

- whether `--script` points to the right file,
- whether a token is present,
- whether runner-file paths resolved,
- whether `cwd` and workspace are what you think they are.

Then run the real script with debug logs:

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

Then narrow the failure:

- If the script never starts, inspect your `--script` path and module-relative imports.
- If GitHub calls fail, inspect `GITHUB_TOKEN`, `GITHUB_API_URL`, `github.context`, and the debug lines for `Creating Octokit client`, `Sending GitHub API request`, and `Received GitHub API response`.
- If outputs or summaries disappear, inspect `GITHUB_OUTPUT` and `GITHUB_STEP_SUMMARY`.
- If outputs or summaries are missing during a local run, inspect the returned `runnerOutput` and `stepSummary` objects before assuming the whole script failed.
- If async code behaves strangely, reduce the script to a single awaited operation and re-run with `--json-result`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `--script is required` | No entrypoint resolved from flags or config | Pass `--script` explicitly and confirm it in `doctor` output |
| GitHub API calls return `401 Bad credentials` | The token is missing, invalid, expired, or not the token you think the runtime used | Re-run with `--log-level debug --log-format text` and verify `github_token_present=true`, `token_present=true`, and the request status lines |
| GitHub API calls return `403 Resource not accessible by personal access token` | The token is valid but lacks enough repository permission for the endpoint | For `permissions-audit.js`, grant fine-grained PAT repo permissions `Actions: Read` and `Administration: Read` |
| The audit result shows `selectedActionsStatus = "skipped-not-selected-policy"` | The repository does not use the `selected` allowed-actions mode | This is expected; inspect `permissions.allowed_actions` and skip the `selected-actions` endpoint for that repo |
| `runnerOutput.written=false` | `GITHUB_OUTPUT` or `--runner-output-file` was not set for a local run | Provide a writable output file path if you need actual runner output side effects |
| `stepSummary.written=false` | `GITHUB_STEP_SUMMARY` or `--runner-summary-file` was not set | Provide a writable summary file path if you need a step summary file |
| A script sees the wrong workspace | `GITHUB_WORKSPACE` beat your expected value in precedence resolution | Run `doctor --output json` and inspect `workspace` |
| `@actions/exec` returns a rejection | The subprocess failed or was not found | Run the command manually first, then inspect `stderr` in the result |

## See Also

- `goja-gha help javascript-api`
- `goja-gha help debugging-goja-gha`
- `goja-gha help developer-guide`
- `goja-gha help design-and-internals`
