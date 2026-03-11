---
Title: goja-github-actions design and implementation guide
Ticket: GHA-1
Status: active
Topics:
    - goja
    - github-actions
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/engine/factory.go
      Note: Shared runtime factory and module composition model to reuse
    - Path: go-go-goja/engine/module_specs.go
      Note: Defines static module specs and runtime initializers
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Owner-thread async settlement pattern for safe Goja callbacks
    - Path: goja-git/gitmodule.go
      Note: Precedent for JS-facing module surface and object graph
    - Path: goja-github-actions/cmd/goja-gha/cmds/doctor.go
      Note: |-
        Bootstrap glaze command that reports resolved settings for early validation
        Current bootstrap inspection flow referenced by the updated implementation guide
    - Path: goja-github-actions/cmd/goja-gha/cmds/run.go
      Note: |-
        Bootstrap bare command that validates decoded runner and GitHub settings
        Current bootstrap state referenced by the updated implementation guide
    - Path: goja-github-actions/go.mod
      Note: Bootstrap module path and dependencies now match the real repository
    - Path: goja-github-actions/pkg/cli/github_actions.go
      Note: |-
        Current Glazed schema split between shared GitHub fields and default runner fields
        Updated schema-boundary design now matches this file
    - Path: goja-github-actions/ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/sources/local/01-imported-planning-notes.md
      Note: Imported planning note defining the first serious GitHub Actions policy/audit use case
ExternalSources: []
Summary: Detailed architecture and phased implementation guide for building a Goja-based GitHub Actions scripting tool.
LastUpdated: 2026-03-10T22:14:00-04:00
WhatFor: Explain how to build goja-gha on top of go-go-goja, map GitHub Actions concepts into Goja modules, and deliver a practical first milestone.
WhenToUse: Use when implementing or reviewing the initial goja-github-actions architecture and runtime/module boundaries.
---



# goja-github-actions design and implementation guide

## Executive Summary

`goja-github-actions` is no longer a blank template. The repository now has a real `goja-gha` entrypoint, Glazed/Cobra command wiring for `run` and `doctor`, and an explicit schema split between runner-facing default flags and a shared `github-actions` section for GitHub-specific cross-command fields. What it still does not have is the runtime itself: no Goja factory wiring, no native `@actions/*` modules, and no middleware yet for runner-env resolution. The fastest credible path remains the same: build on the explicit runtime composition already present in `go-go-goja`, then package GitHub Actions concepts as native modules with a cleaner boundary than the hand-rolled approach used in `goja-git`.

The proposed deliverable is a new CLI tool named `goja-gha`. It runs JavaScript through Goja, exposes a small but useful GitHub Actions API surface through `require("@actions/core")`, `require("@actions/github")`, `require("@actions/io")`, and optional follow-on modules, and lets engineers write repository automation in JavaScript without shipping Node.js code. The first concrete use case is the imported planning note in `sources/local/01-imported-planning-notes.md`, which describes a policy/audit tool for GitHub Actions permissions and workflow hygiene. That is a strong initial target because it mostly needs `core` output primitives, a GitHub API client, file reads for workflow YAML, and a predictable CLI/runtime model.

Two constraints shape the design:

1. `go-go-goja` already solves the hard runtime-lifecycle problem: explicit factory composition, `require()` setup, module-root resolution, and safe owner-thread callbacks through `runtimeowner.Runner`.
2. GitHub Actions does not allow a custom JavaScript runtime in `action.yml`; the official metadata syntax supports `node20` and `node24` for JavaScript actions. That means `goja-gha` should be delivered as a standalone CLI first, and as a composite or container action wrapper later if we want first-class GitHub Actions distribution.

## Problem Statement

The repo needs a way to write GitHub automation in JavaScript while still shipping a Go-based toolchain. The desired developer experience is close to GitHub's own toolkit:

- read inputs and environment variables,
- emit outputs, environment exports, paths, notices, warnings, and summaries,
- access workflow/event context in a predictable object,
- call GitHub REST endpoints with authenticated helpers,
- optionally perform file and process operations needed by real actions,
- run the same script locally and inside CI with minimal differences.

Today the local codebase only partially provides that:

- `goja-github-actions/go.mod` now declares the real module path and pulls in Glazed/Cobra dependencies.
- `goja-github-actions/cmd/goja-gha/...` provides bootstrap `run` and `doctor` commands plus root logging/help wiring.
- `goja-github-actions/pkg/cli/github_actions.go` now defines the first Glazed schema boundary, but only for static decoding and display.
- There is still no runtime package, no `@actions/*` module implementation, and no runner middleware.

We do, however, have two useful precedents:

1. `go-go-goja` shows the preferred runtime architecture.
2. `goja-git` shows what a JS-facing domain module can look like, and also shows which shortcuts we should avoid repeating.

### Scope

In scope for this design:

- a detailed architecture for `goja-gha`,
- repository layout and package boundaries,
- the JS API surface for the MVP,
- how to map GitHub Actions runner concepts into Go/Goja,
- a phased implementation plan,
- testing, packaging, and delivery guidance for an intern.

Out of scope for the MVP:

- full Octokit compatibility,
- full `@actions/toolkit` parity across every package,
- npm module compatibility,
- ESM support,
- artifact/cache parity on day one,
- a Marketplace-ready wrapper action in the first commit.

## Current State Analysis

### 1. The destination repo now has a bootstrap CLI but not a runtime

Observed files:

- `goja-github-actions/go.mod`
- `goja-github-actions/cmd/goja-gha/main.go`
- `goja-github-actions/cmd/goja-gha/cmds/root.go`
- `goja-github-actions/cmd/goja-gha/cmds/run.go`
- `goja-github-actions/cmd/goja-gha/cmds/doctor.go`
- `goja-github-actions/pkg/cli/github_actions.go`
- `goja-github-actions/README.md`

Observed behavior:

- `goja-github-actions/go.mod` now uses the real module path and a minimal dependency set for the bootstrap CLI.
- `goja-github-actions/cmd/goja-gha/cmds/root.go` builds the root command and enables logging/help wiring.
- `goja-github-actions/cmd/goja-gha/cmds/run.go` validates and prints decoded bootstrap settings, then exits with a clear not-implemented runtime error.
- `goja-github-actions/cmd/goja-gha/cmds/doctor.go` reports resolved settings through Glazed output.
- `goja-github-actions/pkg/cli/github_actions.go` splits shared GitHub fields (`workspace`, `github-token`) from runner-facing default flags (`script`, `event-path`, runner file paths, `cwd`, `debug`, `json-result`).
- There are still no packages for runtime setup, native modules, runner-file writers, or GitHub API access.

Implication:

- the design still needs to specify most of the codebase, but it should now treat the bootstrap CLI and current schema split as the fixed starting point rather than as proposed future work.

### 2. `go-go-goja` already has the runtime composition we need

Key evidence:

- `go-go-goja/engine/factory.go:15-29` defines `FactoryBuilder` and `Factory`, separating build-time module composition from runtime creation.
- `go-go-goja/engine/factory.go:90-131` validates unique IDs, builds a `require.Registry`, and registers modules into an immutable factory.
- `go-go-goja/engine/factory.go:134-179` creates a new runtime with `goja.New()`, an event loop, `console`, `require`, and a `runtimeowner.Runner`.
- `go-go-goja/engine/runtime.go:21-49` gives the runtime explicit lifecycle and shutdown behavior.
- `go-go-goja/engine/module_specs.go:14-82` distinguishes static `ModuleSpec` registration from per-runtime initialization.
- `go-go-goja/engine/module_roots.go:11-119` solves module-root derivation from the script path.
- `go-go-goja/modules/common.go:29-102` provides a registry model for native modules.
- `go-go-goja/pkg/runtimeowner/types.go:20-33` and `go-go-goja/pkg/runtimeowner/runner.go:62-189` show the correct pattern for synchronous and asynchronous owner-thread access.

Why this matters:

- the hardest part of Goja systems is not exporting one Go function into JS; it is preserving runtime safety when async work resolves back into the VM.
- `go-go-goja` already has the lifecycle and module-registration model we want. Reusing it reduces risk immediately.

### 3. `goja-git` is a useful precedent and a warning

Useful ideas:

- `goja-git/gitmodule.go:101-110` installs a domain module into the runtime.
- `goja-git/gitmodule.go:161-198` constructs a JS object graph with nested namespaces like `repo.branch.list`.
- `goja-git/gitmodule.go:581-600` shows a simple `mustExport` helper for object-to-struct conversion.

Problems we should not copy:

- `goja-git/main.go:23-49` manually wires a runtime, console, and custom JSON helpers instead of reusing a shared engine.
- `goja-git/gitmodule.go` keeps everything in a single large file, which is manageable for Git demos but not for an API surface like GitHub Actions.
- Panics are converted into Goja errors ad hoc, but there is no structured runtime state, no shared context object, and no package layering between transport, domain logic, and JS adapters.

Implication:

- model the module ergonomics after `goja-git`, but model the runtime/lifecycle after `go-go-goja`.

### 4. The imported planning note defines the first serious workload

The imported source in `sources/local/01-imported-planning-notes.md` describes a real policy/audit tool around GitHub Actions permissions and workflow file inspection. It identifies:

- repo and org Actions permissions endpoints,
- workflow listing endpoints,
- contents APIs for `.github/workflows`,
- the need for repo/org policy inspection plus workflow YAML linting,
- a candidate Go client structure.

Why this matters:

- it gives us a concrete product target for the bindings,
- it narrows the first JS API surface to `core`, `github`, and file access,
- it shows that the first scripts will need authenticated REST requests and readable workflow context.

### 5. Official GitHub Actions references define the compatibility boundary

The current official docs matter because GitHub Actions behavior changes over time:

- The metadata syntax reference currently documents `node20` and `node24` as the JavaScript runtimes for `runs.using`. That means `goja-gha` is not a drop-in JavaScript action runtime; it must be wrapped by a composite or container action if we want GitHub-hosted execution.
- The workflow commands docs define file-based contracts for `GITHUB_ENV`, `GITHUB_OUTPUT`, and `GITHUB_STEP_SUMMARY`.
- The variables reference documents runner-provided paths like `GITHUB_OUTPUT` and `GITHUB_WORKSPACE`.
- The contexts reference defines what data users expect in `github`, `env`, `runner`, `job`, `matrix`, `steps`, and related objects.
- The `actions/toolkit` repo documents the canonical package boundaries of `@actions/core`, `@actions/exec`, `@actions/io`, `@actions/glob`, `@actions/github`, and friends.
- `actions/github-script` documents a closely related UX: injected helpers, a wrapped `require`, optional retries, and step result output.

Those docs are the compatibility reference, but the MVP should aim for behavioral similarity rather than complete implementation parity.

## Gap Analysis

### Architectural gaps

- No executable Goja runtime exists in `goja-github-actions`.
- The package layout only exists for CLI/bootstrap schema code; runtime, modules, config resolution, and transport packages are still missing.
- No strategy exists yet for mapping GitHub runner state into the VM beyond decoded CLI settings.
- No async story exists yet for HTTP and process operations.

### Product gaps

- No `@actions/core` equivalent for inputs, outputs, and log commands.
- No `@actions/github` equivalent for context and REST access.
- No local runner mode that can load an event payload and workspace.
- No guidance for packaging the tool as a reusable action wrapper.

### Quality gaps

- No tests.
- No fixture data.
- No example scripts.
- No clear error model for JS authors.

## Proposed Architecture and APIs

### Design principles

1. Reuse `go-go-goja` for runtime ownership, module registration, and `require()`.
2. Keep JS-visible packages small and familiar.
3. Separate domain logic from JS adapter code.
4. Prefer stable, typed Go internals and thin JS shims.
5. Support a realistic MVP quickly, then grow package coverage.

### High-level component diagram

```text
                   +---------------------------+
                   |       goja-gha CLI        |
                   |  flags, env, action.yml   |
                   +-------------+-------------+
                                 |
                                 v
                   +---------------------------+
                   |      runtime config       |
                   | script path, workspace,   |
                   | event json, token, env    |
                   +-------------+-------------+
                                 |
                                 v
                   +---------------------------+
                   |   go-go-goja Factory      |
                   | require registry + init   |
                   +------+------+-------------+
                          |      |
        +-----------------+      +------------------+
        v                                           v
+------------------+                     +----------------------+
| native modules   |                     | runtime initializers |
| @actions/core    |                     | globals, process,    |
| @actions/github  |                     | context snapshots    |
| @actions/io      |                     +----------+-----------+
| @actions/exec    |                                |
+---------+--------+                                v
          |                              +----------------------+
          +----------------------------->|    Goja runtime      |
                                         | script + promises    |
                                         +----------+-----------+
                                                    |
                                                    v
                                         +----------------------+
                                         | runner side effects  |
                                         | stdout, env files,   |
                                         | API calls, outputs   |
                                         +----------------------+
```

### Recommended repository layout

```text
goja-github-actions/
|-- cmd/goja-gha/
|   |-- main.go
|   `-- cmds/
|       |-- root.go
|       |-- run.go
|       `-- doctor.go
|-- pkg/runtime/
|   |-- factory.go
|   |-- globals.go
|   `-- script_runner.go
|-- pkg/cli/
|   |-- defaults.go
|   |-- middleware.go
|   `-- github_actions.go
|-- pkg/contextdata/
|   |-- github_context.go
|   |-- runner_context.go
|   `-- event_loader.go
|-- pkg/runnerfiles/
|   |-- envfile.go
|   |-- outputfile.go
|   |-- pathfile.go
|   `-- summaryfile.go
|-- pkg/githubapi/
|   |-- client.go
|   |-- request.go
|   |-- actions_permissions.go
|   |-- contents.go
|   `-- workflows.go
|-- pkg/modules/core/
|   |-- module.go
|   |-- inputs.go
|   |-- outputs.go
|   `-- logging.go
|-- pkg/modules/github/
|   |-- module.go
|   |-- context.go
|   |-- client.go
|   `-- rest_actions.go
|-- pkg/modules/io/
|   |-- module.go
|   `-- fs.go
|-- pkg/modules/exec/
|   |-- module.go
|   `-- exec.go
|-- examples/
|   |-- permissions-audit.js
|   |-- list-workflows.js
|   `-- set-output.js
`-- testdata/
    |-- events/
    |-- workflows/
    `-- scripts/
```

### Runtime composition

The runtime should be built with `go-go-goja` rather than manually wiring Goja:

1. derive module roots from the script path,
2. register native modules explicitly,
3. install runtime-scoped globals and context objects,
4. execute the script through the shared runtime.

Recommended flow:

```go
settings := &RunSettings{}
if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
    return err
}

factory, err := engine.NewBuilder(
    engine.WithModuleRootsFromScript(settings.Script, engine.DefaultModuleRootsOptions()),
).WithModules(
    coremodule.Spec(settings),
    githubmodule.Spec(settings),
    iomodule.Spec(settings),
    execmodule.Spec(settings),
).WithRuntimeInitializers(
    runtimeglobals.Initializer(settings),
).Build()

rt, err := factory.NewRuntime(ctx)
defer rt.Close(ctx)

result, err := runner.RunFile(ctx, rt, settings.Script)
```

Why this is the right pattern:

- `go-go-goja/engine/factory.go:90-131` already freezes module registration into a reusable factory.
- `go-go-goja/engine/module_roots.go:29-119` already resolves script-relative module folders.
- `go-go-goja/pkg/runtimeowner/runner.go:62-189` already handles owner-thread calls and async posts.

### Glazed command model

The CLI should be a Glazed application, not a hand-built Cobra app with ad hoc flag parsing and `os.Getenv` lookups. The command layer should follow the established Glazed pattern from the skill and from commands like `openai-app-server/cmd/openai-app-server/thread_read_command.go:17-83`:

1. define a command struct embedding `*cmds.CommandDescription`,
2. define a settings struct with `glazed:"..."` tags,
3. build flags with `fields.New(...)`,
4. add `settings.NewGlazedSchema()` and `cli.NewCommandSettingsSection()`,
5. decode with `vals.DecodeSectionInto(...)`,
6. keep env/config/profile precedence in Glazed middleware rather than reading env vars in business logic.

This matters for `goja-gha` because the GitHub runner values are numerous and easy to scatter incorrectly. The design should keep runner-facing flags in the default command section and only keep the truly shared GitHub settings in a separate `github-actions` section so the command contract stays explicit, introspectable, and testable.

### GitHub Actions schema section

Instead of a `pkg/config` package that manually loads `GITHUB_*` values, keep the schema split aligned with command semantics:

- runner-facing execution settings belong in the default Glazed section,
- GitHub-specific cross-command settings belong in `pkg/cli/github_actions.go` under the `github-actions` section,
- middleware may still populate defaults from environment variables or config profiles, but command and runtime code should only consume decoded settings structs.

Recommended settings structs:

```go
type RunnerSettings struct {
    Script            string `glazed:"script"`
    EventPath         string `glazed:"event-path"`
    ActionPath        string `glazed:"action-path"`
    RunnerEnvFile     string `glazed:"runner-env-file"`
    RunnerOutputFile  string `glazed:"runner-output-file"`
    RunnerPathFile    string `glazed:"runner-path-file"`
    RunnerSummaryFile string `glazed:"runner-summary-file"`
    Cwd               string `glazed:"cwd"`
    Debug             bool   `glazed:"debug"`
    JSONResult        bool   `glazed:"json-result"`
}

type GitHubActionsSettings struct {
    Workspace   string `glazed:"workspace"`
    GithubToken string `glazed:"github-token"`
}
```

Recommended default-section field definitions:

```go
fields.New("script", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Path to the JavaScript entry script"))
fields.New("event-path", fields.TypeString, fields.WithHelp("Path to the GitHub event payload JSON"))
fields.New("action-path", fields.TypeString, fields.WithHelp("Action directory when running inside GitHub Actions"))
fields.New("runner-env-file", fields.TypeString, fields.WithHelp("Override path for the runner env file during local/test runs"))
fields.New("runner-output-file", fields.TypeString, fields.WithHelp("Override path for the runner output file during local/test runs"))
fields.New("runner-path-file", fields.TypeString, fields.WithHelp("Override path for the runner path file during local/test runs"))
fields.New("runner-summary-file", fields.TypeString, fields.WithHelp("Override path for the runner summary file during local/test runs"))
fields.New("cwd", fields.TypeString, fields.WithHelp("Working directory for script execution"))
fields.New("debug", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Enable verbose runtime logging"))
fields.New("json-result", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Emit final script result as JSON"))
```

Recommended shared GitHub section field definitions:

```go
fields.New("workspace", fields.TypeString, fields.WithHelp("Workspace directory shared by GitHub-oriented commands"))
fields.New("github-token", fields.TypeString, fields.WithHelp("GitHub token shared by commands that call the GitHub API"))
```

The important boundary is this: no lower-level package should call `os.Getenv("GITHUB_EVENT_PATH")` or similar directly. If the runtime needs runner data, it receives `RunnerSettings`. If a GitHub module needs a token or workspace, it receives `GitHubActionsSettings`. Environment lookup is an input-resolution concern, not a runtime concern.

### Command grouping

Use explicit Cobra parents that mirror folders, per the Glazed skill:

```text
goja-gha run ...
goja-gha doctor ...
```

That means:

- `cmd/goja-gha/main.go` wires the root Cobra command,
- `cmd/goja-gha/cmds/root.go` registers subcommands,
- `cmd/goja-gha/cmds/run.go` defines the `run` Glazed command,
- `cmd/goja-gha/cmds/doctor.go` defines the `doctor` Glazed command.

The root should also attach the standard logging section and initialize logging in `PersistentPreRunE`.

### Custom parser middleware

The Glazed skill explicitly allows custom middlewares when config/env/profile precedence matters. `goja-gha` should use that mechanism for GitHub runner defaults.

Recommended precedence:

1. explicit CLI flags,
2. optional local config file/profile,
3. GitHub runner environment variables,
4. hardcoded defaults.

This keeps the behavior transparent:

- the schema declares the fields,
- middleware resolves defaults,
- `RunSettings` becomes the only input to the runtime.

### CLI contract

Recommended first command:

```text
goja-gha run ./examples/permissions-audit.js \
  --event-path ./testdata/events/workflow_dispatch.json \
  --workspace "$PWD" \
  --github-token "$GITHUB_TOKEN"
```

Recommended schema fields for `run`:

- `--script`: JS entrypoint; required
- `--event-path`: event payload JSON path
- `--action-path`: action directory
- `--runner-env-file`, `--runner-output-file`, `--runner-path-file`, `--runner-summary-file`: runner command file overrides
- `--cwd`: script working directory
- `--debug`: verbose logs
- `--json-result`: print final script result as JSON
- `--workspace`: shared workspace directory
- `--github-token`: shared GitHub token for API calls

Future commands:

- `goja-gha doctor` to validate the resolved schema values, runner-file paths, and token availability,
- `goja-gha repl` for interactive experimentation,
- `goja-gha package` if we later automate wrapper action generation.

### JS authoring model

Prefer file-based scripts that use familiar `require()` names:

```javascript
const core = require("@actions/core");
const gha = require("@actions/github");
const io = require("@actions/io");

async function main() {
  const token = core.getInput("github-token", { required: true });
  const octokit = gha.getOctokit(token, { retries: 2 });
  const context = gha.context;

  const workflowDir = `${context.workspace}/.github/workflows`;
  const files = await io.readdir(workflowDir);

  core.notice(`Scanning ${files.length} workflow files for ${context.repo.owner}/${context.repo.repo}`);

  const perms = await octokit.request("GET /repos/{owner}/{repo}/actions/permissions", {
    owner: context.repo.owner,
    repo: context.repo.repo,
  });

  core.setOutput("allowed_actions", perms.data.allowed_actions);
  return perms.data;
}

main().catch((err) => {
  core.setFailed(err.message);
});
```

This is intentionally closer to `actions/github-script` than to a bespoke DSL. Engineers should feel that they are writing ordinary action-flavored JavaScript, not learning a second framework.

### `@actions/core` MVP surface

Implement first:

- `getInput(name, options?)`
- `getBooleanInput(name, options?)`
- `getMultilineInput(name, options?)`
- `setOutput(name, value)`
- `exportVariable(name, value)`
- `addPath(path)`
- `setSecret(value)`
- `debug(message)`
- `info(message)`
- `notice(message, properties?)`
- `warning(message, properties?)`
- `error(message, properties?)`
- `startGroup(name)`
- `endGroup()`
- `group(name, asyncFn)`
- `summary.addRaw(...).addHeading(...).write()`
- `setFailed(message)`

Internal mapping:

- inputs come from `INPUT_<NORMALIZED_NAME>` and optionally action metadata defaults,
- outputs append to the resolved `output-file` setting,
- exports append to the resolved `env-file` setting,
- path updates append to the resolved `path-file` setting,
- summaries append to the resolved `summary-file` setting,
- masking and annotations write workflow-command compatible lines to stdout.

Important behavioral note from the GitHub docs:

- values written to the runner env file affect later steps, not the current process state. We should still update the in-process env map optionally for local ergonomics, but the design should clearly label that as local emulation rather than exact runner behavior.

### `@actions/github` MVP surface

Implement first:

- `context` object snapshot
- `getOctokit(token, options?)`
- `request(route, params)`
- `paginate(route, params)`

Pragmatic API shape:

```javascript
const gha = require("@actions/github");
const api = gha.getOctokit(token, { retries: 2 });

const permissions = await api.request("GET /repos/{owner}/{repo}/actions/permissions", {
  owner,
  repo,
});
```

Expose a curated `rest` namespace only for the first concrete domain:

```javascript
api.rest.actions.getRepoPermissions(...)
api.rest.actions.setRepoPermissions(...)
api.rest.actions.getSelectedActions(...)
api.rest.actions.getWorkflowPermissions(...)
api.rest.repos.getContent(...)
api.rest.actions.listRepoWorkflows(...)
```

Why not implement all of Octokit immediately:

- the surface is too large,
- the imported plan only needs a narrow subset,
- a generic `request()` method gives flexibility without blocking the first scripts.

### Context object model

Populate a JS object shaped around documented GitHub expectations:

```text
github.context = {
  eventName,
  action,
  actor,
  workflow,
  job,
  runId,
  runNumber,
  sha,
  ref,
  repository,
  repo: { owner, repo },
  payload,
}
```

Data sources:

- resolved schema values for action/workflow metadata,
- middleware-provided defaults derived from runner environment when available,
- JSON loaded from `settings.EventPath`.

Do not attempt to recreate expression-time contexts like `matrix`, `needs`, or `steps` unless we explicitly pass them in via JSON. Those contexts are resolved by the workflow engine before the runner executes user code.

### Process/global model

We should provide a narrow `process` shim for compatibility:

- `process.env`
- `process.cwd()`
- `process.exitCode`
- `process.stdout.write()`
- `process.stderr.write()`

Do not attempt to ship full Node compatibility:

- no full `Buffer` implementation in phase 1,
- no `child_process`,
- no built-in npm package resolution,
- no ESM.

`console` should come from `go-go-goja`, not from a custom implementation like `goja-git/main.go:28-38`.

### File and process helpers

`@actions/io` phase 1:

- `mkdirP`
- `rmRF`
- `cp`
- `mv`
- `which`
- `readdir`
- `readFile`
- `writeFile`

`@actions/exec` phase 1:

- `exec(command, args?, options?)`
- stdout/stderr listeners,
- current working directory override,
- environment override,
- exit code result.

Async requirement:

- any callback settlement from background goroutines must use `runtimeowner.Runner.Post(...)`.
- that follows the pattern documented in `go-go-goja/pkg/runtimeowner/runner.go:108-158`.

### GitHub transport layer

Create a pure-Go client package under `pkg/githubapi` rather than making JS adapters build HTTP requests directly.

Responsibilities:

- auth header management,
- API version header management,
- retry policy,
- pagination,
- typed structs for the first endpoints we care about,
- translation into JS-friendly plain objects.

Initial endpoint groups, taken directly from the imported plan:

- repository actions permissions,
- selected actions allowlists,
- workflow permissions,
- fork PR approval policy,
- workflow metadata,
- repository contents for `.github/workflows`.

This keeps business logic testable in Go without booting the VM for every edge case.

### Error model

Recommended conventions:

- Go internals return typed Go errors,
- module entrypoints translate those to Goja exceptions,
- user-facing messages should include the action or route being attempted,
- `core.setFailed()` sets a shared failure state and `process.exitCode = 1`,
- the CLI exits non-zero if any uncaught exception or explicit failure state exists.

### Packaging and distribution model

Because GitHub only supports `node20`/`node24` for JavaScript actions, do not model this as a JavaScript action artifact.

Recommended rollout:

1. Ship `goja-gha` as a CLI for local use and CI scripts.
2. Add examples and fixtures.
3. Later add one wrapper action:
   - container action if Linux-only is acceptable and operational simplicity matters most,
   - composite action if cross-platform execution matters more and we are willing to manage binary packaging.

For a composite wrapper, the action can invoke a bundled binary using the resolved `action-path` setting. For a container wrapper, the Go binary becomes the image entrypoint.

## Design Decisions

### Decision 1: Build on `go-go-goja`, not raw Goja

Rationale:

- `go-go-goja` already has explicit factory composition, module registration, module roots, and owner-thread lifecycle.
- repeating `goja-git/main.go` style manual wiring would duplicate solved problems.

### Decision 2: Emulate familiar package names

Rationale:

- `require("@actions/core")` and `require("@actions/github")` reduce onboarding cost.
- an intern can compare our behavior against the official toolkit docs directly.

### Decision 3: Start with a narrow, useful API surface

Rationale:

- the first real script is a policy/audit tool.
- it mostly needs `core`, `github`, file I/O, and possibly command execution.
- implementing every toolkit package before the first working script would stall delivery.

### Decision 4: Expose generic `request()` before full Octokit parity

Rationale:

- it keeps the MVP flexible,
- it avoids premature code generation or huge wrapper trees,
- it still enables all imported-plan endpoints immediately.

### Decision 5: Separate transport/domain logic from JS adapter code

Rationale:

- Go tests can exercise API clients and runner file writers without a JS runtime,
- JS modules stay small and easier to review,
- later code generation or additional modules becomes easier.

## Pseudocode and Key Flows

### 1. CLI boot flow

```text
resolve glazed schema values
  -> apply middleware precedence (flags/config/env/defaults)
  -> load event payload
  -> construct runtime settings struct
  -> build engine factory with module specs
  -> create runtime
  -> install globals/context snapshots
  -> run script
  -> flush summary/output state
  -> exit with result/exitCode
```

### 2. `core.setOutput()` flow

```go
func SetOutput(name string, value any) error {
    encoded := normalizeOutputValue(value)
    path := cfg.OutputFile // usually GITHUB_OUTPUT
    return runnerfiles.AppendKeyValue(path, name, encoded)
}
```

```text
JS script
  -> @actions/core.setOutput("allowed_actions", "selected")
  -> Go module validates name/value
  -> append "allowed_actions=selected" to output file
  -> subsequent workflow steps read steps.<id>.outputs.allowed_actions
```

### 3. `github.getOctokit()` flow

```go
func GetOctokit(token string, opts Options) *JSClient {
    client := githubapi.NewClient(token, opts)
    return WrapForJS(client)
}
```

```text
JS script
  -> const api = gha.getOctokit(token, { retries: 2 })
  -> api.request("GET /repos/{owner}/{repo}/actions/permissions", params)
  -> Go client builds HTTP request
  -> response decoded into Go struct
  -> converted to plain JS object
```

### 4. Async exec flow

```go
func Exec(command string, args []string, opts ExecOptions) goja.Value {
    promise, resolve, reject := vm.NewPromise()

    go func() {
        result, err := runCommand(command, args, opts)
        _ = owner.Post(context.Background(), "exec.settle", func(context.Context, *goja.Runtime) {
            if err != nil {
                _ = reject(vm.ToValue(err.Error()))
                return
            }
            _ = resolve(vm.ToValue(result))
        })
    }()

    return vm.ToValue(promise)
}
```

## Detailed Implementation Plan

### Phase 0: Repository normalization

Files to create or change:

- `goja-github-actions/go.mod`
- `goja-github-actions/cmd/goja-gha/main.go`
- `goja-github-actions/cmd/goja-gha/cmds/root.go`
- `goja-github-actions/cmd/goja-gha/cmds/run.go`
- `goja-github-actions/cmd/goja-gha/cmds/doctor.go`
- `goja-github-actions/README.md`

Tasks:

1. rename the module to the real repository path,
2. create the `goja-gha` binary entrypoint,
3. wire a Glazed/Cobra root command and logging setup,
4. add `go-go-goja` and `glazed` as dependencies,
5. add a minimal README describing purpose and current scope.

Definition of done:

- `go test ./...` compiles,
- `go run ./cmd/goja-gha --help` works.

### Phase 1: Runtime bootstrap and `@actions/core`

Files:

- `cmd/goja-gha/cmds/run.go`
- `pkg/cli/defaults.go`
- `pkg/cli/middleware.go`
- `pkg/cli/sections/github_actions.go`
- `pkg/runtime/*`
- `pkg/runnerfiles/*`
- `pkg/modules/core/*`

Tasks:

1. define the `RunSettings` Glazed struct,
2. define the GitHub Actions schema section with `fields.New(...)`,
3. implement middleware-based default resolution for runner values,
4. build runtime factory with script-based module roots,
5. add `@actions/core`,
6. add `process.env` and script execution,
7. add example `examples/set-output.js`.

Definition of done:

- JS can read `core.getInput(...)`,
- JS can call `core.setOutput(...)`,
- tests verify `GITHUB_OUTPUT`, `GITHUB_ENV`, and `GITHUB_STEP_SUMMARY` semantics.

### Phase 2: `@actions/github` and policy-tool support

Files:

- `pkg/contextdata/*`
- `pkg/githubapi/*`
- `pkg/modules/github/*`
- `examples/permissions-audit.js`

Tasks:

1. load event JSON and build `github.context`,
2. implement authenticated REST client,
3. expose `getOctokit()` and `request()`,
4. add typed wrappers for imported-plan endpoints,
5. add example policy-audit script.

Definition of done:

- a JS script can call the Actions permissions endpoints from the imported plan,
- `github.context.repo` and `github.context.payload` are populated from fixture data,
- `httptest.Server` integration tests cover success and failure cases.

### Phase 3: `@actions/io`, `@actions/exec`, and workflow-file inspection

Files:

- `pkg/modules/io/*`
- `pkg/modules/exec/*`
- `examples/list-workflows.js`

Tasks:

1. implement filesystem helpers,
2. implement async exec with owner-thread settlement,
3. add workflow directory scanning examples,
4. support script-local helper modules under `examples/lib` or `scripts/lib`.

Definition of done:

- a script can inspect `.github/workflows`,
- async exec works without race conditions or VM-thread violations.

### Phase 4: Packaging and wrapper action

Files:

- `action.yml` or wrapper action files,
- release/build config,
- docs for local vs CI execution.

Tasks:

1. decide composite vs container wrapper,
2. add release artifacts,
3. add end-to-end smoke workflow.

Definition of done:

- the binary can run inside GitHub Actions using a supported wrapper model.

## Testing and Validation Strategy

### Unit tests

- runner file writers: encoding, multiline values, path append behavior
- Glazed settings resolution: field decoding, middleware precedence, required fields
- context loading: payload parse and repo/ref extraction
- GitHub client: request building, header injection, pagination

### Runtime tests

Model them after `go-go-goja/engine/factory_test.go`, `go-go-goja/engine/runtime_test.go`, and `go-go-goja/engine/module_roots_test.go`.

Add tests that:

- run a JS file through the real CLI/runtime,
- verify `require("@actions/core")` works,
- verify local helper modules can be required relative to the script directory,
- verify promise-based APIs settle correctly.

### Integration tests

- `httptest.Server` for GitHub REST endpoints,
- temp workspace with `.github/workflows/*.yml`,
- fake runner command files plus middleware-populated schema defaults,
- golden output files for `GITHUB_OUTPUT`, `GITHUB_ENV`, and summaries.

### Smoke scripts

- `examples/set-output.js`
- `examples/permissions-audit.js`
- `examples/list-workflows.js`

Each example should be runnable both locally and in tests.

## Risks, Alternatives, and Open Questions

### Risks

- Goja is not Node.js. Some engineers will assume npm packages and full Node globals are available.
- Full `@actions/github` parity would be expensive if we over-commit early.
- Async module work can become flaky if any goroutine touches the VM outside `runtimeowner.Runner`.
- GitHub Actions contexts resolved at workflow-expression time cannot all be reconstructed at runtime.

### Alternatives considered

#### Alternative A: Wrap `gh` CLI from JS instead of building a GitHub client

Rejected for the MVP because:

- it adds another runtime dependency,
- it is less testable,
- it is less portable outside GitHub-hosted environments,
- the imported plan already maps naturally to REST endpoints.

#### Alternative B: Build a `github-script` clone that only injects globals, no `require("@actions/*")`

Rejected because:

- it feels less like the toolkit users already know,
- it makes code organization worse for larger scripts,
- native module names give a better long-term package boundary.

#### Alternative C: Implement as a custom JavaScript action runtime

Rejected because official action metadata only supports `node20` and `node24` for JavaScript actions. A Go binary must be wrapped as a composite or container action instead.

### Open questions

1. Should `process.env` local emulation reflect writes to `core.exportVariable()` immediately, or should it stay faithful to runner semantics and only write the file?
2. Do we want a small YAML helper module in phase 2 so workflow inspection stays fully in JS?
3. Is a container wrapper acceptable for the first GitHub-hosted release, or do we require cross-platform composite packaging immediately?
4. Should `@actions/github` expose a generated `rest` tree later, or stay intentionally small with `request()` as the primary interface?
5. Which Glazed middleware shape should own GitHub runner env resolution: a small app-local middleware in `pkg/cli/middleware.go`, or a reusable helper extracted later if other runner-oriented tools appear?

## Recommended First Milestone

The first milestone should be:

1. compile and run `goja-gha`,
2. support `@actions/core`,
3. support `@actions/github.getOctokit().request(...)`,
4. run a JS script that audits one repository's Actions permissions and writes a JSON summary to `GITHUB_OUTPUT`.

That milestone proves the entire stack:

- CLI,
- runtime,
- inputs,
- outputs,
- context,
- auth,
- API calls,
- JS authoring ergonomics.

## Alternatives Considered

See the dedicated alternatives section above.

## Implementation Plan

See the phased implementation plan above. An intern should begin with Phase 0 and Phase 1, not with GitHub API parity work.

## Open Questions

See the open-questions section above.

## References

### Local repository evidence

- `goja-github-actions/go.mod`
- `goja-github-actions/cmd/XXX/main.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/runtime.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/engine/module_roots.go`
- `go-go-goja/modules/common.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `goja-git/main.go`
- `goja-git/gitmodule.go`
- `goja-git/filterrepo/filterrepo.go`
- `sources/local/01-imported-planning-notes.md`

### External API references

- GitHub Actions metadata syntax reference: https://docs.github.com/en/actions/sharing-automations/creating-actions/metadata-syntax-for-github-actions
- Workflow commands and environment files: https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands
- Variables reference: https://docs.github.com/en/actions/reference/workflows-and-actions/variables
- Contexts reference: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/accessing-contextual-information-about-workflow-runs
- REST API endpoints for GitHub Actions permissions: https://docs.github.com/rest/actions/permissions
- `actions/toolkit`: https://github.com/actions/toolkit
- `actions/github-script`: https://github.com/actions/github-script
