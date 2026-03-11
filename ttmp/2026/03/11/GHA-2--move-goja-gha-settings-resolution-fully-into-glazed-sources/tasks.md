# Tasks

## Milestones

- [ ] Replace manual parser env mapping with Glazed-native env/config handling.
- [ ] Make runtime settings explicit enough that `github.context` no longer depends on `ProcessEnv()`.
- [ ] Preserve JavaScript/runtime behavior while removing ad hoc settings resolution.
- [ ] Update tests, docs, and examples to the `GOJA_GHA_*` contract.

## Phase 1: Parser cleanup

- [ ] Remove `MiddlewaresFunc: NewMiddlewaresFunc(os.LookupEnv)` from [`pkg/cli/middleware.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/middleware.go).
- [ ] Keep `AppName: "goja-gha"` so Glazed env parsing remains enabled.
- [ ] Keep config discovery by wiring `ConfigFilesFunc: ResolveConfigFiles`.
- [ ] Delete `RunnerEnvValuesFromLookup`.
- [ ] Delete `normalizeEnvValue` if nothing else needs it.
- [ ] Delete `RunnerEnvMappings()` from [`pkg/cli/defaults.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/defaults.go).
- [ ] Decide whether `DefaultFieldValues()` should remain only for true app defaults like `cwd="."`.
- [ ] Verify `--print-parsed-fields` shows env source entries for `GOJA_GHA_*`.

## Phase 2: Explicit schema for GitHub runtime inputs

- [ ] Inventory every runtime value currently obtained from `settings.ProcessEnv()`.
- [ ] Add Glazed fields for required GitHub context values:
  - [ ] `repository`
  - [ ] `actor`
  - [ ] `event-name`
  - [ ] `ref`
  - [ ] `sha`
  - [ ] `api-url`
- [ ] Keep `workspace` and `github-token` in the `github-actions` section only.
- [ ] Keep runner-specific values in the default section.
- [ ] Update `DecodeSettings()` in [`pkg/cli/settings.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/cli/settings.go) to decode any new struct(s).
- [ ] Extend validation rules if any new fields become required for specific flows.

## Phase 3: Runtime settings refactor

- [ ] Refactor [`pkg/runtime/factory.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go) so runtime settings explicitly carry all required GitHub values.
- [ ] Rename or remove `ProcessEnv()` in [`pkg/runtime/globals.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/globals.go).
- [ ] Introduce a narrower helper for runtime env seeding, for example `BuildInitialRuntimeEnvironment()`.
- [ ] Initialize `State.Environment` from the new runtime env builder.
- [ ] Ensure the new builder consumes decoded settings instead of calling the host environment.
- [ ] Decide and document whether ambient inheritance remains the base env for subprocess compatibility.

## Phase 4: GitHub context refactor

- [ ] Refactor [`pkg/contextdata/github_context.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/contextdata/github_context.go) to stop reading `settings.ProcessEnv()`.
- [ ] Use explicit runtime settings for:
  - [ ] repository
  - [ ] actor
  - [ ] event name
  - [ ] ref
  - [ ] sha
- [ ] Keep payload fallback only where explicit values are absent.
- [ ] Add tests covering payload fallback behavior.

## Phase 5: Module adjustments

- [ ] Update [`pkg/modules/github/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go) to resolve API base URL from explicit settings instead of `ProcessEnv()`.
- [ ] Update [`pkg/modules/exec/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go) so subprocess env comes from runtime state.
- [ ] Update [`pkg/modules/core/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/core/module.go) so `exportVariable` and `addPath` mutate the new runtime env state cleanly.
- [ ] Confirm `process.env` remains synchronized with runtime state mutations.

## Phase 6: Tests

- [ ] Update integration tests in [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) to use `GOJA_GHA_*` parse env names.
- [ ] Add a parser-focused test that proves `GOJA_GHA_GITHUB_TOKEN` decodes into `GitHubActionsSettings.GitHubToken`.
- [ ] Add a parser-focused test that proves flags override `GOJA_GHA_*`.
- [ ] Add a parser-focused test that proves config files are lower precedence than env and flags.
- [ ] Add a runtime test that proves `process.env.GITHUB_TOKEN` still exists when the token was provided through Glazed settings.
- [ ] Add a context test that proves repository/actor/ref/sha come from explicit fields, not ambient env.
- [ ] Add a subprocess test that proves `@actions/exec` receives updated `PATH` after `addPath()`.

## Phase 7: Documentation and migration

- [ ] Update [`README.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/README.md) to document `GOJA_GHA_*`.
- [ ] Update Glazed help pages under [`pkg/helpdoc`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc) to explain the new input contract.
- [ ] Add a migration note if raw `GITHUB_*` parse inputs are removed.
- [ ] Add at least one concrete example command showing `GOJA_GHA_WORKSPACE` and `GOJA_GHA_GITHUB_TOKEN`.

## Review checklist

- [ ] There are no `os.LookupEnv` calls left for command setting resolution.
- [ ] `github.context` no longer reads from synthetic env maps for core fields.
- [ ] Runtime env mutation still works for `@actions/core`.
- [ ] `@actions/exec` still works with inherited process basics like `PATH`.
- [ ] `docmgr doctor --ticket GHA-2 --stale-after 30` passes cleanly.
