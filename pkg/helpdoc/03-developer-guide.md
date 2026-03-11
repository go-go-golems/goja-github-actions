---
Title: "Developer Guide"
Slug: "developer-guide"
Short: "Understand the package layout, runtime ownership rules, test strategy, and the workflow for adding or changing native modules."
Topics:
- goja
- github-actions
- javascript
Commands:
- run
- doctor
Flags:
- print-schema
- output
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This guide explains how to work on the codebase as a contributor. It is written for a new intern or a new maintainer who can read Go comfortably but does not yet know how this repository is organized or why certain implementation constraints exist.

The most important fact to internalize is that this repository is a runtime boundary project. Most bugs are not "plain business logic" bugs. They happen at the edges between Glazed decoding, Goja ownership rules, GitHub runner semantics, and JavaScript-facing API shape. If you keep that mental model, the package layout will feel coherent instead of arbitrary.

## Repository Map

This section explains what each major directory is for and why it exists.

| Path | Responsibility |
|---|---|
| `cmd/goja-gha/cmds` | Cobra/Glazed command wiring |
| `pkg/cli` | field definitions, defaults, middleware, decoding, validation |
| `pkg/runtime` | runtime creation, bindings, globals, script execution |
| `pkg/runnerfiles` | writes for `GITHUB_ENV`, `GITHUB_OUTPUT`, `GITHUB_PATH`, `GITHUB_STEP_SUMMARY` |
| `pkg/contextdata` | event payload and GitHub context shaping |
| `pkg/githubapi` | raw GitHub HTTP request building and response handling |
| `pkg/modules/core` | `@actions/core` |
| `pkg/modules/github` | `@actions/github` |
| `pkg/modules/io` | `@actions/io` |
| `pkg/modules/exec` | `@actions/exec` |
| `examples` | runnable JavaScript examples |
| `integration` | end-to-end CLI tests |
| `pkg/helpdoc` | embedded Glazed help pages |

Why this matters: when you add functionality, you should be able to answer which layer owns it. If you cannot, the change probably needs more design work before code.

## Request Flow Through The System

This section follows a `goja-gha run` invocation from CLI entry to script result. Understanding this flow is the fastest way to become productive in the codebase.

```text
cmd/goja-gha/cmds/run.go
  -> pkg/cli.DecodeSettings(...)
  -> pkg/runtime.NewSettings(...)
  -> pkg/runtime.BuildFactory(...)
  -> register module specs
  -> create runtime
  -> install process globals
  -> require(entrypoint)
  -> call exported function
  -> await Promise if needed
  -> emit JSON result / runner side effects
```

File references to read in order:

1. `cmd/goja-gha/cmds/run.go`
2. `pkg/cli/middleware.go`
3. `pkg/runtime/factory.go`
4. `pkg/runtime/globals.go`
5. `pkg/runtime/script_runner.go`

## The Owner-Thread Rule

This section covers the single most important correctness constraint in the repository. Goja is not safe to mutate from arbitrary goroutines. If you remember only one rule while adding features, remember this one.

The runtime relies on `go-go-goja/pkg/runtimeowner/runner.go` to serialize VM access. In this repository:

- synchronous JS calls run on the owner thread,
- asynchronous native work happens in background goroutines,
- Promise settlement and JS callback invocation return to the owner thread via `Runner.Post(...)`.

Pseudocode:

```text
JS calls native module
native module creates Promise
native module starts goroutine
goroutine does blocking work
goroutine posts resolve/reject back to runtime owner
owner thread settles Promise inside the VM
```

Why this matters:

- if you call a Goja function directly from a background goroutine, you are writing a race,
- if you settle a Promise off-thread, you are writing a race,
- if you add a listener callback and forget the owner-thread bridge, you are writing a race.

Concrete examples:

- `pkg/modules/exec/module.go` is the main reference for correct async settlement.
- `pkg/runtime/bindings.go` exists so modules can discover the current runtime owner safely.

## How To Add A New Native Module

This section gives the recommended sequence for adding a module without scattering logic across the wrong layers.

### Step 1: Define The JS Contract

Write down:

- module name,
- exported functions,
- input shape,
- returned value shape,
- error behavior.

Do this before writing Go code. JS-facing shape should be stable and lower-case.

### Step 2: Separate Domain Logic From Glue

If the module does real work, keep the core behavior in plain Go helper functions or service types. Use the module layer only for:

- argument decoding,
- result normalization,
- Goja object wiring,
- owner-thread interactions.

### Step 3: Add The Module Package

Create a package under `pkg/modules/<name>`. A minimal module needs:

- a `Dependencies` struct,
- a `Module` struct,
- a `Spec(...)` function returning `ggjengine.ModuleSpec`,
- exported JS functions wired in the loader.

### Step 4: Register It In `run`

Add the module spec in `cmd/goja-gha/cmds/run.go`.

### Step 5: Test At Two Levels

Write:

- focused package tests for the module itself,
- an end-to-end example or CLI integration test if the module is user-visible.

## How To Decide Which Package Owns A Change

This section is here because new contributors often put the right code in the wrong place.

Use this rule set:

- If the change is about flag/env/config precedence, it belongs in `pkg/cli`.
- If the change is about `process`, CommonJS loading, or Promise awaiting, it belongs in `pkg/runtime`.
- If the change is about writing GitHub runner command files, it belongs in `pkg/runnerfiles`.
- If the change is about HTTP routes, headers, pagination, or GitHub error handling, it belongs in `pkg/githubapi`.
- If the change is about JS-visible module behavior, it belongs in `pkg/modules/...`.

Anti-pattern:

```text
native module calls os.Getenv(...)
```

Preferred pattern:

```text
CLI resolves settings
runtime stores settings
module reads from runtime settings or runtime state
```

## Test Strategy

This section explains why the repo uses multiple test layers. A single test style is not enough here.

### Unit-Style Package Tests

Use these for:

- middleware precedence,
- request building,
- runner-file formatting,
- module behavior with small fixture scripts.

Examples:

- `pkg/cli/middleware_test.go`
- `pkg/githubapi/request_test.go`
- `pkg/modules/core/module_test.go`
- `pkg/modules/exec/module_test.go`

### Integration Tests

Use these for:

- real CLI invocation,
- JSON result behavior,
- fake GitHub API coverage,
- example-script verification.

Main reference:

- `integration/examples_test.go`

### Manual Smokes

Use these when changing docs, examples, or packaging:

```bash
go run ./cmd/goja-gha run --script ./examples/trivial.js --json-result
go run ./cmd/goja-gha doctor --script ./examples/trivial.js --output json
```

## Common Failure Modes For Contributors

This section explains the mistakes you are most likely to make when changing the codebase.

### Returning Raw Go Structs To JS

Problem: the JS side sees `Repo`, `Data`, or `Stdout` instead of `repo`, `data`, or `stdout`.

Fix: normalize to plain `map[string]interface{}` before exposing a result to JavaScript.

### Reading Directly From Host Env In Modules

Problem: behavior becomes inconsistent with `doctor`, config files, and test fixtures.

Fix: flow everything through decoded settings and runtime state.

### Mixing Runtime Logic And GitHub API Logic

Problem: tests become harder to isolate and module boundaries blur.

Fix: keep `pkg/runtime` generic and keep GitHub-specific HTTP behavior in `pkg/githubapi`.

## Review Checklist For A Change

This section is meant to be used before you open a PR or before you commit a large slice.

- Does the change preserve the owner-thread rule?
- Does it keep JS-visible property names lower-case and explicit?
- Does it avoid direct `os.Getenv(...)` in runtime and module code?
- Does it have at least one test at the correct layer?
- Does it need a doc/example update?
- Does it change runner semantics or config precedence in a way `doctor` should surface?

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A new module works in Go tests but crashes in a CLI run | You violated an owner-thread rule or returned an unsafe Goja value | Reproduce with a small script and inspect where VM access crosses goroutines |
| A JS property name looks wrong | A Go struct leaked into JS without normalization | Convert the result to a plain map first |
| `doctor` disagrees with runtime behavior | The module or runtime read host env directly | Route the value through decoded settings |
| A fake GitHub API test passes but real usage fails | The helper route or auth assumptions differ from real endpoints | Inspect request headers and route interpolation in `pkg/githubapi` |

## See Also

- `goja-gha help user-guide`
- `goja-gha help javascript-api`
- `goja-gha help design-and-internals`
