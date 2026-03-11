# Tasks

## Milestone

- [ ] Add a Go-native `@goja-gha/workflows` module and migrate the existing workflow audit scripts to use it.

## Phase 1: Ticket And API Definition

- [ ] Fill in the `GHA-7` ticket docs with the problem statement, API proposal, task list, and diary baseline.
- [ ] Relate the main implementation files to the ticket docs.
- [ ] Record the user requirement that the existing scripts must migrate to the new API.

## Phase 2: Go Workflow Service Package

- [ ] Add `pkg/workflows` for workflow discovery and parsing.
- [ ] Implement `.github/workflows` file discovery for `.yml` and `.yaml`.
- [ ] Implement YAML decode with `gopkg.in/yaml.v3`.
- [ ] Extract normalized workflow document fields:
  - [ ] file name and relative path
  - [ ] workflow name
  - [ ] trigger names
  - [ ] `uses` references
  - [ ] checkout steps
  - [ ] permissions entries
- [ ] Preserve file and line metadata.
- [ ] Add parser unit tests for representative workflow layouts.

## Phase 3: Goja Native Module

- [ ] Add `pkg/modules/workflows/module.go`.
- [ ] Expose:
  - [ ] `listFiles()`
  - [ ] `parseFile(path)`
  - [ ] `parseAll()`
- [ ] Wire the module into `run.go`.
- [ ] Add runtime/module integration tests using `require("@goja-gha/workflows")`.

## Phase 4: Script Migration

- [ ] Update `examples/pin-third-party-actions.js` to consume parsed workflow documents.
- [ ] Update `examples/checkout-persist-creds.js` to consume parsed workflow documents.
- [ ] Update `examples/no-write-all.js` to consume parsed workflow documents.
- [ ] Update `examples/permissions-audit.js` to use workflow file discovery from the new API.
- [ ] Update `examples/list-workflows.js` to use workflow file discovery from the new API.
- [ ] Simplify or remove JS helper code that existed only to work around missing workflow parsing.

## Phase 5: Validation

- [ ] Expand `integration/examples_test.go` to cover the migrated scripts.
- [ ] Add ticket-scoped validation scripts under `ttmp/.../scripts`.
- [ ] Run `GOWORK=off go test ./...`.
- [ ] Run the migrated scripts against `/tmp/geppetto`.
- [ ] Run `docmgr doctor --ticket GHA-7 --stale-after 30`.

## Phase 6: Bookkeeping And Delivery

- [ ] Update the diary after each completed implementation slice.
- [ ] Update `changelog.md` with focused entries and commit hashes.
- [ ] Commit in focused slices rather than one large batch.
