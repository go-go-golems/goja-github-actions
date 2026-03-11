# Tasks

## Research And Documentation

- [x] Create ticket workspace for Goja GitHub Actions bindings
- [x] Import `/tmp/goja-gha-plan.md` into the ticket sources
- [x] Inspect `go-go-goja`, `goja-git`, and `goja-github-actions`
- [x] Review current official GitHub Actions references relevant to the design
- [x] Write the detailed design and implementation guide for `goja-gha`
- [x] Update the design to use a Glazed schema-based command architecture for GitHub runner settings
- [x] Record the investigation diary entry
- [x] Relate key files and update changelog/bookkeeping
- [x] Run `docmgr doctor --ticket GHA-1 --stale-after 30`
- [x] Dry-run and complete the reMarkable bundle upload

## Milestone Plan

- [ ] Milestone 1: `goja-gha run <script>` starts, resolves Glazed schema fields, boots Goja, and executes a trivial script successfully
- [ ] Milestone 2: `@actions/core` can read inputs and write outputs/env/path/summary files in local fixture runs
- [ ] Milestone 3: `@actions/github` can call the Actions permissions endpoints needed by the permissions-audit example
- [ ] Milestone 4: `@actions/io` and `@actions/exec` support realistic workflow inspection and helper command execution
- [ ] Milestone 5: packaged binary or wrapper action can run the example script in CI

## Immediate Execution Order

- [x] Finish Phase 0 before starting any module work
- [ ] Finish Phase 1 before wiring runtime globals or context loading
- [ ] Finish Phase 2 and Phase 3 before implementing `@actions/core`
- [ ] Finish Phase 4 before starting `@actions/github`
- [ ] Finish Phase 5 before adding the permissions-audit example
- [ ] Finish example scripts before packaging work

## Phase 0: Repository Bootstrap

- [x] Rename `goja-github-actions/go.mod` from the placeholder module path to the real repo path
- [x] Run `go mod tidy` after the initial dependency and module-path changes
- [ ] Add `go-go-goja`, `glazed`, and Cobra dependencies needed for the first command
- [x] Decide whether command code implements `cmds.BareCommand` or a Glaze processor command for `run`
- [x] Create `cmd/goja-gha/main.go`
- [x] Create `cmd/goja-gha/cmds/root.go`
- [x] Create `cmd/goja-gha/cmds/run.go`
- [x] Create `cmd/goja-gha/cmds/doctor.go`
- [x] Register subcommands from `cmd/goja-gha/cmds/root.go`
- [x] Wire `cli.BuildCobraCommandFromCommand(...)` for each command
- [x] Add the standard Glazed output section and command settings section
- [x] Add logging section wiring and logger initialization in root `PersistentPreRunE`
- [x] Add help text with concrete examples to `run` and `doctor`
- [x] Verify `go run ./cmd/goja-gha --help` works
- [x] Verify `go run ./cmd/goja-gha run --help` works
- [x] Verify `go run ./cmd/goja-gha doctor --help` works
- [x] Replace the template README with a real project overview

## Phase 1: Glazed Schema And Input Resolution

- [x] Define bootstrap settings structs with `glazed:"..."` tags for runner inputs and shared GitHub inputs
- [x] Create `pkg/cli/github_actions.go` with default runner fields plus the shared GitHub section
- [x] Add `settings.NewGlazedSchema()` to `run` so output controls are present from day one
- [x] Add `cli.NewCommandSettingsSection()` so parsed-field/schema debugging is available immediately
- [ ] Create `pkg/cli/defaults.go` for hardcoded app defaults
- [ ] Create `pkg/cli/middleware.go` for field resolution precedence
- [ ] Decide how local config/profile support will be represented, even if phase 1 only implements runner-env defaults
- [ ] Implement precedence `flags > config/profile > runner env > hardcoded defaults`
- [ ] Map runner env names to Glazed fields explicitly in one place
- [ ] Add validation for required combinations such as `script` plus a valid token for GitHub API use cases
- [x] Ensure bootstrap command code consumes only decoded settings structs, not direct `os.Getenv` calls
- [x] Add `doctor` checks that print resolved values and missing requirements cleanly
- [x] Add a `--print-schema` sanity check to the manual validation steps
- [ ] Add tests for schema decoding, required fields, defaults, and middleware precedence

## Phase 2: Runtime Bootstrap

- [ ] Create `pkg/runtime/factory.go`
- [ ] Create `pkg/runtime/globals.go`
- [ ] Create `pkg/runtime/script_runner.go`
- [ ] Decide where script compilation and execution errors will be normalized into CLI exit behavior
- [ ] Build the runtime via `go-go-goja/engine.NewBuilder(...)`
- [ ] Use `engine.WithModuleRootsFromScript(...)` for script-relative `require()` behavior
- [ ] Add runtime initialization for `console`, `process`, and app globals
- [ ] Add a minimal `process.env` backed by resolved settings plus ambient env snapshot
- [ ] Add `process.cwd`, `process.stdout.write`, `process.stderr.write`, and `process.exitCode`
- [ ] Ensure runtime shutdown is explicit and tested
- [ ] Verify plain `require("./local-helper.js")` works from a fixture script
- [ ] Add a smoke test that runs a trivial JS file end to end

## Phase 3: Runner File Infrastructure

- [ ] Create `pkg/runnerfiles/envfile.go`
- [ ] Create `pkg/runnerfiles/outputfile.go`
- [ ] Create `pkg/runnerfiles/pathfile.go`
- [ ] Create `pkg/runnerfiles/summaryfile.go`
- [ ] Implement single-line and multiline write helpers
- [ ] Ensure all writer helpers can create missing files/directories in temp fixtures when appropriate
- [ ] Add normalization so line endings are stable across platforms in tests
- [ ] Add tests for escaping, append behavior, and file creation semantics
- [ ] Add fixtures for local runs that emulate `GITHUB_ENV`, `GITHUB_OUTPUT`, `GITHUB_PATH`, and `GITHUB_STEP_SUMMARY`
- [ ] Add a small helper that binds resolved Glazed file settings to concrete writer implementations

## Phase 4: `@actions/core`

- [ ] Create `pkg/modules/core/module.go`
- [ ] Create `pkg/modules/core/inputs.go`
- [ ] Create `pkg/modules/core/outputs.go`
- [ ] Create `pkg/modules/core/logging.go`
- [ ] Implement `getInput`, `getBooleanInput`, and `getMultilineInput`
- [ ] Implement `setOutput`, `exportVariable`, and `addPath`
- [ ] Implement `debug`, `info`, `notice`, `warning`, `error`, and `setSecret`
- [ ] Implement `startGroup`, `endGroup`, and `group`
- [ ] Implement step summary builder methods
- [ ] Implement `setFailed` and exit-code propagation
- [ ] Decide whether `exportVariable` mutates the in-process env snapshot in addition to writing the runner env file
- [ ] Add annotation-property handling for notice/warning/error where practical
- [ ] Add JS-facing tests that verify file writes and stdout command formatting
- [ ] Add a fixture script that exercises every core primitive in one run

## Phase 5: Context Loading And `@actions/github`

- [ ] Create `pkg/contextdata/github_context.go`
- [ ] Create `pkg/contextdata/runner_context.go`
- [ ] Create `pkg/contextdata/event_loader.go`
- [ ] Load and validate the event payload JSON from the resolved settings
- [ ] Build the `github.context` snapshot object
- [ ] Add `context.workspace` or equivalent runtime convenience field
- [ ] Decide which fields are first-class in context versus left only in `payload`
- [ ] Create `pkg/githubapi/client.go`
- [ ] Create `pkg/githubapi/request.go`
- [ ] Create `pkg/githubapi/actions_permissions.go`
- [ ] Create `pkg/githubapi/contents.go`
- [ ] Create `pkg/githubapi/workflows.go`
- [ ] Create `pkg/modules/github/module.go`
- [ ] Create `pkg/modules/github/context.go`
- [ ] Create `pkg/modules/github/client.go`
- [ ] Create `pkg/modules/github/rest_actions.go`
- [ ] Implement `getOctokit(token, options?)`
- [ ] Implement generic `request(route, params)` and `paginate(route, params)`
- [ ] Add the first curated `rest.actions.*` helpers needed by the permissions-audit use case
- [ ] Normalize Go API responses into plain JS-friendly objects and arrays
- [ ] Add rate-limit and HTTP error surfaces that are understandable from JS
- [ ] Add `httptest.Server` coverage for success, pagination, auth failures, and API errors

## Phase 6: `@actions/io` And `@actions/exec`

- [ ] Create `pkg/modules/io/module.go`
- [ ] Create `pkg/modules/io/fs.go`
- [ ] Implement `readdir`, `readFile`, `writeFile`, `mkdirP`, `rmRF`, `cp`, `mv`, and `which`
- [ ] Create `pkg/modules/exec/module.go`
- [ ] Create `pkg/modules/exec/exec.go`
- [ ] Implement promise-based `exec(command, args?, options?)`
- [ ] Ensure async callback settlement goes through `runtimeowner.Runner.Post(...)`
- [ ] Decide how streaming stdout/stderr hooks are exposed to JS callers
- [ ] Decide how command timeouts and cancellation map into JS promise rejection
- [ ] Add tests for stdout/stderr capture, cwd/env overrides, and non-zero exit behavior

## Phase 7: Example Scripts

- [ ] Create `examples/set-output.js`
- [ ] Create `examples/permissions-audit.js`
- [ ] Create `examples/list-workflows.js`
- [ ] Add any small helper modules needed under a script-local library directory
- [ ] Add a README snippet showing exactly how to run each example locally
- [ ] Add expected outputs or golden files for each example
- [ ] Verify examples work both locally and under automated tests

## Phase 8: Packaging And Delivery

- [ ] Decide whether the first GitHub-hosted wrapper will be composite or container based
- [ ] Add wrapper action files for the chosen strategy
- [ ] Add release/build plumbing for the `goja-gha` binary
- [ ] Add CI smoke coverage that runs the binary against a fixture script
- [ ] Add a CI job that exercises the permissions-audit example against a fake server or controlled test target
- [ ] Document local usage, CI usage, and wrapper-action limitations

## Exit Criteria

- [ ] `goja-gha run` can execute a fixture JS file using only decoded Glazed settings
- [ ] `@actions/core` can round-trip inputs and runner command files in tests
- [ ] `@actions/github` can call the required Actions permissions endpoints for the imported use case
- [ ] The permissions-audit example can emit a structured result and a workflow output
- [ ] `go test ./...` passes in the repo
- [ ] A smoke path exists for both local execution and CI execution

## Review Checklist

- [ ] Confirm no runtime package reads GitHub env vars directly outside the Glazed input-resolution layer
- [ ] Confirm all commands decode into settings structs via `vals.DecodeSectionInto(...)`
- [ ] Confirm command sections include both Glazed output fields and command settings fields
- [ ] Confirm module registration is explicit at factory-build time
- [ ] Confirm async APIs never touch the Goja VM from background goroutines
- [ ] Confirm the first milestone can audit repository Actions permissions from JavaScript and emit results through `@actions/core`
