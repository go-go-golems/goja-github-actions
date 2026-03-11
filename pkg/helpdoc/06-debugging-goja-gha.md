---
Title: "Debugging goja-gha"
Slug: "debugging-goja-gha"
Short: "Use the built-in logging flags, request tracing, and JavaScript console output to debug runtime and GitHub API failures."
Topics:
- goja
- github-actions
- javascript
- glazed
Commands:
- run
Flags:
- log-level
- log-format
- with-caller
- json-result
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page explains the practical debugging workflow for `goja-gha`. It is the page to read when a script fails with a message like `GitHub API request failed` and you need to work out whether the problem is your CLI inputs, runtime setup, JavaScript code, or the GitHub HTTP call itself.

The most important fact is that `goja-gha` already has root logging support through Glazed. You do not need a special debug binary. You debug by turning up the existing log level and then reading the boundary logs emitted by the CLI, runtime, and GitHub API client.

## Quick Start

Start with text logs at debug level:

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/trivial.js
```

That gives you readable zerolog output on stderr. For most local debugging sessions, this is the right default.

## What The Logs Tell You

At `--log-level debug`, the current implementation emits logs at several useful boundaries:

- command startup and resolved run settings,
- runtime creation and entrypoint resolution,
- exported function execution,
- promise waiting and rejection,
- GitHub API request start and response status,
- Octokit client creation with token presence and base URL.

The logs are designed to help with questions like:

- was a token present at all?
- did the runtime see a workspace or repository?
- what request URL was actually called?
- did the script get as far as running its exported function?

Secrets are intentionally not logged. You should expect `github_token_present=true`, not the raw token value.

## Recommended Command Patterns

### 1. Basic runtime debugging

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/trivial.js
```

Use this first to confirm that:

- the command starts,
- the script path resolves,
- the runtime is created,
- the exported function actually runs.

### 2. GitHub API debugging

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./examples/permissions-audit.js \
  --cwd /path/to/local/repo \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result
```

When the script calls `@actions/github`, look for logs with:

- `component=actions-github`
- `component=githubapi`

Those logs tell you:

- whether the token was present,
- which base URL was used,
- which route was requested,
- which final request URL was called,
- what HTTP status came back.

### 3. Include caller info when narrowing down log origins

```bash
go run ./cmd/goja-gha --log-level debug --log-format text --with-caller run \
  --script ./examples/trivial.js
```

Use `--with-caller` when you need to map a log line back to the exact Go source file.

## Debugging A 401

If you see:

```text
Error:
GitHub API request failed

Route: GET /repos/{owner}/{repo}/actions/permissions
Status: 401 Unauthorized
Message: Bad credentials
```

debug in this order:

1. Confirm the root command is running with `--log-level debug`.
2. Check the `Resolved run settings` line for:
   - `github_token_present`
   - `workspace`
   - `repository`
3. Check the `Creating Octokit client` line for:
   - `token_present`
   - `token_source`
   - `base_url`
4. Check the `Sending GitHub API request` and `Received GitHub API response` lines for:
   - `route`
   - `request_url`
   - `status`

If `github_token_present=false` or `token_present=false`, the problem is upstream of the HTTP client. If `token_present=true` but the API still returns `401`, the token itself is wrong, expired, scoped incorrectly, or not approved for the target repository or org.

If the status is `403` with a message like `Resource not accessible by personal access token`, that usually means the token is valid but does not have the required fine-grained permission for that endpoint. For `permissions-audit.js`, the repo permissions endpoints are stricter than the plain workflow-list endpoint, so a token can succeed on one call and still fail on the audit endpoints.

If the status is `409 Conflict` for `GET /repos/{owner}/{repo}/actions/permissions/selected-actions`, the problem is usually not auth. It means the endpoint is not applicable because the repository policy is not `allowed_actions == "selected"`. The shipped `permissions-audit.js` example now avoids that call unless the initial permissions document says the repo is using the `selected` policy.

## Reading The New CLI Error Format

The CLI now tries to surface operational errors as a short, human-readable block instead of dumping the wrapped Goja native stack by default.

Examples:

```text
Error:
GitHub API request failed

Route: GET /repos/{owner}/{repo}/actions/permissions/selected-actions
Status: 409 Conflict
Hint: this endpoint only applies when the repository allowed_actions policy is set to "selected"; fetch /actions/permissions first and call selected-actions only when allowed_actions == "selected"
```

```text
Error:
JavaScript execution failed

Message: runner output file path is empty
Location: <native>:-
```

Why this matters:

- the main stderr path now prioritizes the actionable message,
- debug logs still preserve the deeper wrapped context,
- users do not need to parse `GoError:` and `at ... (native)` unless they are actively debugging internals.

## Using JavaScript `console.log`

`goja-gha` also exposes `console` to your JavaScript runtime, and local runs surface that output directly. This is useful when you need script-level context that Go-side logs do not know about.

Example script:

```javascript
module.exports = async function () {
  console.log("console-log: script starting");
  console.warn("console-warn: warning path");
  console.error("console-error: error path");
  return { ok: true };
};
```

You can run the validated ticket copy here:

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run \
  --script ./ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/scripts/console-debug.js \
  --json-result
```

Why this matters:

- Go logs tell you about runtime and HTTP boundaries,
- `console.*` tells you what your script thought it was doing,
- together they give you a complete local debugging loop.

## Example Output Shape

With debug logging enabled, a successful trivial run now looks roughly like this:

```text
DBG Logger initialized
DBG Resolved run settings component=run script=./examples/trivial.js ...
DBG Creating runtime with modules component=runtime module_count=4 ...
DBG Resolving entrypoint module component=runtime entrypoint=trivial.js
DBG Executing exported function component=runtime entrypoint=trivial.js
```

For GitHub API scripts, you should also see lines like:

```text
DBG Creating Octokit client component=actions-github token_present=true base_url=https://api.github.com
DBG Sending GitHub API request component=githubapi method=GET route="GET /repos/..."
DBG Received GitHub API response component=githubapi status=401 request_url=https://api.github.com/...
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| no debug logs appear | log level is still `info` | run with `--log-level debug` |
| logs are hard to read | JSON/text choice is wrong for the session | use `--log-format text` locally |
| you need the exact Go source line | caller info is off | add `--with-caller` |
| you still only see the final 401 | the script is not running with debug logs enabled | confirm the root flags are placed before `run` |
| you now see `403 Resource not accessible by personal access token` | the token is valid but lacks enough repository permission for the endpoint | inspect the fine-grained PAT repo permissions and org approval state |
| you now see `409 Conflict` for `selected-actions` | the endpoint is not applicable for the repo policy mode | inspect `permissions.allowed_actions`; for the shipped audit example this should now be skipped automatically |
| the audit result includes `runnerOutput.written=false` or `stepSummary.written=false` | you are running locally without runner output/summary files | provide `GITHUB_OUTPUT` or `GITHUB_STEP_SUMMARY` only if you need those side effects |
| you need script-level context | only Go-side logs are enabled | add `console.log(...)` or use the ticket console-debug script |

## See Also

- `goja-gha help cli-middleware-and-env-resolution` for parsing and precedence details
- `goja-gha help developer-guide` for architecture and package ownership
- `goja-gha help javascript-api` for the JavaScript surface, including `@actions/github`
