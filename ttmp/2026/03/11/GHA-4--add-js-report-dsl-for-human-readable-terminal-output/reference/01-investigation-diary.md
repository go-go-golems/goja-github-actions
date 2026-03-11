---
Title: Investigation diary
Ticket: GHA-4
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-gha/cmds/run.go
      Note: Output suppression contract for human reports versus returned JSON
    - Path: examples/permissions-audit.js
      Note: First real script migrated to the DSL
    - Path: integration/examples_test.go
      Note: CLI proof that permissions-audit now renders a human report
    - Path: pkg/helpdoc/07-js-report-dsl-api.md
      Note: |-
        Dedicated Glazed help-page reference for the new module
        Detailed Glazed help reference for script authors
    - Path: pkg/modules/ui/module.go
      Note: New native module implementing the report DSL
    - Path: pkg/modules/ui/module_test.go
      Note: Regression coverage for DSL rendering behavior
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T11:31:08.233780466-04:00
WhatFor: ""
WhenToUse: ""
---


# Investigation diary

## Goal

Record the design and implementation work for `GHA-4`, which adds a JavaScript-facing report DSL for human-readable terminal output in `goja-gha`.

## Step 1: Add a native report builder and migrate `permissions-audit.js`

This step added the first real version of the UI DSL. The important architectural decision was to implement it as a native module rather than a pure JavaScript helper. That let the renderer understand runtime mode, respect `--json-result`, and coordinate with the `run` command so a rendered report suppresses the default returned-object print.

The first migrated consumer is `examples/permissions-audit.js`. That script already had the right data model. The missing piece was a concise way to present that data to a human operator without giving up the structured return value needed for `--json-result`.

### Prompt Context

**User prompt (verbatim):** "create a new ticket for this, I like this. Create a detailed js-report-dsl-api in $glazed-help-page-authoring format. Add tasks to the ticket, and create this nice report DSL and use it in goja-github-actions/examples/permissions-audit.js ."

**Assistant interpretation:** Open a new ticket for the proposed report DSL, document the API as a Glazed help page, implement the native module, and migrate the permissions audit example to use it.

**Inferred user intent:** Turn the earlier DSL sketch into a real, documented feature that improves the operator-facing output of `goja-gha`.

**Commit (code):** N/A — code changes not yet committed

### What I did
- Created ticket `GHA-4`.
- Added the new native module in [`pkg/modules/ui/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go).
- Added runtime output coordination in:
  - [`pkg/runtime/factory.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/factory.go)
  - [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- Migrated [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) to render a report with:
  - title,
  - status line,
  - key/value metadata,
  - warnings,
  - workflow table,
  - local workflow-file list.
- Added tests in:
  - [`pkg/modules/ui/module_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module_test.go)
  - [`integration/examples_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go)
- Added the dedicated help page:
  - [`pkg/helpdoc/07-js-report-dsl-api.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/07-js-report-dsl-api.md)

### Why
- A pure JS formatter would not have been enough. The feature needed CLI/runtime coordination.
- `permissions-audit.js` is the right first adopter because it already collects structured data and clearly benefits from a human-facing summary.

### What worked
- The `geppetto` permissions audit now prints a readable summary without `--json-result`.
- The same script still returns clean structured JSON when `--json-result` is set.
- The UI DSL is concise enough that the example reads like a report definition rather than a formatting function.

### What didn't work
- The first section-builder implementation used a generic chaining helper and indirect slice mutation. The symptom was that section tables silently disappeared from the final rendered output.
- The failing output looked like this:

```text
Workflows
---------
```

- with the expected table rows missing.

### What I learned
- The easiest way to make nested report sections reliable in Goja is to bind methods explicitly and mutate section blocks by pointer.
- A report DSL only feels clean if the CLI output policy is also clean. The runtime suppression flag was not optional.

### What was tricky to build
- The tricky part was not the renderer. It was the interaction between:
  - JavaScript DSL calls,
  - Goja function binding,
  - mutable nested section state,
  - CLI output suppression.
- The first implementation looked reasonable but lost nested table state because the section builder mutated copies rather than the real section block.

### What warrants a second pair of eyes
- [`pkg/modules/ui/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go) should be reviewed for long-term ergonomics before more scripts depend on it.
- The output policy in [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go) should be reviewed together with the UI module so future changes do not reintroduce duplicate output.

### What should be done in the future
- Consider a Markdown renderer or step-summary backend.
- Consider whether scripts should eventually be able to force rendering to `stderr` in JSON mode.

### Code review instructions
- Start with [`pkg/modules/ui/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go).
- Then read [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js).
- Then verify the output-mode contract in [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go).
- Validate with:
  - `GOWORK=off go test ./pkg/modules/ui ./integration`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/permissions-audit.js --cwd /tmp/geppetto --event-path ./testdata/events/workflow_dispatch.json`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/permissions-audit.js --cwd /tmp/geppetto --event-path ./testdata/events/workflow_dispatch.json --json-result`

### Technical details

#### Current DSL shape

```js
const ui = require("@goja-gha/ui");

ui.report("Title")
  .status("ok", "Done")
  .kv("Repository", repo)
  .section("Workflows", (section) => {
    section.table({
      columns: ["Name", "Path"],
      rows: [["CI", ".github/workflows/ci.yml"]]
    });
  })
  .render();
```

#### Output contract

```text
if JSONResult == true:
  ui.render() does nothing
  run may emit returned JSON

if JSONResult == false:
  ui.render() writes the human report
  ui.render() sets HumanOutputRendered = true
  run suppresses the default returned-object print
```
