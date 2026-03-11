# Tasks

## Completion Summary

- [x] Create the ticket workspace and import `/tmp/goja-gha-plan.md`
- [x] Inspect `go-go-goja`, `goja-git`, `goja-github-actions`, and current GitHub Actions references
- [x] Write the design and implementation guide
- [x] Build the `goja-gha` CLI with `run` and `doctor`
- [x] Implement Glazed schema decoding, config/env/default precedence, and schema tests
- [x] Build the Goja runtime factory, process globals, and async entrypoint handling
- [x] Implement runner-file writers for env/output/path/summary files
- [x] Implement `@actions/core`
- [x] Implement `@actions/github` with request, paginate, and the first `rest.actions.*` helpers
- [x] Implement `@actions/io`
- [x] Implement `@actions/exec`
- [x] Add example scripts and fixture event payloads
- [x] Add CLI integration tests that exercise the examples against a fake GitHub API
- [x] Add a local composite action wrapper and CI smoke workflow
- [x] Update the diary, changelog, and design packet for the shipped implementation

## Milestones

- [x] Milestone 1: `goja-gha run <script>` resolves Glazed schema fields, boots Goja, and executes a trivial script successfully
- [x] Milestone 2: `@actions/core` reads inputs and writes outputs/env/path/summary files in local fixture runs
- [x] Milestone 3: `@actions/github` calls the Actions permissions endpoints needed by the permissions-audit example
- [x] Milestone 4: `@actions/io` and `@actions/exec` support workflow inspection and helper command execution
- [x] Milestone 5: a wrapper action and CI workflow can run example scripts in automation

## Delivered Implementation

### Phase 0: Repository Bootstrap

- [x] Real module path and dependency set in `go.mod`
- [x] `cmd/goja-gha/main.go`
- [x] `cmd/goja-gha/cmds/root.go`
- [x] `cmd/goja-gha/cmds/run.go`
- [x] `cmd/goja-gha/cmds/doctor.go`
- [x] Glazed output section, command settings section, logging, and help wiring
- [x] Updated repo README with runnable examples

### Phase 1: Glazed Schema And Input Resolution

- [x] `pkg/cli/github_actions.go`
- [x] `pkg/cli/defaults.go`
- [x] `pkg/cli/middleware.go`
- [x] `pkg/cli/settings.go`
- [x] Explicit runner-env mapping in one place
- [x] Precedence `flags > args > config file > runner env > app defaults > field defaults`
- [x] `doctor` output for resolved settings and validation errors
- [x] Middleware/schema tests in `pkg/cli/middleware_test.go`

### Phase 2: Runtime Bootstrap

- [x] `pkg/runtime/factory.go`
- [x] `pkg/runtime/bindings.go`
- [x] `pkg/runtime/globals.go`
- [x] `pkg/runtime/script_runner.go`
- [x] `engine.WithModuleRootsFromScript(...)`
- [x] `process.env`, `process.cwd()`, `process.stdout.write`, `process.stderr.write`, and `process.exitCode`
- [x] async entrypoint support by awaiting returned Promises
- [x] runtime tests in `pkg/runtime/script_runner_test.go`

### Phase 3: Runner File Infrastructure

- [x] `pkg/runnerfiles/envfile.go`
- [x] `pkg/runnerfiles/outputfile.go`
- [x] `pkg/runnerfiles/pathfile.go`
- [x] `pkg/runnerfiles/summaryfile.go`
- [x] single-line and multiline writers plus directory creation
- [x] normalization and append-behavior tests in `pkg/runnerfiles/runnerfiles_test.go`

### Phase 4: `@actions/core`

- [x] `pkg/modules/core/module.go`
- [x] `pkg/modules/core/inputs.go`
- [x] `pkg/modules/core/outputs.go`
- [x] `pkg/modules/core/logging.go`
- [x] inputs, outputs, env export, path export, summary builder, groups, masking, and `setFailed`
- [x] runtime-state/process-env mutation for env/path/failure propagation
- [x] JS-facing module tests in `pkg/modules/core/module_test.go`
- [x] end-to-end fixture script in `examples/core-primitives.js`

### Phase 5: Context Loading And `@actions/github`

- [x] `pkg/contextdata/github_context.go`
- [x] `pkg/contextdata/runner_context.go`
- [x] `pkg/contextdata/event_loader.go`
- [x] `pkg/githubapi/client.go`
- [x] `pkg/githubapi/request.go`
- [x] `pkg/githubapi/actions_permissions.go`
- [x] `pkg/githubapi/contents.go`
- [x] `pkg/githubapi/workflows.go`
- [x] `pkg/modules/github/module.go`
- [x] `pkg/modules/github/context.go`
- [x] `pkg/modules/github/client.go`
- [x] `pkg/modules/github/rest_actions.go`
- [x] generic `request(route, params)` and `paginate(route, params)`
- [x] first curated `rest.actions.*` helpers for permissions auditing
- [x] JS-friendly `{ status, data, headers }` normalization and `APIError` surfaces
- [x] `httptest.Server` coverage for request building, auth, pagination, and API errors

### Phase 6: `@actions/io` And `@actions/exec`

- [x] `pkg/modules/io/module.go`
- [x] `pkg/modules/io/fs.go`
- [x] `readdir`, `readFile`, `writeFile`, `mkdirP`, `rmRF`, `cp`, `mv`, and `which`
- [x] `pkg/modules/exec/module.go`
- [x] `pkg/modules/exec/exec.go`
- [x] promise-based `exec(command, args?, options?)`
- [x] owner-thread promise settlement via `runtimeowner.Runner.Post(...)`
- [x] stdout/stderr listener hooks
- [x] JS-facing tests for file operations and command execution

### Phase 7: Example Scripts

- [x] `examples/trivial.js`
- [x] `examples/core-primitives.js`
- [x] `examples/set-output.js`
- [x] `examples/permissions-audit.js`
- [x] `examples/list-workflows.js`
- [x] local fixture event in `testdata/events/workflow_dispatch.json`
- [x] repo README snippets showing how to run examples locally
- [x] CLI integration tests in `integration/examples_test.go`

### Phase 8: Packaging And Delivery

- [x] Chosen wrapper strategy: composite action
- [x] Root `action.yml` wrapper
- [x] CI workflow in `.github/workflows/ci.yml`
- [x] CI smoke coverage for the local action and example execution
- [x] Fake-server coverage for the permissions-audit example
- [x] Updated ticket docs and changelog

## Exit Criteria

- [x] `goja-gha run` executes fixture JS using only decoded Glazed settings
- [x] `@actions/core` round-trips inputs and runner command files in tests
- [x] `@actions/github` calls the required Actions permissions endpoints for the imported use case
- [x] The permissions-audit example emits a structured result and a workflow output
- [x] `go test ./...` passes in the repo
- [x] A smoke path exists for both local execution and CI execution

## Validation Checklist

- [x] Confirm no runtime package reads GitHub env vars directly outside the Glazed input-resolution layer
- [x] Confirm all commands decode into settings structs via `vals.DecodeSectionInto(...)`
- [x] Confirm command sections include both Glazed output fields and command settings fields
- [x] Confirm module registration is explicit at factory-build time
- [x] Confirm async APIs never touch the Goja VM from background goroutines
- [x] Confirm the first milestone can audit repository Actions permissions from JavaScript and emit results through `@actions/core`

## Post-Ticket Ideas

- Add richer `@actions/github.rest.*` coverage beyond the initial Actions permissions/workflows endpoints.
- Add timeout and cancellation controls to `@actions/exec` options if future scripts need long-running subprocess management.
- Expand the composite action into a release/distribution story once binary packaging conventions are settled.
