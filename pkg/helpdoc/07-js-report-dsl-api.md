---
Title: "JS Report DSL API"
Slug: "js-report-dsl-api"
Short: "Reference for the @goja-gha/ui report builder used to render human-readable terminal summaries from JavaScript."
Topics:
- goja
- github-actions
- javascript
- glazed
Commands:
- run
Flags:
- script
- json-result
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page explains the `@goja-gha/ui` report DSL. It is the main reference for script authors who want to produce human-readable terminal output without hand-formatting strings, ANSI escapes, or ad hoc table layouts.

The core idea is simple: JavaScript describes the structure of a report, and Go renders that structure into terminal-friendly text. That keeps scripts focused on intent instead of spacing and formatting rules.

## Why This Module Exists

This section explains the problem the module solves, how it behaves in practice, and why it matters to both script authors and reviewers.

Before `@goja-gha/ui`, scripts had two awkward choices:

- write raw `console.log(...)` output and lose structure,
- return a JSON object and rely on `--json-result`, which is good for machines but not ideal for humans.

The report DSL gives you a middle path. A script can still return a structured object for automation, but it can also render a readable report for operators.

The current implementation also coordinates with the CLI result printer:

- if a script renders a UI report in normal human mode, `run` suppresses the automatic returned-object print,
- if `--json-result` is enabled, the report renderer becomes a no-op so machine-readable JSON stays clean.

That coordination is the reason this is a runtime module rather than a helper library implemented only in JavaScript.

## Quick Start

This section shows the smallest useful example. Use it first so you can see the builder shape before reading the reference tables.

```js
const ui = require("@goja-gha/ui");

module.exports = function () {
  ui.report("Workflow Audit")
    .status("ok", "Inspection completed")
    .kv("Repository", "acme/widgets")
    .kv("Allowed actions", "all")
    .section("Workflows", (section) => {
      section.table({
        columns: ["Name", "Path"],
        rows: [
          ["CI", ".github/workflows/ci.yml"],
          ["Lint", ".github/workflows/lint.yml"]
        ]
      });
    })
    .render();

  return { ok: true };
};
```

When you run that script without `--json-result`, the terminal sees a report instead of raw JSON. When you run the same script with `--json-result`, the UI render is skipped and the returned object is emitted as JSON.

## Mental Model

This section explains how the module works conceptually, not just which methods exist. That matters because the behavior around `render()` and `--json-result` is policy, not just syntax.

Think of the module as a two-step system:

1. JavaScript builds an in-memory report tree.
2. Go renders that tree when `render()` is called.

The report tree is made of blocks such as:

- status lines,
- key/value rows,
- bullet lists,
- tables,
- titled sections.

Pseudocode:

```text
report = ui.report("Title")
report.kv("Repository", repo)
report.section("Workflows", (section) => {
  section.table(...)
})
report.render()
```

Runtime flow:

```text
JavaScript script
    |
    v
@goja-gha/ui builder methods
    |
    v
Go report model
    |
    v
terminal text renderer
    |
    +--> writes human output when JSONResult == false
    |
    +--> no-op when JSONResult == true
```

Why this matters:

- scripts stay declarative,
- formatting logic stays centralized,
- the CLI can avoid printing two human-facing outputs for the same run.

## The Top-Level Module

This section covers the functions exported directly by `require("@goja-gha/ui")`.

### `ui.report(title)`

Creates a new report builder object.

Example:

```js
const ui = require("@goja-gha/ui");
const report = ui.report("GitHub Actions Audit");
```

### `ui.enabled()`

Returns `true` when the current run is in human-report mode. In the current implementation this means `--json-result` is not active.

Use this only when you need conditional behavior. Most scripts can simply call `report.render()` and let the module decide whether to emit output.

## Report Builder API

This section is the method-by-method reference for the report builder returned by `ui.report(...)`.

All builder methods except `render()` return the builder object so calls can be chained.

### `report.status(kind, text)`

Adds a status line. Supported `kind` values in v1 are:

- `ok`
- `info`
- `warn`
- `error`
- `skip`

Aliases like `success` and `warning` normalize to the canonical values.

Example:

```js
report.status("skip", "selected-actions is not applicable for this repository");
```

### Convenience status methods

Available methods:

- `report.success(text)`
- `report.note(text)`
- `report.warn(text)`
- `report.error(text)`

These are shorthand for common `status(...)` calls.

### `report.kv(label, value)`

Adds a key/value row. The renderer aligns labels within the current block set so the output stays readable.

Example:

```js
report.kv("Repository", "go-go-golems/geppetto");
report.kv("Workflow count", 10);
```

### `report.list(items)`

Adds a bullet list. `items` is usually a JavaScript array of strings, but the renderer will stringify scalar values as needed.

Example:

```js
report.list(["push.yml", "release.yml", "lint.yml"]);
```

### `report.table({ columns, rows })`

Adds a table block.

Expected shape:

```js
report.table({
  columns: ["Name", "Path"],
  rows: [
    ["CI", ".github/workflows/ci.yml"],
    ["Lint", ".github/workflows/lint.yml"]
  ]
});
```

Rules:

- `columns` should be an array of strings,
- `rows` should be an array of arrays,
- extra row cells are ignored,
- missing cells render as empty strings.

### `report.section(title, fn)`

Adds a titled section and invokes `fn(section)` with a section builder.

Example:

```js
report.section("Warnings", (section) => {
  section.warn("Runner output file not configured");
  section.warn("Step summary file not configured");
});
```

Use sections when you want clear visual grouping. Sections can themselves contain nested sections in the current implementation.

### `report.render()`

Renders the report.

This is the point where the DSL becomes visible to the user. Until `render()` is called, the builder is only collecting blocks.

Current behavior:

- if `--json-result` is **not** set:
  - render to standard output,
  - mark runtime state so the CLI does not also print the returned object.
- if `--json-result` **is** set:
  - do not render the human report,
  - leave the returned object available for JSON output.

That policy is why `render()` is safe to call unconditionally from `permissions-audit.js`.

## Section Builder API

This section covers the builder object passed into `report.section(title, fn)`.

The section builder supports the same content-building methods as the report builder, except it does not have its own `render()` method:

- `status(kind, text)`
- `success(text)`
- `note(text)`
- `warn(text)`
- `error(text)`
- `kv(label, value)`
- `list(items)`
- `table({ columns, rows })`
- `section(title, fn)`

Example:

```js
report.section("Workflows", (section) => {
  section.table({
    columns: ["Name", "Path"],
    rows: workflows.map((workflow) => [workflow.name, workflow.path || ""])
  });
});
```

## Output Rules

This section explains the current rendering contract. It matters because output behavior is part of the CLI UX, not just a formatting detail.

Current renderer rules:

- headings render as plain terminal text,
- section headings render with separators,
- tables render as aligned columns,
- lists render as `- item`,
- status lines render as uppercase labels like `OK`, `WARN`, `SKIP`,
- color is enabled only when the output stream looks like a real terminal and color is not disabled.

The module intentionally does **not** expose low-level style controls in v1. JavaScript describes meaning, not terminal escape sequences.

This is a deliberate design decision:

- good scripts should survive no-color terminals,
- the renderer should remain free to evolve,
- consistent output matters more than per-script styling freedom.

## Real Example: `permissions-audit.js`

This section shows how the shipped example uses the DSL in practice and why the DSL is better than manual formatting for this case.

The current `permissions-audit.js` script uses the report DSL to render:

- an overall success line,
- repository and policy metadata,
- a skip message when `selected-actions` does not apply,
- warnings for missing local runner output files,
- a workflow table from the GitHub API,
- a local workflow-file list from the checkout.

That gives the user a readable report during a normal `run`, while the same script still returns a structured result for `--json-result`.

## File References

This section points you to the implementation files if you need to review or extend the DSL.

- [`module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go)
- [`module_test.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module_test.go)
- [`run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [`permissions-audit.js`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js)

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `render()` shows nothing during `--json-result` runs | The renderer intentionally suppresses human output in machine-output mode | Remove `--json-result` for human runs, or inspect the returned JSON instead |
| The returned object still prints after a report | The runtime did not record human output as rendered | Review [`run.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go) and [`module.go`](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go) together |
| A table renders blank | `columns` or `rows` is not the expected array shape | Pass `table({ columns: [...], rows: [[...], [...]] })` |
| Colors do not appear | The output stream is not a terminal, `NO_COLOR` is set, or `TERM=dumb` | This is expected; the renderer falls back to plain text |
| The script needs machine output and human output at once | The current v1 renderer is intentionally single-mode by default | Return a structured object and rely on `--json-result`; multi-stream rendering is future work |

## See Also

- `goja-gha help javascript-api`
- `goja-gha help user-guide`
- `goja-gha help debugging-goja-gha`
