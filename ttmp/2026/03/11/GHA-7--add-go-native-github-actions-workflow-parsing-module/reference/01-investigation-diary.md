---
Title: Investigation diary
Ticket: GHA-7
Status: active
Topics:
    - github-actions
    - goja
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: integration/examples_test.go
      Note: CLI integration coverage for migrated scripts
    - Path: pkg/modules/workflows/module_test.go
      Note: Runtime integration test for require(@goja-gha/workflows)
    - Path: pkg/workflows/parser_test.go
      Note: Unit tests that document expected parser behavior
    - Path: ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/validate-geppetto-workflow-scripts.sh
      Note: Ticket-scoped /tmp/geppetto validation harness
ExternalSources: []
Summary: Chronological implementation diary for the Go-native workflow parsing module and the migration of existing audit scripts onto the new API.
LastUpdated: 2026-03-11T13:07:51.791108294-04:00
WhatFor: Capture why the parser was added, what was implemented in each slice, and how to review or continue the work.
WhenToUse: Use when continuing `GHA-7`, reviewing implementation history, or validating the migration of JS scripts.
---


# Investigation diary

## Goal

Capture the implementation history for `GHA-7`, especially the transition from ad hoc JavaScript YAML parsing to a shared Go-native workflow parsing module.

## Step 1: Establish the ticket scope and migration requirement

The user requested a new ticket for a Go-native GitHub Actions workflow parsing module and asked that the existing scripts move to the new API as part of the work. That migration requirement matters because it changes the scope from “add a parser” to “add a parser and prove it reduces real script complexity.”

### Prompt Context

**User prompt (verbatim):** "alright,  create a ticket to build this, then add tasks, and implement task by task."

**Assistant interpretation:** Create and execute a new ticket for a Go-native workflow parsing module instead of leaving the idea at the discussion stage.

**Inferred user intent:** Replace fragile local workflow parsing with a more maintainable foundation that future security scripts can build on.

### What I did

- Created `GHA-7` scaffold files under `ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module`.
- Inspected the existing workflow-facing scripts and confirmed they still use text parsing and repeated local file reads.
- Confirmed `yaml.v3` is already available in the repo dependency graph.

### Why

- The current scripts already needed parser fixes to handle common YAML layouts, which is a sign that the parser concern is in the wrong layer.
- Before changing code, the ticket needed a real implementation plan and task list.

### What worked

- The repo already had the necessary runtime/module patterns and YAML dependency available.
- Workspace-first execution semantics were already in place, which simplifies workflow discovery.

### What didn't work

- The initial `GHA-7` scaffold was empty and not yet usable as a working ticket record.

### What I learned

- The existing scripts are close enough in purpose that one normalized workflow model should replace most of their parser code.
- The user explicitly wants the existing scripts migrated, not left as legacy examples.

### What was tricky to build

- N/A at this stage. The work so far was investigation and ticket setup.

### What warrants a second pair of eyes

- The exact JS-facing shape of the workflow document API, especially how much permission and trigger detail to expose in v1.

### What should be done in the future

- Implement the parser service and module.
- Migrate the existing scripts and validate them on `/tmp/geppetto`.

### Code review instructions

- Start with the `GHA-7` design doc and task list.
- Then compare the current script implementations in `examples/` against the planned parser-backed replacement.

### Technical details

- Existing scripts inspected:
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/no-write-all.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/list-workflows.js`

## Step 2: Implement the Go workflow parser and native module

The first implementation slice introduced a proper Go-side workflow parser and exposed it to JavaScript as `@goja-gha/workflows`. The key objective was to move YAML decoding, normalization, and line attribution out of the JS rules and into a single shared Go package. That slice landed as commit `8ba4e56` (`Add Go-native workflow parsing module`).

The parser package and the module were built together because the service package alone would not prove anything useful. The runtime wiring in `run.go` was part of the same slice so that the new module could immediately be exercised by tests and then consumed by the existing example scripts in the next slice.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Build the parser ticket in a way that creates a reusable module instead of another one-off helper.

**Inferred user intent:** Establish a shared API that future workflow security rules can trust and reuse.

### What I did

- Added `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/workflows/parser.go`.
- Added `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/workflows/module.go`.
- Wired `@goja-gha/workflows` into `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go`.
- Added unit tests in `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/workflows/parser_test.go`.
- Added runtime integration coverage in `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/workflows/module_test.go`.
- Ran `GOWORK=off go test ./pkg/workflows ./pkg/modules/workflows ./cmd/goja-gha/cmds`.

### Why

- The current scripts had outgrown text parsing.
- `yaml.v3` line metadata is much easier to preserve centrally in Go than in repeated JS regex scanners.

### What worked

- The repo already had a clean native-module pattern to copy.
- `yaml.v3` was already available, so there was no dependency churn.
- The parser API stayed small enough to be understandable: `listFiles`, `parseFile`, `parseAll`.

### What didn't work

- The first runtime integration test expected `goja` to export `[]interface{}`, but the actual exported value was `[]map[string]interface{}`. The failure was:

```text
result = []map[string]interface {}{...}, want []interface{}
```

- The fix was to update the test to assert the concrete exported type instead of an overly generic slice type.

### What I learned

- A narrow normalized document model is enough to replace the existing JS parsers.
- Reusable-workflow job references and checkout-step metadata fit naturally in the same parser walk.

### What was tricky to build

- The main sharp edge was deciding how much structure to expose in v1. A raw YAML AST would have been too low-level, but a “complete GitHub Actions AST” would have been oversized. The compromise was to expose only the fields the current rules actually need while preserving room to grow.

### What warrants a second pair of eyes

- The permission value encoding. The module currently normalizes scalar values and simple maps; if later rules need richer permission semantics, this is the place to extend carefully.

### What should be done in the future

- Add richer trigger and job metadata when `pull_request_target` and `workflow_run` rules are implemented.

### Code review instructions

- Start with `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/workflows/parser.go`.
- Then review `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/workflows/module.go`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions && GOWORK=off go test ./pkg/workflows ./pkg/modules/workflows ./cmd/goja-gha/cmds`

### Technical details

- Exported JS API:
  - `require("@goja-gha/workflows").listFiles()`
  - `require("@goja-gha/workflows").parseFile(path)`
  - `require("@goja-gha/workflows").parseAll()`

## Step 3: Migrate the existing audit scripts and validate on /tmp/geppetto

The second implementation slice switched the existing local audit scripts over to the new workflow API and then validated them in both automated integration tests and a live `/tmp/geppetto` run. That slice landed as commit `b7dd4c5` (`Migrate workflow scripts to parser API`), with the final test fixture adjustment and ticket artifacts recorded in the docs-only follow-up that accompanies this diary update.

This was the point where the new API either proved its value or failed. The migration deleted the duplicated text scanners from the scripts and replaced them with direct consumption of parsed workflow documents. The `/tmp/geppetto` run then confirmed the new path works against a real repository checkout.

### Prompt Context

**User prompt (verbatim):** "you should update the previous scripts to use the new API too btw. 

take notice and then continue working"

**Assistant interpretation:** Do not leave the old examples on the legacy parsing path; migrate them now and keep going on the ticket.

**Inferred user intent:** Ensure the new parser meaningfully simplifies the current scripts instead of existing beside them as dead infrastructure.

### What I did

- Migrated:
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/no-write-all.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/list-workflows.js`
- Simplified `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/workspace.js`.
- Tightened `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go` to include a job-level reusable-workflow reference in the pinning test.
- Added `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/validate-geppetto-workflow-scripts.sh`.
- Captured ticket-scoped validation outputs under `ttmp/.../scripts/`.
- Ran:
  - `cd /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions && GOWORK=off go test ./...`
  - `ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/validate-geppetto-workflow-scripts.sh`

### Why

- The ticket goal was not just to expose a parser, but to make the current scripts better and less fragile.
- A live repository run was needed to verify the new API against realistic workflow files.

### What worked

- The migrated scripts kept their existing output contracts while dropping most of their parser code.
- `/tmp/geppetto` results were sensible:
  - `pin-third-party-actions`: `22` findings
  - `checkout-persist-creds`: `8` findings
  - `no-write-all`: `0` findings
  - `permissions-audit`: `2` findings

### What didn't work

- The first migration pass introduced a naming clash in `permissions-audit.js` between the workflow module import and the GitHub API workflow list variable. That was fixed before the full test run.

### What I learned

- The new module is already enough to simplify the current rule pack without changing user-facing behavior.
- Using the parser for workflow discovery as well as YAML extraction makes the example scripts more internally consistent.

### What was tricky to build

- The main tricky point was keeping the migration focused on parser replacement rather than silently changing rule semantics. The scripts were rewritten to use normalized parser output, but the result schemas and report content were kept intentionally close to the previous behavior.

### What warrants a second pair of eyes

- Review whether `permissions-audit.js` should later consume more parsed local workflow metadata than just the file list.

### What should be done in the future

- Build the next workflow-security rules on top of `@goja-gha/workflows` instead of adding any new text parsers.

### Code review instructions

- Review the migrated example scripts listed above.
- Review `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go` for the end-to-end behavior checks.
- Re-run:
  - `cd /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions && GOWORK=off go test ./...`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/validate-geppetto-workflow-scripts.sh`

### Technical details

- Validation outputs stored under:
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-list-workflows.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-pin-third-party-actions.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-checkout-persist-creds.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-no-write-all.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-permissions-audit.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-workflow-script-summary.json`
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module/scripts/geppetto-workflow-script-summary.txt`

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
