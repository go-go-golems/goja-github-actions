---
Title: JS report DSL API
Ticket: GHA-4
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-gha/cmds/run.go
      Note: Runtime output suppression contract
    - Path: examples/permissions-audit.js
      Note: First real script migrated to the report DSL
    - Path: pkg/modules/ui/module.go
      Note: Core report DSL implementation
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T11:31:08.23029264-04:00
WhatFor: ""
WhenToUse: ""
---


# JS report DSL API

## Executive Summary

This document describes the `@goja-gha/ui` report DSL added to `goja-gha`. The DSL lets JavaScript scripts build human-readable terminal reports with a small, declarative API instead of manually formatting text or mixing operator-friendly output with machine JSON.

The implementation is intentionally narrow in v1. It supports reports, sections, status lines, key/value rows, lists, and tables. The key runtime rule is that `render()` produces human output only when `--json-result` is not active, and marks runtime state so the CLI does not also print the returned object in the same run.

## Problem Statement

Before this ticket, scripts had an awkward output story:

- `console.log(...)` was easy but structurally weak,
- returning a JavaScript object plus `--json-result` was machine-friendly but not pleasant for human operators,
- examples like `permissions-audit.js` had no good way to show a readable summary without hand-built string formatting.

The specific user need was to let scripts such as `permissions-audit.js` show a readable report during normal local runs. That required more than a helper function. It needed runtime coordination so:

- the script can still return structured data,
- `run` can suppress the automatic returned-object print when a human report has already been rendered,
- `--json-result` can still produce clean JSON for automation.

## Proposed Solution

Add a new native module, `@goja-gha/ui`, with a report-builder DSL:

```js
const ui = require("@goja-gha/ui");

ui.report("GitHub Actions Audit")
  .status("ok", "Inspection complete")
  .kv("Repository", "acme/widgets")
  .section("Workflows", (section) => {
    section.table({
      columns: ["Name", "Path"],
      rows: [["CI", ".github/workflows/ci.yml"]]
    });
  })
  .render();
```

The Go side stores the report as an in-memory block tree and renders it as aligned terminal text.

The current block model is:

- `statusBlock`
- `kvBlock`
- `listBlock`
- `tableBlock`
- `sectionBlock`

Runtime contract:

```text
JavaScript report DSL
    |
    v
Go report model
    |
    +--> render() no-op when JSONResult == true
    |
    +--> render() writes human report when JSONResult == false
           and sets State.HumanOutputRendered = true
    |
    v
run command suppresses automatic returned-object print
when HumanOutputRendered == true and JSONResult == false
```

That contract is the key design point. Without it, scripts would emit both a human report and the returned JSON object during normal runs.

## Design Decisions

### 1. Use a native module instead of a pure JavaScript helper

Reason:

- the renderer needs direct access to runtime mode such as `JSONResult`,
- the module needs to coordinate with CLI output suppression through runtime state,
- Go owns terminal formatting policy and stream selection.

### 2. Keep the DSL declarative

Scripts describe structure:

- report title,
- status lines,
- sections,
- table rows,
- lists.

Scripts do not control low-level spacing or ANSI codes. That keeps output consistent across examples and leaves renderer evolution on the Go side.

### 3. Make `render()` safe to call unconditionally

`permissions-audit.js` now always calls `render()`. The UI module itself decides whether to emit output.

Reason:

- scripts stay simple,
- the JSON/non-JSON policy stays centralized,
- authors do not need to duplicate mode checks in every example.

### 4. Use runtime state to suppress duplicate output

The CLI already had logic to print a returned object automatically for interactive terminals. Adding a report DSL without a suppression flag would have produced duplicate output.

The chosen fix is:

- add `HumanOutputRendered bool` to `runtime.State`,
- set it when the UI module renders in human mode,
- have `run` skip the default returned-object print when that flag is set.

### 5. Use plain text as the only backend in v1

Reason:

- enough to solve the immediate operator-facing problem,
- easy to validate in tests,
- leaves space for future Markdown or alternative backends later.

## Alternatives Considered

### Pure `console.log(...)` formatting in `permissions-audit.js`

Rejected because:

- the formatting logic would be duplicated per script,
- tables and aligned fields would become ad hoc,
- the CLI would still need a duplicate-output suppression story.

### Always print the returned object and skip a DSL entirely

Rejected because:

- the returned object is structurally useful but visually poor for operators,
- scripts like `permissions-audit.js` want status lines, warnings, and tables, not raw JSON.

### Add a generic `console.table(...)` style primitive instead of a report builder

Rejected because:

- the real need is sectioned summaries, not just tables,
- a single table primitive does not solve headings, warnings, or key/value metadata.

### Render to `stderr` in JSON mode

Deferred. It may be useful later, but it introduces another output policy surface. For v1, the simpler rule is that `render()` is suppressed when `--json-result` is active.

## Implementation Plan

1. Add `HumanOutputRendered` to runtime state.
2. Update the `run` command so default result printing respects that flag.
3. Add `pkg/modules/ui` with:
   - `ui.report(title)`
   - `ui.enabled()`
   - report-builder and section-builder methods
   - plain-text renderer
4. Wire the module into `run`.
5. Migrate `examples/permissions-audit.js` to the new DSL.
6. Add:
   - module tests,
   - CLI integration tests,
   - help documentation.

## Open Questions

- Should the module eventually support explicit stream targets such as `stderr`?
- Should the report tree gain a Markdown renderer for step summaries or artifact output?
- Should color/style customization remain renderer-owned forever, or should scripts eventually be able to hint at themes?

## References

- [`pkg/modules/ui/module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go)
- [`pkg/modules/ui/module_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module_test.go)
- [`cmd/goja-gha/cmds/run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [`examples/permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js)
- [`pkg/helpdoc/07-js-report-dsl-api.md`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/helpdoc/07-js-report-dsl-api.md)
