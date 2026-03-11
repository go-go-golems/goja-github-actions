---
Title: Investigation diary
Ticket: GHA-3
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/glazed/cmd/glaze/main.go
      Note: Compared against Glazed root logging setup
    - Path: cmd/goja-gha/cmds/root.go
      Note: Confirmed that logging infrastructure already exists
    - Path: pkg/githubapi/client.go
      Note: Inspected the current opaque 401 path
    - Path: pkg/helpdoc/06-debugging-goja-gha.md
      Note: Help page added during the first implementation slice
    - Path: pkg/runtime/script_runner.go
      Note: Inspected current runtime failure surfacing path
    - Path: ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/scripts/console-debug.js
      Note: Durable console validation script stored in the ticket scripts folder
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T10:10:18.464733963-04:00
WhatFor: ""
WhenToUse: ""
---



# Investigation diary

## Goal

Record the debugging/observability investigation for `GHA-3`, including what is already present in the codebase, what is missing, and how the follow-up implementation should be reviewed.

## Step 1: Close the paused Glazed ticket and open a debugging ticket

This step re-scoped the work. The earlier `GHA-2` ticket focused on moving settings resolution more fully into Glazed, but the immediate user need shifted to debugging a concrete 401 failure and improving observability. The right move was to close the Glazed refactor ticket for now and open a new ticket dedicated to logging, request tracing, and JS/runtime debugging.

While gathering evidence for the new ticket, I found that one key assumption had already changed from earlier discussion: `goja-gha` already wires Glazed's zerolog/logging support at the root command. The problem is not the absence of logging infrastructure. The problem is the lack of useful log emission from the subsystems that matter during failures.

### Prompt Context

**User prompt (verbatim):** "I don't have the repository set for this it seems :❯   go run ./cmd/goja-gha run \
    --script ./examples/permissions-audit.js \
    --cwd /path/to/local/geppetto \
    --event-path ./testdata/events/workflow_dispatch.json \
    --json-result        
Error: execute exported function: GoError: github api error: status 401: Bad credentials at github.com/go-go-golems/goja-github-actions/pkg/modules/github.(*Module).newActionsObject.func1 (native)
exit status 1

(at least not as a glazed setting, how can I best debug this? add zerolog log support like in glazed/cmd/glaze/main.go  and add more debugging, and also the javascript console.log.

Create a new ticket for this, now that we're skipping the glazed ticket (close it, even if not done)"

**Assistant interpretation:** Close the paused Glazed settings ticket, open a new ticket for practical debugging improvements, and anchor that work to the current logging and runtime boundaries rather than a larger parsing refactor.

**Inferred user intent:** Make `goja-gha` easier to debug in day-to-day use, especially for GitHub API failures and JavaScript script behavior.

**Commit (code):** N/A — ticketing and design setup only

### What I did
- Confirmed that `goja-gha` already wires Glazed logging in [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go).
- Inspected [`glazed/cmd/glaze/main.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/cmd/glaze/main.go) to compare the root logging pattern.
- Inspected the current GitHub request path in [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go).
- Inspected runtime execution flow in [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go).
- Closed `GHA-2` and created `GHA-3`.
- Added a design doc and diary for the new ticket.

### Why
- The immediate user problem is practical debugging, not architecture cleanup.
- Logging and request tracing are the shortest path to making 401-style failures diagnosable.
- Closing the paused ticket prevents split focus and makes the active priority explicit.

### What worked
- The code inspection quickly showed that zerolog support is already initialized at the root command.
- The ticket split is clean: `GHA-2` now records the paused design direction, and `GHA-3` captures the new active direction.

### What didn't work
- `docmgr doc add --ticket GHA-3 ...` failed once immediately after ticket creation with:

```text
Error: failed to find ticket directory: ticket not found: GHA-3
```

- Rerunning the same command after confirming the ticket list worked without further changes.

### What I learned
- The debugging ticket should not be framed as "add logging support", because that is already partly true.
- The real missing capabilities are:
  - subsystem-specific log emission,
  - GitHub request tracing,
  - better runtime failure context,
  - a documented `console.log` workflow for JS authors.

### What was tricky to build
- The main subtlety was scoping the new ticket correctly.
- It would have been easy to start implementing logging immediately, but the user explicitly asked to close the old ticket and create a new one first.
- It also required distinguishing between "logging infrastructure exists" and "useful logs exist", which are not the same thing.

### What warrants a second pair of eyes
- The follow-up implementation should be reviewed for secret handling. Any request logging near auth must avoid printing raw tokens.
- The console/debugging plan should be reviewed alongside `go-go-goja` behavior so we do not accidentally document console semantics incorrectly.

### What should be done in the future
- Implement `GHA-3` in small slices:
  - root/help documentation for logging flags,
  - GitHub request tracing,
  - runtime boundary logs,
  - JS console debugging notes and tests.

### Code review instructions
- Start with [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go).
- Then inspect [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go) and [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go).
- Finally read the new design doc:
  - [`01-debugging-and-logging-design-for-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/design-doc/01-debugging-and-logging-design-for-goja-gha.md)

### Technical details

#### Quick summary table

| Concern | Current state | Ticket direction |
| --- | --- | --- |
| root zerolog setup | already present | document and reuse |
| request tracing | minimal | add structured HTTP logs |
| runtime execution logs | minimal | add boundary logs |
| JS debugging story | implicit | document `console.*` and validate it |
| token visibility | opaque | log presence/provenance, not secret values |

## Related

- [`01-debugging-and-logging-design-for-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/design-doc/01-debugging-and-logging-design-for-goja-gha.md)
- [`GHA-2 index.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-2--move-goja-gha-settings-resolution-fully-into-glazed-sources/index.md)

## Step 2: Implement the first debugging slice

This step turned the design into actual behavior. The goal was not to finish every observability improvement in one pass. The goal was to make `goja-gha` materially easier to debug right away: turn on debug logging, see resolved run settings, see runtime milestones, see GitHub request traces, and confirm that plain JavaScript `console.*` output is a viable part of the workflow.

The resulting shape is intentionally pragmatic. It reuses the existing Glazed logging flags instead of inventing new ones, adds logs only at subsystem boundaries that answer real debugging questions, and avoids printing raw credentials while still telling the user whether a token is present.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Start implementing the active debugging ticket, focusing on concrete logging and observability improvements that help debug GitHub API failures and runtime behavior.

**Inferred user intent:** Be able to run `goja-gha` with one clear debug command and quickly understand whether a failure comes from settings, runtime startup, JavaScript behavior, or the GitHub API call itself.

**Commit (code):** N/A — code changes not yet committed

### What I did
- Added secret-safe run-setting logs in [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go).
- Added runtime milestone logs in [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go):
  - runtime creation
  - entrypoint resolution
  - exported function execution
  - promise wait / fulfillment / rejection / timeout
- Added GitHub request/response tracing in [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go).
- Added Octokit client creation logs in [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go), including token presence, token source, base URL, and resolved repository.
- Added a user-facing help page:
  - [`06-debugging-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/06-debugging-goja-gha.md)
- Added a durable console validation script in the ticket:
  - [`console-debug.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/scripts/console-debug.js)
- Validated the new workflow with:
  - `GOWORK=off go run ./cmd/goja-gha --log-level debug --log-format text run --script ./examples/trivial.js`
  - `GOWORK=off go run ./cmd/goja-gha help debugging-goja-gha`
  - `GOWORK=off go run ./cmd/goja-gha --log-level debug --log-format text run --script ./ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/scripts/console-debug.js --json-result`
  - `GOWORK=off go test ./...`
  - `GOWORK=off go build ./...`

### Why
- The user needs a practical debugging path immediately, not just another design memo.
- Boundary logs provide the highest signal-to-noise ratio for failures like 401s.
- A help page plus a known-good `console-debug.js` script makes the workflow discoverable for future users.

### What worked
- The new logs show up correctly with the existing root flags:
  - `--log-level debug`
  - `--log-format text`
- The runtime logs make it obvious whether the script reached module resolution and exported function execution.
- The console validation run proved that `console.log`, `console.warn`, and `console.error` are already visible during local runs.
- The test suite stayed green after the logging changes.

### What didn't work
- An early validation attempt with `examples/set-output.js` failed because no runner output file path was provided:

```text
Error: execute exported function: GoError: runner output file path is empty at github.com/go-go-golems/goja-github-actions/pkg/modules/core.(*Module).setOutput-fm (native)
```

- That failure was expected once the example was inspected more closely. It was not a regression in the new logging code. I switched to a dedicated console-only script for validating the debugging workflow.

### What I learned
- The existing root logging integration was enough; no new flag plumbing was needed.
- `console.*` already works as a viable local debugging tool, so the missing part was documentation and a repeatable script, not runtime support.
- The most useful single debug command today is:

```bash
go run ./cmd/goja-gha --log-level debug --log-format text run --script ./examples/trivial.js
```

### What was tricky to build
- The main sharp edge was adding useful logs without leaking credentials.
- Another subtle point was deciding where to log token information: logging the token value would be unsafe, but logging only `token_present` and `token_source` still gives the user enough signal to debug most auth failures.
- The other tricky part was avoiding "chatty logs". The implementation works because the logs sit at meaningful boundaries instead of tracing every internal function call.

### What warrants a second pair of eyes
- The `component` naming and field naming should be reviewed for long-term consistency before more logs are added.
- The GitHub client logging should be reviewed once real 401 runs are exercised, to confirm the current fields are sufficient.
- If richer error text is added later, it should be reviewed alongside the logging so the CLI does not become redundant or noisy.

### What should be done in the future
- Improve direct error context around 401s and other GitHub API failures.
- Add regression tests that assert useful log output or helper behavior without snapshotting the entire logger stream.
- Add a dedicated auth-failure playbook or example for GitHub API troubleshooting.

### Code review instructions
- Start in [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go) to see the new resolved-settings log.
- Then inspect [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go) for runtime milestone logging.
- Then inspect [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go) and [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go) for token/base URL/request tracing.
- Finally read the user-facing workflow doc:
  - [`06-debugging-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/06-debugging-goja-gha.md)

### Technical details

#### Validated console output sample

```text
DBG Resolved run settings component=run ...
DBG Creating runtime with modules component=runtime ...
DBG Resolving entrypoint module component=runtime entrypoint=console-debug.js
DBG Executing exported function component=runtime entrypoint=console-debug.js
2026/03/11 10:26:16 console-log: script starting
2026/03/11 10:26:16 console-warn: warning path
2026/03/11 10:26:16 console-error: error path
DBG Awaiting promise result component=runtime
DBG Promise fulfilled component=runtime
```

## Step 3: Fix the missing-token fallback bug exposed by the new logs

Once the new debug logs were in place, they immediately exposed a real bug in the GitHub module. Running `permissions-audit.js` with a sourced `.envrc` showed `token_source=call-argument` even though the script called `github.getOctokit()` without passing a token. That was the clue: an omitted JavaScript argument was being treated as if it were a real string, so the runtime was not actually falling back to the configured token.

After fixing that bug and tightening the module test, the same run stopped failing with `401 Bad credentials` and instead returned the real API result: `403 Resource not accessible by personal access token`. That is a much better failure mode because it tells the user the token is valid but under-scoped for the permissions endpoint they are calling.

### Prompt Context

**User prompt (verbatim):** "I still get that : ❯   go run ./cmd/goja-gha run \
    --script ./examples/permissions-audit.js \
    --cwd /path/to/local/geppetto \
    --event-path ./testdata/events/workflow_dispatch.json \
    --json-result
Error: execute exported function: GoError: github api error: status 401: Bad credentials at github.com/go-go-golems/goja-github-actions/pkg/modules/github.(*Module).newActionsObject.func1 (native)
exit status 1

but still don't get more debug info"

**Assistant interpretation:** Reproduce the failing command with the new debug logs, identify why the logs are still not showing up for the user's command, and fix any runtime bug uncovered by the new instrumentation.

**Inferred user intent:** Turn the generic auth failure into a concrete diagnosis that distinguishes missing/incorrect token handling from actual GitHub permission problems.

**Commit (code):** N/A — code changes not yet committed

### What I did
- Reproduced the command with the correct root flag placement:
  - `--log-level debug --log-format text` before `run`
- Observed this key debug line before the fix:

```text
DBG Creating Octokit client ... token_present=true token_source=call-argument
```

- Confirmed from [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) that the script calls `github.getOctokit()` with no token argument.
- Fixed [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go) so undefined/null JS args do not masquerade as a real token.
- Tightened [`pkg/modules/github/module_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module_test.go) to assert that API requests actually use the runtime token.
- Re-ran the real command with the sourced `.envrc`.

### Why
- The new logs were only useful if they led to an actual fix when they exposed a bug.
- A 401 from `Bearer undefined` is a runtime defect. A 403 from a valid but under-scoped PAT is a user/environment issue. Those need to be distinguished cleanly.

### What worked
- The debug logs made the bad token-source behavior immediately visible.
- The module fix changed the observed auth failure from `401 Bad credentials` to:

```text
403 Resource not accessible by personal access token
```

- The updated module test now guards the intended fallback behavior.
- The full Go test suite still passed after the fix.

### What didn't work
- Before the fix, the missing-argument path in `getOctokit()` behaved incorrectly because the JS argument handling treated an omitted value as a real string.

### What I learned
- The new logging was already paying for itself in the same ticket: it found a real bug that was invisible before.
- The user's token was not "bad credentials" after all. The underlying token appears valid, but the permissions endpoint requires stronger access than the token currently has.

### What was tricky to build
- The tricky part was that the original symptom strongly suggested a token or org-approval problem, but the logs showed a code bug first. It would have been easy to stop at the GitHub UI explanation and miss the runtime defect.

### What warrants a second pair of eyes
- The `getOctokit()` argument handling should be reviewed for other JS edge cases like explicit empty strings and `null`.
- The help page language around 401 vs 403 should be reviewed once more endpoints are exercised.

### What should be done in the future
- Add a debugging example or playbook specifically for PAT scope failures on admin/permissions endpoints.
- Consider improving the surfaced CLI error text so common 403 scope failures are easier to interpret without turning on debug logs.

### Code review instructions
- Review [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go) first.
- Then review [`pkg/modules/github/module_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module_test.go).
- Finally check the updated troubleshooting guidance in:
  - [`06-debugging-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/06-debugging-goja-gha.md)

### Technical details

#### Before fix

```text
DBG Creating Octokit client ... token_present=true token_source=call-argument
...
status=401
```

#### After fix

```text
DBG Creating Octokit client ... token_present=true token_source=runtime-settings
...
status=403
Error: ... Resource not accessible by personal access token
```

## Step 4: Format CLI errors and make the audit example degrade gracefully

This step closed the loop on the real local user workflow. Better logs and better API hints were already in place, but the command-line experience was still rough in two ways: runtime failures surfaced as wrapped Goja/native strings, and once the `selected-actions` issue was understood, the local `permissions-audit.js` path still tripped over missing runner output files. The fix had to address both the CLI boundary and the shipped example itself.

The implementation ended up in two layers. First, the `run` command now owns its error-return path closely enough to format wrapped Goja and GitHub API failures into cleaner stderr blocks. Second, `permissions-audit.js` now behaves more like a robust operator tool: it only calls `selected-actions` when the repo policy makes that endpoint applicable, and it reports missing local runner files as structured status instead of aborting.

### Prompt Context

**User prompt (verbatim):** "do the formatCLIError, and then do then do the skip?"

**Assistant interpretation:** Implement a CLI-side formatter for the current wrapped runtime/API errors, then update the permissions-audit example so it skips the non-applicable `selected-actions` call.

**Inferred user intent:** Turn the current debugging work into a smoother day-to-day user experience by making common failures readable and by removing an avoidable example-script failure mode.

**Commit (code):** N/A — code changes not yet committed

### What I did
- Added [`cmd/goja-gha/cmds/errors.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/errors.go) with:
  - `FormatCLIError`
  - structured formatting for `githubapi.APIError`
  - formatting for `goja.Exception`
  - cleanup of wrapped `GoError: ... at ... (native)` strings
- Added formatter regression tests in [`cmd/goja-gha/cmds/errors_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/errors_test.go).
- Updated [`cmd/goja-gha/main.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/main.go) to print formatted stderr blocks.
- Reworked [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go) so the `run` subcommand uses the Glazed parser directly but returns errors normally instead of going through Glazed's internal `cobra.CheckErr()` path.
- Updated [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) to:
  - fetch repository permissions first,
  - skip `getAllowedActionsRepository(...)` unless `allowed_actions == "selected"`,
  - report `selectedActionsStatus` and `selectedActionsReason`,
  - treat missing `GITHUB_OUTPUT` / `GITHUB_STEP_SUMMARY` as best-effort status in `runnerOutput` / `stepSummary` instead of throwing.
- Added integration coverage in [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) for the skipped `selected-actions` path.
- Updated user-facing docs in:
  - [`README.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/README.md)
  - [`pkg/helpdoc/01-user-guide.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/01-user-guide.md)
  - [`pkg/helpdoc/02-javascript-api.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/02-javascript-api.md)
  - [`pkg/helpdoc/06-debugging-goja-gha.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/06-debugging-goja-gha.md)

### Why
- The improved `APIError` text was not enough on its own because Glazed's generated command wrapper still printed raw errors with `cobra.CheckErr()`.
- The shipped audit example is part of the product surface. Once the user had a correct token and correct API understanding, it was not acceptable for the example to fail on a non-applicable endpoint or on missing local runner files.

### What worked
- `FormatCLIError` now turns runtime wrapper text such as:

```text
execute exported function: GoError: runner output file path is empty at ... (native)
```

into:

```text
Error:
JavaScript execution failed

Message: runner output file path is empty
Location: <native>:-
```

- Local `permissions-audit.js` now succeeds against `go-go-golems/geppetto` and returns:
  - `selectedActionsStatus = "skipped-not-selected-policy"`
  - `runnerOutput.written = false`
  - `stepSummary.written = false`
- The integration suite now asserts that the `selected-actions` endpoint is not called when `allowed_actions` is not `selected`.

### What didn't work
- The first formatter pass only changed `main.go`, but real `run` failures still printed the raw error. Inspecting the Glazed source in the local workspace showed why:

```go
err = runFunc(ctx, parsedValues)
cobra.CheckErr(err)
```

- That meant the formatter never got a chance to run for those failures. I had to replace the generated `run` wrapper with a local one that uses the same parser but returns errors instead of exiting inside Glazed.
- Once the `409` path was removed, the validation run exposed a second local failure:

```text
Error: execute exported function: GoError: runner output file path is empty at github.com/go-go-golems/goja-github-actions/pkg/modules/core.(*Module).setOutput-fm (native)
```

- That was not a regression in the formatter. It was the next real issue in the example script.

### What I learned
- CLI error quality depends on where the process exits. Formatting in `main.go` is useless if an inner library path already calls `os.Exit` through `cobra.CheckErr()`.
- The `permissions-audit.js` example was too strict in two separate ways:
  - it assumed `selected-actions` was always applicable,
  - it assumed a local run always wanted real runner output files.

### What was tricky to build
- The tricky part was identifying which error path was actually user-visible. The formatter tests passed quickly, but the real command output stayed unchanged until I traced the Glazed-generated wrapper in the sibling `glazed` workspace.
- Another subtle point was keeping the `run` command behavior aligned with Glazed. The local wrapper had to preserve:
  - the same parser config,
  - the same command settings section,
  - `--print-parsed-fields`,
  - `--print-yaml`,
  - `--print-schema`,
  - signal-aware context handling.

### What warrants a second pair of eyes
- [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go) should be reviewed to confirm the local `run` wrapper stays close enough to Glazed behavior over time.
- [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) should be reviewed as product behavior, not just test scaffolding, because its JSON contract changed.

### What should be done in the future
- Consider upstreaming a configurable error-handling hook to Glazed so local apps do not need a custom wrapper to control `cobra.CheckErr()` behavior.
- Consider adding a small helper in `@actions/core` or the runtime that lets scripts ask whether runner output/summary files are configured before attempting side effects.

### Code review instructions
- Start with [`cmd/goja-gha/cmds/errors.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/errors.go) and [`cmd/goja-gha/main.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/main.go).
- Then review the `run` command wiring in [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go).
- Then review the example behavior change in [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) and the integration assertions in [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go).
- Validate with:
  - `GOWORK=off go test ./...`
  - `GOWORK=off go build ./...`
  - `GOWORK=off go run ./cmd/goja-gha run --script ./examples/set-output.js`
  - `GOWORK=off go run ./cmd/goja-gha run --script README.md --print-parsed-fields | yq '.default.script.value'`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/permissions-audit.js --cwd /tmp/geppetto --event-path ./testdata/events/workflow_dispatch.json --json-result | jq '.selectedActionsStatus, .runnerOutput, .stepSummary'`

### Technical details

#### Final local audit result shape

```json
{
  "selectedActions": null,
  "selectedActionsStatus": "skipped-not-selected-policy",
  "selectedActionsReason": "selected-actions only applies when allowed_actions == \"selected\" (got all)",
  "runnerOutput": {
    "written": false,
    "error": "runner output file path is empty"
  },
  "stepSummary": {
    "written": false,
    "error": "runner summary file path is empty"
  }
}
```
