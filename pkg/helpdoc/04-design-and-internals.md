---
Title: "Design And Internals"
Slug: "design-and-internals"
Short: "Deep architecture notes for internals work: why the system is shaped the way it is, what invariants matter, and where to extend it safely."
Topics:
- goja
- github-actions
- javascript
Commands:
- run
- doctor
Flags:
- github-token
- event-path
- runner-output-file
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This document is the deep internals guide. It is for someone who needs to change the runtime model, extend the module surface, or review whether the code still matches the architecture described in the ticket design packet.

The main theme is that `goja-gha` is built around explicit boundaries. Each boundary exists because a simpler design would be easier to write once and harder to maintain forever. The internals only make sense if you understand which shortcuts the project is deliberately refusing to take.

## Why The Architecture Uses `go-go-goja`

This section explains the foundational choice. The repository builds on `go-go-goja` because the hard part of a Goja-based tool is not `vm.RunString(...)`; it is lifecycle, module registration, require roots, and safe asynchronous interaction with the VM.

What `go-go-goja` gives this project:

- explicit factory building,
- static module registration,
- runtime initializers,
- script-relative module roots,
- a runtime owner abstraction with `Call(...)` and `Post(...)`,
- an event loop already integrated into runtime creation.

That is why `pkg/runtime/factory.go` is thin. The repo intentionally reuses the upstream builder instead of creating another ad hoc runtime stack.

## The Real Boundaries

This section names the architectural boundaries directly. If you keep these straight, the package layout stops feeling accidental.

### Boundary 1: Decoding Versus Execution

`pkg/cli` owns decoding. `pkg/runtime` owns execution.

Why the split matters:

- decoding is about precedence and observability,
- execution is about runtime behavior,
- mixing them creates bugs that `doctor` cannot explain.

### Boundary 2: Runtime Versus GitHub Semantics

`pkg/runtime` should be mostly ignorant of GitHub. It knows about:

- script paths,
- working directories,
- process globals,
- module registration,
- Promise awaiting.

It should not know about:

- repository permissions endpoints,
- GitHub-specific HTTP headers beyond what modules ask for,
- workflow audit business logic.

### Boundary 3: Go Domain Shapes Versus JS API Shapes

Internal Go structs can be whatever is convenient. JS-facing objects cannot. JS-facing objects need:

- lower-case keys,
- predictable nested objects,
- plain arrays and maps,
- understandable thrown errors.

This is why modules often normalize Go results before exposing them.

## Internals Of A `run` Invocation

This section gives a more detailed version of the user-facing flow. Read it when you are trying to place a bug or reason about where a new feature belongs.

```text
CLI layer
  parse cobra flags
  apply Glazed middlewares
  decode RunnerSettings + GitHubActionsSettings
  validate required combinations

Runtime settings layer
  create runtime.Settings
  snapshot ambient env
  initialize mutable runtime state

Factory layer
  derive module roots from script path
  register module specs
  install runtime initializers

Initializer layer
  register runtime bindings
  install process global

Execution layer
  require(entry module)
  discover exported function/main/default
  call entrypoint
  await Promise if returned

Result layer
  return JS value
  surface script failure / exitCode
  optionally encode JSON result
```

File references:

- `cmd/goja-gha/cmds/run.go`
- `pkg/runtime/factory.go`
- `pkg/runtime/bindings.go`
- `pkg/runtime/globals.go`
- `pkg/runtime/script_runner.go`

## Why `doctor` Exists

This section explains the design purpose of `doctor`, because it is more than a convenience command. It is an architectural pressure valve.

Without `doctor`, debugging precedence bugs would require:

- reading Go code,
- adding temporary logs,
- or writing custom probe scripts.

`doctor` exists so a user or maintainer can answer questions like:

- which workspace won,
- whether a token resolved,
- whether the event path exists,
- whether runner command-file paths were populated,
- whether validation errors are about missing settings or later runtime behavior.

That is why new precedence-sensitive features should usually appear in `doctor` output too.

## How Promise Awaiting Works

This section is important because it explains why `async` entrypoints work even though Goja returns a `Promise` object immediately.

`pkg/runtime/script_runner.go` does not assume the script result is final. It inspects the returned value, and if the value exports as `*goja.Promise`, it polls the Promise state through the runtime owner until it is fulfilled or rejected.

Pseudocode:

```text
value = call entrypoint
if value is not a Promise:
  return value

loop until timeout:
  inspect promise state on owner thread
  if fulfilled:
    return promise result
  if rejected:
    return error
  sleep briefly
```

Why this matters:

- `@actions/exec` would be much less useful if entrypoints could not `await` it,
- example scripts would need awkward workarounds,
- the CLI would otherwise emit raw unresolved Promise objects.

## Why `@actions/exec` Needs Runtime Bindings

This section explains the reason for `pkg/runtime/bindings.go`. It exists so native modules can discover runtime-scoped resources without turning those resources into public JS globals.

`@actions/exec` needs the runtime owner to settle Promises safely. The loader only receives `vm` and `moduleObj`, so the runtime stores a Go-side binding keyed by the runtime instance. The module can then ask `pkg/runtime.LookupBindings(vm)` for:

- the runtime owner,
- the current settings.

That is a deliberate tradeoff:

- better than exposing internal runtime handles to JavaScript,
- simpler than threading owner references through unrelated layers,
- explicit enough that module authors can reason about it.

## GitHub API Design Choices

This section explains why the GitHub module is split between `pkg/contextdata`, `pkg/githubapi`, and `pkg/modules/github`.

### `pkg/contextdata`

Owns shaping the runtime view of GitHub context:

- repo owner/repo name extraction,
- event payload loading,
- workspace/event-path convenience fields.

### `pkg/githubapi`

Owns transport and HTTP mechanics:

- route parsing,
- path interpolation,
- query/body building,
- response decoding,
- API error formatting.

### `pkg/modules/github`

Owns the JS module contract:

- `github.context`,
- `getOctokit(...)`,
- generic request helpers,
- curated `rest.actions.*` helpers.

Why this matters: when a GitHub API bug appears, you want to ask "is this a transport bug or a JS surface bug?" and then have separate packages to inspect.

## How To Extend The System Safely

This section is the design-oriented extension guide. Use it before adding any large feature.

### Add A New Runner Concept

If the new concept is a decoded input:

1. add the field in `pkg/cli/github_actions.go`,
2. map env/default behavior in `pkg/cli/defaults.go`,
3. confirm precedence in `pkg/cli/middleware_test.go`,
4. surface it in `doctor` if users need to inspect it.

### Add A New JS Module

1. define the JS contract,
2. create `pkg/modules/<name>`,
3. keep blocking work out of direct VM access paths,
4. add tests,
5. register it in `cmd/goja-gha/cmds/run.go`,
6. document it in help docs and examples if user-visible.

### Add More GitHub Endpoints

1. add or reuse transport logic in `pkg/githubapi`,
2. normalize returned shapes,
3. expose only the JS helpers you actually need,
4. cover them with fake-server tests before trusting live usage.

## Maintenance Guidance For Interns

This section is blunt on purpose. It names the habits that keep the repo healthy.

- Read the smallest relevant file chain before editing.
- Do not start in `examples` and guess what the runtime does.
- Do not read env vars directly from a module because "it is faster."
- Do not return raw Go structs to JS and assume property names will work out.
- Do not add asynchronous callbacks without thinking about owner-thread settlement.
- Do not trust a single green unit test when the feature is user-facing; add an example or integration test too.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A design change feels easy but cuts across many packages | You are probably crossing an architectural boundary | Stop and identify which layer actually owns the behavior |
| A Promise never settles | Background work completed but result was not posted back to the owner thread | Audit `Runner.Post(...)` usage |
| JS sees surprising property names | A Go struct leaked across the boundary | Normalize to a map before export |
| GitHub pagination behaves oddly | Link-header parsing or absolute-URL handling is wrong | Inspect `pkg/githubapi/request.go` and `pkg/modules/github/client.go` |
| A local run differs from CI | Precedence or runner-file setup differs | Compare `doctor` output and runner env/file inputs |

## See Also

- `goja-gha help user-guide`
- `goja-gha help javascript-api`
- `goja-gha help developer-guide`
