---
Title: Glazed-native settings resolution design for goja-gha
Ticket: GHA-2
Status: closed
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/glazed/pkg/cli/cobra-parser.go
      Note: Built-in Cobra parser chain showing native env and config support
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/glazed/pkg/cmds/sources/update.go
      Note: FromEnv implementation and env key derivation rules
    - Path: pkg/cli/defaults.go
      Note: Manual GITHUB_* to field mapping targeted for removal
    - Path: pkg/cli/middleware.go
      Note: Current custom parser middleware chain that bypasses Glazed env parsing
    - Path: pkg/contextdata/github_context.go
      Note: Current github.context population from synthetic env values
    - Path: pkg/modules/github/module.go
      Note: Current API URL fallback path through ProcessEnv
    - Path: pkg/runtime/globals.go
      Note: Current ProcessEnv implementation that synthesizes runtime GitHub env
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T08:05:59.831543373-04:00
WhatFor: ""
WhenToUse: ""
---


# Glazed-native settings resolution design for goja-gha

## Executive Summary

`goja-gha` currently resolves command settings in two different ways:

1. Glazed decodes flags and files into typed settings structs.
2. `goja-gha` then bypasses that model by manually reading `GITHUB_*` environment variables and re-synthesizing a GitHub-flavored environment through `ProcessEnv()`.

That split makes the command harder to reason about. A reader has to understand both Glazed's parse pipeline and `goja-gha`'s hand-written overlay code before they can answer a simple question like "where did `workspace` come from?" or "why did `github.context.repository` get this value?".

The direction of this ticket is to move all command-input resolution into Glazed. The CLI should accept values from Glazed-supported sources only: defaults, config files, environment variables under the application prefix, positional arguments, and flags. In concrete terms, `GOJA_GHA_GITHUB_TOKEN`, `GOJA_GHA_WORKSPACE`, `GOJA_GHA_EVENT_PATH`, and similar names should be parsed by Glazed middleware rather than by `os.LookupEnv` calls in application code.

This design guide recommends three structural changes:

- remove manual command-input environment mapping from [`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go),
- replace implicit context-building from runtime env in [`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go) with explicit decoded settings,
- split "command inputs" from "runtime-visible process environment" so JavaScript still gets a `process.env`, but that object is derived from decoded state, not by re-reading the host process environment ad hoc.

The end state should be easy to explain to a new intern:

- Glazed owns input resolution.
- `RunnerSettings` and `GitHubActionsSettings` are the single typed source of truth.
- Runtime components consume those settings directly.
- JavaScript-facing environment data is built from an explicit model, not inferred from the machine the command happens to run on.

## Problem Statement

### What the user asked for

The requested direction is explicit:

- do not look up settings from the environment ourselves,
- rely on Glazed middleware to parse `GOJA_GHA_*`,
- remove `ProcessEnv`-driven GitHub context population,
- produce a detailed implementation guide that a new intern can execute safely.

### Why the current design is a problem

The current implementation has two distinct categories of environment handling that are coupled together:

1. **Command parsing concerns**
   - Which values should the CLI accept?
   - What is the precedence between defaults, config, env, args, and flags?
   - How do values get decoded into `RunnerSettings` and `GitHubActionsSettings`?

2. **Runtime behavior concerns**
   - What should `process.env` contain for JavaScript code?
   - How should `@actions/core.exportVariable()` mutate state?
   - What should `@actions/github.context` expose?

Right now those concerns bleed into each other.

### Evidence from the current codebase

#### 1. `goja-gha` overrides Glazed's normal env/config parser

[`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go) builds a custom middleware chain:

- `FromCobra`
- `FromArgs`
- `FromFiles`
- `FromMap(RunnerEnvValuesFromLookup(...))`
- `FromMapAsDefault(DefaultFieldValues())`
- `FromDefaults`

That means command inputs are not coming from Glazed's standard `FromEnv("GOJA_GHA")` mechanism. They are coming from a custom map built by `RunnerEnvValuesFromLookup`.

[`pkg/cli/defaults.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/defaults.go) hardcodes mappings like:

- `GITHUB_EVENT_PATH -> event-path`
- `GITHUB_WORKSPACE -> github-actions.workspace`
- `GITHUB_TOKEN -> github-actions.github-token`

This duplicates functionality Glazed already provides through environment parsing and makes the CLI's behavior special-case-heavy.

#### 2. Runtime code re-synthesizes a GitHub environment

[`pkg/runtime/globals.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go) defines `Settings.ProcessEnv()`. That method:

- starts with `AmbientEnvironment`,
- injects `GITHUB_WORKSPACE`,
- injects `GITHUB_EVENT_PATH`,
- injects runner command file paths,
- injects `GITHUB_TOKEN`.

This is a second resolution layer. It does not just expose already-decoded state. It actively constructs a new environment map with GitHub semantics.

#### 3. GitHub context reads from that synthetic environment

[`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go) uses:

```go
env := settings.ProcessEnv()
```

Then it derives:

- `repository` from `GITHUB_REPOSITORY` or payload,
- `actor` from `GITHUB_ACTOR`,
- `event_name` from `GITHUB_EVENT_NAME`,
- `ref` from `GITHUB_REF`,
- `sha` from `GITHUB_SHA`.

So `github.context` is not built from decoded command settings. It is built from an environment bag whose contents depend on prior synthetic logic.

#### 4. Multiple modules depend on the same ambient/synthetic environment

- [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go) reads `GITHUB_API_URL` through `settings.ProcessEnv()`.
- [`pkg/modules/exec/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go) seeds subprocess env from `settings.ProcessEnv()`.
- [`pkg/modules/core/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/core/module.go) initializes runtime state env from `settings.ProcessEnv()`.

This makes `ProcessEnv()` a hidden dependency injection mechanism across the whole runtime.

### Why Glazed already solves most of the command-input problem

The local Glazed docs and source already describe the intended model.

[`glazed/pkg/doc/tutorials/05-build-first-command.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md) emphasizes:

- define fields in the schema,
- let the parser resolve values,
- decode into a struct,
- do not read Cobra flags directly.

[`glazed/pkg/doc/topics/24-config-files.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/topics/24-config-files.md) documents precedence as:

- defaults,
- config files,
- env,
- positional args,
- flags.

[`glazed/pkg/cli/cobra-parser.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cobra-parser.go) shows that if `AppName` is set and no custom middleware function is supplied, Glazed already builds a chain that includes:

- `FromCobra`
- `FromArgs`
- `FromEnv(strings.ToUpper(AppName))`
- resolved config files
- `FromDefaults`

[`glazed/pkg/cmds/sources/update.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/sources/update.go) shows the environment variable key derivation algorithm:

- field name = `section prefix + field name`,
- hyphens become underscores,
- uppercased,
- optional global prefix prepended.

This means `goja-gha` should not need custom `os.LookupEnv` logic for command settings at all.

## Proposed Solution

### Design goal

Move all **command input resolution** into Glazed, while preserving a clean and explicit model for **runtime-visible environment data**.

The key architectural decision is:

> Command parsing and runtime environment exposure are different problems and should be represented by different data structures.

### High-level architecture

```text
                 ┌───────────────────────────────────────┐
                 │ Host process environment              │
                 │ GOJA_GHA_*                            │
                 │ config files                          │
                 │ CLI flags / args                      │
                 └───────────────────┬───────────────────┘
                                     │
                                     ▼
                    ┌────────────────────────────────┐
                    │ Glazed parser pipeline         │
                    │ defaults < config < env <     │
                    │ args < flags                  │
                    └────────────────┬──────────────┘
                                     │
                                     ▼
                    ┌────────────────────────────────┐
                    │ Decoded settings structs       │
                    │ RunnerSettings                 │
                    │ GitHubActionsSettings          │
                    │ GitHubContextSettings          │
                    └────────────────┬──────────────┘
                                     │
                     ┌───────────────┼────────────────┐
                     │               │                │
                     ▼               ▼                ▼
          ┌────────────────┐ ┌───────────────┐ ┌────────────────┐
          │ runtime.Settings│ │github.context │ │process.env seed│
          │ explicit fields │ │explicit model │ │explicit model  │
          └────────────────┘ └───────────────┘ └────────────────┘
```

### Proposed settings model

The runtime should receive explicit settings for all values it actually needs. A minimal target shape looks like this:

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
    GitHubToken string `glazed:"github-token"`
}

type GitHubContextSettings struct {
    Repository string `glazed:"repository"`
    Actor      string `glazed:"actor"`
    EventName  string `glazed:"event-name"`
    Ref        string `glazed:"ref"`
    SHA        string `glazed:"sha"`
    APIURL     string `glazed:"api-url"`
}
```

This guide does **not** require these exact names, but it strongly recommends the following rule:

- every value that influences runtime behavior should have an explicit field,
- if a value can come from env/config/flags, it should be a schema field,
- if a value is not meant to be configured, it should not be sourced from ambient env.

### Recommended section layout

The user already clarified a useful schema boundary:

- the shared `github-actions` section should contain only values common to commands interacting with GitHub,
- runner-specific inputs belong in the default section.

That means:

#### Default section

Use for command/run-specific values:

- `script`
- `cwd`
- `event-path`
- `action-path`
- `runner-env-file`
- `runner-output-file`
- `runner-path-file`
- `runner-summary-file`
- `debug`
- `json-result`
- possibly context fields like `repository`, `actor`, `event-name`, `ref`, `sha`, `api-url` if they are primarily "runner invocation inputs"

#### `github-actions` section

Use only for shared cross-command GitHub parameters:

- `workspace`
- `github-token`

This keeps the section semantics narrow and matches the user's stated preference.

### Environment variable naming

Glazed derives environment keys from:

- global prefix from `AppName`,
- optional section prefix,
- field name.

Current `AppName` is `goja-gha`, so the built-in global env prefix becomes `GOJA_GHA`.

With the current unprefixed `github-actions` section, these names would resolve cleanly:

- `GOJA_GHA_WORKSPACE`
- `GOJA_GHA_GITHUB_TOKEN`
- `GOJA_GHA_EVENT_PATH`
- `GOJA_GHA_RUNNER_OUTPUT_FILE`
- `GOJA_GHA_DEBUG`

That naming is good and matches the user's expectation for `GOJA_GHA_GITHUB_TOKEN`.

### Parser recommendation

Replace custom middleware wiring with Glazed-native parsing.

#### Current parser setup

[`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go) currently sets:

```go
glazedcli.CobraParserConfig{
    AppName:         AppName,
    MiddlewaresFunc: NewMiddlewaresFunc(os.LookupEnv),
}
```

#### Recommended parser setup

Prefer:

```go
glazedcli.CobraParserConfig{
    AppName:           AppName,
    ConfigFilesFunc:   ResolveConfigFiles,
    ShortHelpSections: []string{},
}
```

That keeps app-level config file discovery while handing env parsing back to Glazed.

### Runtime recommendation

Split the current `ProcessEnv()` responsibility into two narrower responsibilities:

#### 1. `DecodedSettings`

This is just the set of typed settings Glazed produced. It is the source of truth for command inputs.

#### 2. `RuntimeEnvironment`

This is the environment map actually shown to JavaScript and subprocesses.

It should be constructed from:

- explicit decoded settings,
- explicit internal runtime state,
- optional inherited ambient env only where inheritance is a deliberate runtime feature.

The runtime builder should not call `os.LookupEnv` for command setting resolution. If an env value matters, Glazed should already have decoded it into a field.

### Proposed runtime API shape

One viable refactor is:

```go
type RuntimeSettings struct {
    ScriptPath       string
    WorkingDirectory string
    Workspace        string
    EventPath        string
    ActionPath       string
    RunnerEnvFile    string
    RunnerOutputFile string
    RunnerPathFile   string
    RunnerSummaryFile string
    GitHubToken      string
    GitHubRepository string
    GitHubActor      string
    GitHubEventName  string
    GitHubRef        string
    GitHubSHA        string
    GitHubAPIURL     string
    Debug            bool
    JSONResult       bool
    InheritedEnv     map[string]string
    State            *State
}

type State struct {
    Environment    map[string]string
    ExitCode       int
    FailureMessage string
}
```

Then create a dedicated constructor:

```go
func BuildInitialRuntimeEnvironment(s *RuntimeSettings) map[string]string {
    env := clone(s.InheritedEnv)

    setIfNonEmpty(env, "GITHUB_WORKSPACE", s.Workspace)
    setIfNonEmpty(env, "GITHUB_EVENT_PATH", s.EventPath)
    setIfNonEmpty(env, "GITHUB_ACTION_PATH", s.ActionPath)
    setIfNonEmpty(env, "GITHUB_ENV", s.RunnerEnvFile)
    setIfNonEmpty(env, "GITHUB_OUTPUT", s.RunnerOutputFile)
    setIfNonEmpty(env, "GITHUB_PATH", s.RunnerPathFile)
    setIfNonEmpty(env, "GITHUB_STEP_SUMMARY", s.RunnerSummaryFile)
    setIfNonEmpty(env, "GITHUB_TOKEN", s.GitHubToken)
    setIfNonEmpty(env, "GITHUB_REPOSITORY", s.GitHubRepository)
    setIfNonEmpty(env, "GITHUB_ACTOR", s.GitHubActor)
    setIfNonEmpty(env, "GITHUB_EVENT_NAME", s.GitHubEventName)
    setIfNonEmpty(env, "GITHUB_REF", s.GitHubRef)
    setIfNonEmpty(env, "GITHUB_SHA", s.GitHubSHA)
    setIfNonEmpty(env, "GITHUB_API_URL", s.GitHubAPIURL)

    return env
}
```

The important part is not the helper name. The important part is that this is **not** a command parser. It is a runtime projection from already-decoded settings into a JS-facing environment bag.

### GitHub context recommendation

[`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go) should stop reading from `settings.ProcessEnv()`.

Instead it should build context from:

- explicit runtime fields,
- the event payload file,
- payload fallback only where GitHub actually allows absence of those fields.

Recommended logic:

```go
func BuildGitHubContext(settings *RuntimeSettings) (*GitHubContext, error) {
    payload, err := LoadEventPayload(settings.EventPath)
    if err != nil {
        return nil, err
    }

    repository := firstNonEmpty(
        settings.GitHubRepository,
        nestedString(payload, "repository", "full_name"),
    )

    owner, repo := splitRepository(repository)
    if owner == "" {
        owner = firstNonEmpty(
            nestedString(payload, "repository", "owner", "login"),
            nestedString(payload, "repository", "owner", "name"),
        )
    }
    if repo == "" {
        repo = nestedString(payload, "repository", "name")
    }

    return &GitHubContext{
        Actor:      settings.GitHubActor,
        EventName:  firstNonEmpty(settings.GitHubEventName, nestedString(payload, "action")),
        EventPath:  settings.EventPath,
        Ref:        settings.GitHubRef,
        Repository: repository,
        SHA:        settings.GitHubSHA,
        Workspace:  settings.Workspace,
        Repo: RepoContext{Owner: owner, Repo: repo},
        Payload: payload,
    }, nil
}
```

This makes the data flow inspectable.

### What should happen to `AmbientEnvironment`

There are two reasonable options:

#### Option A: keep inherited ambient environment

Use host env as a base for subprocess behavior and `process.env`, but not as a source of command settings.

Pros:

- preserves normal CLI behavior for things like `PATH`, `HOME`, `TMPDIR`,
- avoids surprising subprocess breakage,
- still satisfies the user's main request because settings are not looked up ad hoc anymore.

Cons:

- `process.env` still contains many host values not modeled in schema,
- some readers may still confuse inherited env with command inputs.

#### Option B: minimize inherited ambient environment

Start from an empty map and add only the variables that the runtime deliberately wants to expose.

Pros:

- maximal determinism,
- easiest mental model,
- strongest separation between command input parsing and runtime environment.

Cons:

- high compatibility risk for subprocesses,
- likely breaks `PATH`-based execution unless reconstructed explicitly.

### Recommended choice

For this ticket, choose **Option A**:

- stop using host env for settings resolution,
- keep inherited env as the base runtime env for now,
- document clearly that inherited env is a runtime convenience, not a parse source.

That is the smallest change that respects the user's requirement and does not destabilize `@actions/exec`.

## Intern Walkthrough

### Decision 1: Glazed owns command-input precedence

Use Glazed's built-in parse chain rather than custom `os.LookupEnv` mapping.

Rationale:

- It matches Glazed's documented model.
- It keeps `--print-parsed-fields` meaningful.
- It removes duplicated precedence logic from application code.

### Decision 2: Add explicit fields for currently ambient GitHub context values

If `github.context` needs `repository`, `actor`, `event name`, `ref`, `sha`, or `api url`, those values should be explicit settings, not hidden env reads.

Rationale:

- A field can be validated, documented, and tested.
- A hidden env read cannot.

### Decision 3: Keep `github-actions` section narrow

The section should continue to carry only:

- `workspace`
- `github-token`

Rationale:

- This was explicitly requested by the user.
- It keeps the section semantically stable for future commands.

### Decision 4: Keep runtime env mutation, but reframe it as runtime state, not settings resolution

`@actions/core.exportVariable()` and `@actions/core.addPath()` should still mutate runtime state and `process.env`.

Rationale:

- That matches user expectations from GitHub Actions JavaScript.
- Those mutations are runtime side effects, not input parsing.

### Decision 5: Avoid backwards-compatibility shims for raw `GITHUB_*` parse inputs unless intentionally documented

The cleanest outcome is:

- parse from `GOJA_GHA_*`,
- expose `GITHUB_*` inside the JS runtime because that is the GitHub Actions contract.

Rationale:

- the user explicitly asked for Glazed-native parsing,
- mixing both naming schemes for CLI input will keep the code muddy,
- tests and docs should teach one clear external contract.

If a temporary migration shim is needed, it should be documented as transitional and isolated.

## Design Decisions

### Data-flow diagram

```text
Before
------

host env ──► custom lookupEnv() ──► parsed values
   │
   └──────► ambient env ──► ProcessEnv() ──► github.context / process.env / exec env


After
-----

host env (GOJA_GHA_*) ──► Glazed FromEnv("GOJA_GHA") ──► parsed values ──► decoded settings
                                                                           │
                                                                           ├──► github.context builder
                                                                           ├──► runtime env seed
                                                                           └──► module dependencies
```

### New mental model for interns

When debugging, ask these questions in order:

1. Is the value defined as a Glazed field?
2. Which source won the precedence race?
3. Did `DecodeSettings()` populate the expected struct field?
4. Did runtime initialization copy that explicit field into runtime state?
5. Did JavaScript read from `github.context` or `process.env`?

That five-step model is much easier than "maybe it came from Glazed, maybe from a manual env mapping, maybe from `ProcessEnv()`, maybe from the event payload".

## Alternatives Considered

### Alternative A: keep current custom env mapping and just document it better

Rejected.

Why:

- It contradicts the requested direction.
- It duplicates parser behavior Glazed already provides.
- It preserves hidden precedence logic.

### Alternative B: parse both `GOJA_GHA_*` and raw `GITHUB_*` indefinitely

Rejected as the main plan.

Why:

- It creates two public input contracts.
- It encourages new code to keep depending on GitHub runner env as parse input.
- It makes debugging more ambiguous.

Possible transitional use:

- acceptable only as a short-lived migration layer,
- should live in one place,
- should be removed after docs and tests are updated.

### Alternative C: remove inherited ambient env entirely

Rejected for the initial refactor.

Why:

- likely to break subprocess expectations,
- unrelated to the user's core complaint,
- can be revisited later as a hardening step.

### Alternative D: continue deriving `github.context` from env because GitHub Actions itself uses env heavily

Rejected.

Why:

- the point of this CLI is to create a predictable local/runtime model,
- env-driven context derivation is exactly the hidden behavior the user asked to eliminate,
- payload fallback gives a cleaner and more explicit source hierarchy.

## Implementation Plan

### Phase 1: remove custom Glazed parser bypass

Files:

- [`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go)
- [`pkg/cli/defaults.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/defaults.go)

Changes:

- delete `RunnerEnvValuesFromLookup`,
- delete raw `GITHUB_*` field mapping logic,
- keep config discovery through `ResolveConfigFiles`,
- let `CobraParserConfig{AppName: AppName, ConfigFilesFunc: ResolveConfigFiles}` activate built-in env handling.

Pseudocode:

```go
func NewParserConfig() glazedcli.CobraParserConfig {
    return glazedcli.CobraParserConfig{
        AppName:           AppName,
        ConfigFilesFunc:   ResolveConfigFiles,
        ShortHelpSections: []string{},
    }
}
```

Expected effect:

- command inputs come from config/env/args/flags through Glazed only,
- `GOJA_GHA_*` becomes the documented env contract.

### Phase 2: model missing runtime inputs explicitly

Files:

- [`pkg/cli/github_actions.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/github_actions.go)
- [`pkg/cli/settings.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/settings.go)

Changes:

- add schema fields for values currently read from ambient/synthetic env:
  - `repository`
  - `actor`
  - `event-name`
  - `ref`
  - `sha`
  - `api-url`
- decide whether they live in the default section or a dedicated context section.

Recommendation:

- keep them in the default section for now because they are runner/invocation values, not shared cross-command GitHub credentials.

Validation additions:

- extend `ValidateRunSettings` if some of these become required for API-backed examples.

### Phase 3: replace `ProcessEnv()` as a settings source

Files:

- [`pkg/runtime/factory.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go)
- [`pkg/runtime/globals.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go)

Changes:

- replace `ProcessEnv()` with a function whose role is clearly runtime-specific, for example:
  - `InitialRuntimeEnv()`
  - `BuildProcessEnvironment()`
- initialize `State.Environment` from that explicit builder once,
- stop using that builder as an implicit source of missing settings.

Important code smell to remove:

```go
settings.State = &State{
    Environment: settings.ProcessEnv(),
}
```

The replacement should still initialize state, but with a clearer contract name.

### Phase 4: rebuild GitHub context from explicit settings

Files:

- [`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go)
- [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go)

Changes:

- replace `settings.ProcessEnv()` reads with explicit fields on `runtime.Settings`,
- keep payload fallback only where explicit fields are empty,
- change GitHub API base URL resolution to:
  - explicit option argument,
  - explicit runtime setting,
  - hardcoded default `https://api.github.com`.

Pseudocode:

```go
baseURL := firstNonEmpty(
    options.BaseURL,
    m.deps.Settings.GitHubAPIURL,
    "https://api.github.com",
)
```

### Phase 5: keep `process.env` and subprocess env working

Files:

- [`pkg/modules/core/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/core/module.go)
- [`pkg/modules/exec/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go)

Changes:

- state initialization should use the new runtime env builder,
- `exportVariable` and `addPath` should mutate `State.Environment`,
- `@actions/exec` should use `State.Environment` as subprocess base env.

Behavioral contract:

- CLI input parsing is Glazed-owned,
- runtime mutation is state-owned,
- subprocesses observe current runtime state.

### Phase 6: update tests to the new public contract

Files:

- [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go)
- module tests under [`pkg/modules`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules)

Changes:

- replace raw input env usage like `GITHUB_TOKEN=...` for CLI parsing with `GOJA_GHA_GITHUB_TOKEN=...`,
- replace `GITHUB_WORKSPACE` input for parsing with `GOJA_GHA_WORKSPACE=...`,
- add assertions that decoded settings still result in correct JS-visible `GITHUB_*` values where required.

Recommended test matrix:

- config file only,
- env only via `GOJA_GHA_*`,
- flags override env,
- event payload fallback when explicit GitHub context fields are absent,
- `@actions/exec` inherits runtime state changes,
- `@actions/core.exportVariable()` updates both file output and JS `process.env`.

### Phase 7: update docs

Files:

- [`README.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/README.md)
- Glazed help docs under [`pkg/helpdoc`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc)

Changes:

- document `GOJA_GHA_*` as the input contract,
- document that JS code still sees GitHub-style variables in `process.env`,
- explain the difference between input parsing and runtime exposure.

### Implementation checklist

- [ ] Remove custom `MiddlewaresFunc` env mapping.
- [ ] Keep config discovery through `ConfigFilesFunc`.
- [ ] Add explicit fields for GitHub context values still sourced from env.
- [ ] Rename or remove `ProcessEnv()` to clarify its narrower role.
- [ ] Refactor `BuildGitHubContext`.
- [ ] Refactor `@actions/github` base URL resolution.
- [ ] Refactor `@actions/exec` env seed path.
- [ ] Update integration and module tests to `GOJA_GHA_*`.
- [ ] Update user/developer documentation.

## Open Questions

### 1. Which GitHub context fields belong in schema v1?

The smallest correct set appears to be:

- `repository`
- `actor`
- `event-name`
- `ref`
- `sha`
- `api-url`

But the exact minimum should be validated against current examples and intended future commands.

### 2. Should `api-url` be user-facing or test-only?

It is useful for tests and GitHub Enterprise Server support. If it remains user-facing, docs should explain when to use it.

### 3. Should raw `GITHUB_*` parse compatibility exist for one release?

This is a migration policy decision, not a technical necessity.

### 4. How aggressively should ambient env inheritance be trimmed?

This ticket should likely leave inheritance mostly intact for subprocess compatibility, but that should be called out as a follow-up hardening topic.

## References

### Primary code references

- [`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go)
- [`pkg/cli/defaults.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/defaults.go)
- [`pkg/cli/github_actions.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/github_actions.go)
- [`pkg/cli/settings.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/settings.go)
- [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [`pkg/runtime/factory.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go)
- [`pkg/runtime/globals.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go)
- [`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go)
- [`pkg/modules/core/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/core/module.go)
- [`pkg/modules/exec/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go)
- [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go)
- [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go)

### Glazed references

- [`glazed/pkg/doc/tutorials/05-build-first-command.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md)
- [`glazed/pkg/doc/topics/24-config-files.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/topics/24-config-files.md)
- [`glazed/pkg/doc/topics/21-cmds-middlewares.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/topics/21-cmds-middlewares.md)
- [`glazed/pkg/doc/tutorials/migrating-from-viper-to-config-files.md`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/migrating-from-viper-to-config-files.md)
- [`glazed/pkg/cli/cobra-parser.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cobra-parser.go)
- [`glazed/pkg/cmds/sources/update.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/sources/update.go)
- [`glazed/pkg/appconfig/options.go`](/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/appconfig/options.go)

## References

<!-- Link to related documents, RFCs, or external resources -->
