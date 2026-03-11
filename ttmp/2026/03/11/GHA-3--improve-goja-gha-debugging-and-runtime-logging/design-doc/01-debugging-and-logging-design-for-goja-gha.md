---
Title: Debugging and logging design for goja-gha
Ticket: GHA-3
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-gha/cmds/root.go
      Note: Existing Glazed zerolog setup and root logging flags
    - Path: cmd/goja-gha/cmds/run.go
      Note: Best place to log decoded run settings and command startup context
    - Path: pkg/githubapi/client.go
      Note: Primary place for GitHub request and response tracing
    - Path: pkg/helpdoc/06-debugging-goja-gha.md
      Note: User-facing debugging workflow and logging guide
    - Path: pkg/modules/github/module.go
      Note: Current GitHub client construction and API URL/token fallback path
    - Path: pkg/runtime/script_runner.go
      Note: Runtime execution milestones and promise failure context
    - Path: ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/scripts/console-debug.js
      Note: Validated script used to confirm console output during local runs
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T10:10:18.477131417-04:00
WhatFor: ""
WhenToUse: ""
---



# Debugging and logging design for goja-gha

## Executive Summary

`goja-gha` already has Glazed's root logging section wired in, via [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go). That means the missing piece is not "introduce zerolog from scratch". The missing piece is to actually emit useful debug events at the boundaries where users get lost today: settings resolution, runtime bootstrap, JavaScript console output, and GitHub HTTP requests.

The central recommendation is to add structured debug logging in four places:

- CLI settings and environment resolution,
- runtime initialization and script execution,
- GitHub API request/response handling,
- JavaScript `console.*` output capture and forwarding.

For a new engineer, the simplest way to frame the work is:

1. keep Glazed/Cobra logging initialization as-is,
2. add contextual logs where data crosses a subsystem boundary,
3. avoid leaking secrets,
4. make `--log-level debug` and JS `console.log(...)` genuinely useful for field debugging.

## Problem Statement

When a script fails today, the user often sees only the final error envelope. For example:

```text
Error: execute exported function: GoError: github api error: status 401: Bad credentials
```

That error is accurate but not sufficient for efficient debugging. It does not tell the user:

- whether a token was present,
- which token source won,
- which repository was targeted,
- which route failed,
- whether the base URL was default or overridden,
- whether the failure happened before or after JavaScript code ran,
- whether `console.log(...)` output was produced but lost in the noise.

There are also two additional sources of confusion:

### 1. Logging support already exists, but is easy to miss

[`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go) already uses:

- `logging.AddLoggingSectionToRootCommand(root, appName)`
- `logging.InitLoggerFromCobra(cmd)`

So the CLI already has Glazed/zerolog support. The gap is that the rest of the application emits almost no structured logs.

### 2. JavaScript console behavior is not documented as a debugging tool

The runtime includes `console`, via the `go-go-goja` engine stack, but there is no explicit `goja-gha` debugging story around `console.log`, `console.error`, or routing those outputs in a way that helps correlate script-level activity with Go-side traces.

## Proposed Solution

Introduce a layered debugging model with one consistent rule:

> Every subsystem boundary should emit enough information to explain what it is doing, but never enough to leak secrets.

### Layer 1: CLI and settings logs

Add debug logs during command startup and settings decode:

- command name,
- script path,
- cwd,
- event path,
- workspace,
- whether GitHub token is present,
- whether JSON output is enabled.

Do not log raw token values. Log booleans or masked summaries only.

### Layer 2: Runtime bootstrap logs

Add debug logs around:

- runtime creation,
- module registration,
- entrypoint selection,
- exported function invocation,
- promise awaiting and settlement.

These logs should live close to [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go) and related runtime setup files.

### Layer 3: GitHub API request tracing

Add structured logs in [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go) for:

- normalized base URL,
- HTTP method,
- route string,
- resolved request URL,
- response status,
- whether auth header was present,
- request duration.

Never log the raw bearer token. If useful, log only:

- `auth_present=true|false`
- maybe token length or source category if that becomes explicit elsewhere

### Layer 4: JavaScript console forwarding

Make the script's own `console.log(...)`, `console.warn(...)`, and `console.error(...)` part of the debugging story.

Recommended first step:

- keep JS console on stderr/stdout as it already behaves through `go-go-goja`,
- document it clearly,
- optionally add a thin wrapper or prefixing layer later if correlation becomes difficult.

## Design Decisions

### Decision 1: Treat this as an observability ticket, not a parser rewrite ticket

Rationale:

- the user explicitly deprioritized the Glazed settings refactor,
- the current blocker is operational debugging,
- small, targeted logs provide faster value than another architectural detour.

### Decision 2: Reuse existing Glazed logging support

Rationale:

- `root.go` already wires Glazed's logging section,
- reusing that path avoids duplicate flag systems,
- users should debug through a standard `--log-level` flow instead of a bespoke switch if possible.

### Decision 3: Log presence and provenance, not secrets

Rationale:

- tokens and request auth are the first thing users want to verify,
- raw credential logging is unsafe,
- "present/missing/masked/source" is usually enough.

### Decision 4: Prefer boundary logs over chatty internal logs

Rationale:

- too many logs make 401-style debugging harder, not easier,
- the most useful messages are at subsystem handoff points.

### Decision 5: Make `console.log` part of the documented debugging workflow

Rationale:

- many users debug JS first, not Go first,
- if the runtime exposes `console`, the user should know they can rely on it,
- script logs and Go logs complement each other.

## Alternatives Considered

### Alternative A: Add a one-off `fmt.Printf` near the 401 path

Rejected as the main approach.

Why:

- solves one symptom, not the debugging model,
- does not help with future failures in runtime/bootstrap/config paths,
- is hard to keep consistent.

### Alternative B: Add a dedicated `--debug-http` flag and separate logger

Rejected for now.

Why:

- the CLI already has Glazed logging support,
- another flag family would fragment debugging behavior,
- zerolog filtering should be enough initially.

### Alternative C: Depend only on JavaScript `console.log`

Rejected.

Why:

- JS logs cannot explain Go-side request construction or config decoding,
- a failure may happen before useful JS code runs.

## Implementation Plan

### Phase 1: Surface the existing logging contract

- audit which logging flags Glazed already exposes on `goja-gha`,
- document `--log-level debug` and related root logging behavior,
- add a help/example snippet for debugging a failing GitHub API script.

### Phase 2: Add CLI and runtime boundary logs

Files:

- [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go)

Tasks:

- log decoded high-level settings with masking,
- log runtime creation and entrypoint execution milestones,
- log promise rejection/timeout context.

### Phase 3: Add GitHub request tracing

Files:

- [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go)
- possibly [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go)

Tasks:

- emit request start/end logs,
- include route, base URL, auth presence, status, and duration,
- improve wrapped errors with more path/route context if useful.

### Phase 4: Document and test JavaScript console behavior

Files:

- help docs under [`pkg/helpdoc`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc)
- examples under [`examples`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples)

Tasks:

- write a short debugging help page,
- add example scripts or snippets that use `console.log` to print resolved context,
- validate that console output is visible during local runs.

### Phase 5: Add debugging regression checks

Tasks:

- add tests for masked logging helpers if introduced,
- add a fake-server integration test that asserts 401 failures include route/status context,
- add one smoke test that runs with debug logging enabled.

## Open Questions

### 1. What exact logging flags does Glazed expose on this root command today?

This should be confirmed with `goja-gha --help --long-help` and documented directly.

### 2. Does `go-go-goja` already route `console.*` exactly the way we want?

If not, we need to decide whether to wrap it locally or just document current behavior.

### 3. Should token provenance be logged?

For example:

- `token_source=flag`
- `token_source=runner-env`
- `token_source=resolved-setting`

This may be useful, but only if the provenance is reliable and not misleading.

## References

- [`cmd/goja-gha/cmds/root.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/root.go)
- [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [`pkg/runtime/script_runner.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go)
- [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go)
- [`pkg/githubapi/client.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/githubapi/client.go)
- [`glazed/cmd/glaze/main.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/cmd/glaze/main.go)
