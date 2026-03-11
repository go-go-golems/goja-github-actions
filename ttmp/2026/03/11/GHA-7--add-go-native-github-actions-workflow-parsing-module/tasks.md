# Tasks

## Milestone

- [x] Add a Go-native `@goja-gha/workflows` module and migrate the existing workflow audit scripts to use it.

## Phase 1: Ticket And API Definition

- [x] Fill in the `GHA-7` ticket docs with the problem statement, API proposal, task list, and diary baseline.
- [x] Relate the main implementation files to the ticket docs.
- [x] Record the user requirement that the existing scripts must migrate to the new API.

## Phase 2: Go Workflow Service Package

- [x] Add `pkg/workflows` for workflow discovery and parsing.
- [x] Implement `.github/workflows` file discovery for `.yml` and `.yaml`.
- [x] Implement YAML decode with `gopkg.in/yaml.v3`.
- [x] Extract normalized workflow document fields:
- [x] file name and relative path
- [x] workflow name
- [x] trigger names
- [x] `uses` references
- [x] checkout steps
- [x] permissions entries
- [x] Preserve file and line metadata.
- [x] Add parser unit tests for representative workflow layouts.

## Phase 3: Goja Native Module

- [x] Add `pkg/modules/workflows/module.go`.
- [x] Expose:
- [x] `listFiles()`
- [x] `parseFile(path)`
- [x] `parseAll()`
- [x] Wire the module into `run.go`.
- [x] Add runtime/module integration tests using `require("@goja-gha/workflows")`.

## Phase 4: Script Migration

- [x] Update `examples/pin-third-party-actions.js` to consume parsed workflow documents.
- [x] Update `examples/checkout-persist-creds.js` to consume parsed workflow documents.
- [x] Update `examples/no-write-all.js` to consume parsed workflow documents.
- [x] Update `examples/permissions-audit.js` to use workflow file discovery from the new API.
- [x] Update `examples/list-workflows.js` to use workflow file discovery from the new API.
- [x] Simplify or remove JS helper code that existed only to work around missing workflow parsing.

## Phase 5: Validation

- [x] Expand `integration/examples_test.go` to cover the migrated scripts.
- [x] Add ticket-scoped validation scripts under `ttmp/.../scripts`.
- [x] Run `GOWORK=off go test ./...`.
- [x] Run the migrated scripts against `/tmp/geppetto`.
- [x] Run `docmgr doctor --ticket GHA-7 --stale-after 30`.

## Phase 6: Bookkeeping And Delivery

- [x] Update the diary after each completed implementation slice.
- [x] Update `changelog.md` with focused entries and commit hashes.
- [x] Commit in focused slices rather than one large batch.
