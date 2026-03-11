---
Title: Go-native workflow parsing module for goja-gha
Ticket: GHA-7
Status: active
Topics:
    - github-actions
    - goja
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-gha/cmds/run.go
      Note: Runtime module wiring for @goja-gha/workflows
    - Path: examples/checkout-persist-creds.js
      Note: First migrated rule using parsed checkout step data
    - Path: examples/no-write-all.js
      Note: First migrated rule using parsed permissions data
    - Path: examples/pin-third-party-actions.js
      Note: First migrated rule using parsed uses references
    - Path: pkg/modules/workflows/module.go
      Note: JavaScript-facing native module adapter
    - Path: pkg/workflows/parser.go
      Note: Core workflow discovery and YAML normalization logic
ExternalSources: []
Summary: Design for a Go-native workflow parsing package and @goja-gha/workflows native module that exposes normalized GitHub Actions workflow structures to JavaScript audit scripts.
LastUpdated: 2026-03-11T13:07:51.797258808-04:00
WhatFor: Guide the implementation and review of workflow parsing in Go, and explain how the JS audit scripts should consume the resulting API.
WhenToUse: Use when implementing the parser, reviewing the API shape, or onboarding a new contributor to workflow-analysis work.
---


# Go-native workflow parsing module for goja-gha

## Executive Summary

The current security scripts under [`examples/`](../../../../../../examples) inspect local workflow files by reading YAML as plain text and applying regular expressions. That is fast to prototype, but it does not scale well. The scripts already needed parser rewrites to catch common YAML layouts such as `- name: ...` followed by `uses:` on a later line, and each new rule would repeat similar logic.

This ticket introduces a Go-native parser package, tentatively `pkg/workflows`, and a JavaScript-facing native module, `@goja-gha/workflows`. The Go package owns YAML decoding, traversal, normalization, and file/line attribution. The JS module exposes a small, concrete API that scripts can use directly:

- `listFiles()`
- `parseAll()`
- `parseFile(path)`

Each parsed workflow document will include normalized slices for the items the current rules need:

- action or reusable-workflow references (`uses`)
- checkout steps (`checkoutSteps`)
- permissions entries (`permissions`)

This is intentionally narrower than a full GitHub Actions AST. The first objective is to replace fragile per-script text parsing with one shared parser that is easy to reason about and easy to extend.

## Problem Statement

The local workflow audit scripts currently have three structural problems.

First, they treat YAML as text instead of as a document tree. This makes the logic sensitive to indentation quirks and formatting choices that are not semantically meaningful. A step can be written across multiple lines, fields can move around inside mappings, and quoted or unquoted values can appear in different layouts. The regex-based scripts have to rediscover those shapes one rule at a time.

Second, the parsing logic is duplicated. [`pin-third-party-actions.js`](../../../../../../examples/pin-third-party-actions.js), [`checkout-persist-creds.js`](../../../../../../examples/checkout-persist-creds.js), and [`no-write-all.js`](../../../../../../examples/no-write-all.js) all re-open the same workflow files and scan them with unrelated custom parsers. That duplication increases maintenance cost and makes it harder to trust the findings because every script has its own blind spots.

Third, the current JS scripts are missing a clear semantic boundary between “workflow file discovery”, “YAML parsing”, and “policy evaluation”. A stronger split would make the system easier for a new intern to understand:

- Go: decode and normalize workflow structure
- JS: express policy rules and render findings

That split matches the broader `goja-gha` architecture, where Go owns runtime integration and JS owns task-specific logic.

## Proposed Solution

The proposed solution has two layers.

The first layer is a Go service package, `pkg/workflows`, that performs local workflow discovery and parsing. It should:

- find `.yml` and `.yaml` files under `.github/workflows`,
- parse them with `gopkg.in/yaml.v3`,
- walk the YAML node tree,
- extract normalized records that policy scripts can consume,
- preserve path and line-number evidence.

The second layer is a Goja native module, `@goja-gha/workflows`, that exposes those parsed structures to JavaScript. The module should follow the same native-module pattern as [`pkg/modules/io`](../../../../../../pkg/modules/io/module.go) and [`pkg/modules/github`](../../../../../../pkg/modules/github/module.go): keep the parser logic in a pure Go package and keep the Goja adapter focused on JS conversion and export wiring.

The initial JavaScript API should be:

```javascript
const workflows = require("@goja-gha/workflows");

const files = workflows.listFiles();
const docs = workflows.parseAll();
const single = workflows.parseFile(".github/workflows/ci.yml");
```

The returned documents should have a stable, explicit shape. A first pass that is strong enough for the current scripts looks like this:

```javascript
{
  fileName: "ci.yml",
  path: ".github/workflows/ci.yml",
  name: "CI",
  triggerNames: ["push", "pull_request"],
  uses: [
    {
      kind: "step",
      jobId: "build",
      stepName: "Checkout",
      uses: "actions/checkout@v4",
      line: 17
    },
    {
      kind: "job",
      jobId: "reusable",
      uses: "org/repo/.github/workflows/build.yml@main",
      line: 44
    }
  ],
  checkoutSteps: [
    {
      jobId: "build",
      stepName: "Checkout",
      uses: "actions/checkout@v4",
      line: 17,
      persistCredentials: null
    }
  ],
  permissions: [
    {
      scope: "workflow",
      jobId: null,
      line: 5,
      kind: "scalar",
      value: "read-all"
    },
    {
      scope: "job",
      jobId: "release",
      line: 53,
      kind: "scalar",
      value: "write-all"
    }
  ]
}
```

The current scripts then become much simpler:

- `pin-third-party-actions.js` iterates over `doc.uses`
- `checkout-persist-creds.js` iterates over `doc.checkoutSteps`
- `no-write-all.js` iterates over `doc.permissions`
- `permissions-audit.js` and `list-workflows.js` use `listFiles()` instead of direct `io.readdir()`

### Architecture Diagram

```text
JS example script
  |
  | require("@goja-gha/workflows")
  v
Goja native module adapter
pkg/modules/workflows/module.go
  |
  | calls
  v
Workflow service package
pkg/workflows/*
  |
  | reads .github/workflows/*.yml
  | parses yaml.v3 nodes
  | emits normalized records with line metadata
  v
JS-facing plain objects
```

### Separation of Concerns

- `pkg/workflows`
  - file discovery
  - YAML decode
  - traversal and normalization
  - plain Go structs
- `pkg/modules/workflows`
  - `require("@goja-gha/workflows")`
  - `goja.Value` conversion
  - argument decoding and error translation
- `examples/*.js`
  - policy logic
  - findings
  - human-readable report output

### Pseudocode

```text
parseAll(root):
  files = listWorkflowFiles(root)
  docs = []
  for each file in files:
    docs.append(parseWorkflowFile(root, file))
  return docs

parseWorkflowFile(root, relPath):
  yamlNode = read + yaml decode
  doc = new WorkflowDocument(relPath)
  rootMapping = unwrap document node

  if root has "name":
    doc.name = scalar value

  doc.triggerNames = collectTriggerNames(root["on"])
  doc.permissions += collectPermissions(scope=workflow, jobId="")

  jobs = root["jobs"]
  for each jobId -> jobNode in jobs:
    doc.permissions += collectPermissions(scope=job, jobId=jobId)
    if jobNode has "uses":
      doc.uses += reusable workflow reference
    for each step in jobNode["steps"]:
      if step has "uses":
        doc.uses += step reference
        if uses starts with "actions/checkout@":
          doc.checkoutSteps += checkout metadata

  return doc
```

## Design Decisions

### Decision 1: Use a Go-native parser instead of a JS YAML library

This keeps YAML semantics, error handling, and normalization in one place. The main benefit is not raw performance; it is consistency. Every security rule should see the same workflow model, not its own home-grown parser.

### Decision 2: Return a normalized workflow document, not a raw YAML AST

A raw AST would expose too much parser complexity to JS and push low-level traversal back into the scripts. The initial API should focus on what scripts need most often. If a later rule needs additional workflow details, the Go package can grow those fields incrementally.

### Decision 3: Preserve file and line metadata in every extracted record

Security findings are only useful if the user can map them back to a concrete location in the workflow. `yaml.v3` gives line numbers on nodes, so the normalized objects should carry them forward.

### Decision 4: Keep file discovery workspace-relative

The repo already moved to workspace-first execution semantics. The workflow module should follow that model and treat `.github/workflows` as relative to the execution root, which now defaults to the workspace.

### Decision 5: Migrate the existing scripts immediately

The parser module is only justified if it reduces complexity in the real scripts. The user also asked explicitly that the earlier scripts move to the new API.

## Alternatives Considered

### Alternative 1: Keep regex scanners and just improve them

Rejected because the complexity would keep growing in the scripts themselves. Every new edge case would produce another round of parser fixes, and line attribution would remain fragile.

### Alternative 2: Expose only raw YAML parse results to JS

Rejected for the first iteration because it still forces every script author to understand YAML node trees and GitHub Actions layout details. That is not a good onboarding story for a new intern.

### Alternative 3: Build a very large, full-fidelity GitHub Actions AST first

Rejected for now because it would slow delivery and overfit the first version. The immediate need is to support the current security scripts and give future rules a sane base. A smaller normalized model is enough.

## Implementation Plan

1. Document the ticket properly: index, design doc, tasks, diary.
2. Add `pkg/workflows` for discovery and parsing.
3. Add `pkg/modules/workflows` with `listFiles`, `parseFile`, and `parseAll`.
4. Wire the module into [`cmd/goja-gha/cmds/run.go`](../../../../../../cmd/goja-gha/cmds/run.go).
5. Add unit tests for parser behavior and runtime integration tests for `require("@goja-gha/workflows")`.
6. Migrate:
   - [`examples/pin-third-party-actions.js`](../../../../../../examples/pin-third-party-actions.js)
   - [`examples/checkout-persist-creds.js`](../../../../../../examples/checkout-persist-creds.js)
   - [`examples/no-write-all.js`](../../../../../../examples/no-write-all.js)
   - [`examples/permissions-audit.js`](../../../../../../examples/permissions-audit.js)
   - [`examples/list-workflows.js`](../../../../../../examples/list-workflows.js)
7. Add integration coverage and validate on `/tmp/geppetto`.
8. Update the ticket diary, changelog, and task state after each completed slice.

## Open Questions

The first open question is how far the initial API should go on trigger normalization. A simple `triggerNames` slice is probably enough for the current phase, but more semantic trigger detail may be useful for `pull_request_target` and `workflow_run` rules later.

The second open question is whether permissions should include mapping payloads in addition to scalar values. `no-write-all` only needs scalar detection today, but more advanced rules may want the full permission map. The recommended first step is to preserve both `kind` and `value`, where `value` can be either a string or a map.

The third open question is how to handle malformed YAML files. The likely answer is to fail fast with a clear parse error that includes the file path, because a security audit should not silently skip invalid workflow definitions.

## References

- [`run.go`](../../../../../../cmd/goja-gha/cmds/run.go)
- [`pkg/modules/io/module.go`](../../../../../../pkg/modules/io/module.go)
- [`pkg/modules/github/module.go`](../../../../../../pkg/modules/github/module.go)
- [`examples/pin-third-party-actions.js`](../../../../../../examples/pin-third-party-actions.js)
- [`examples/checkout-persist-creds.js`](../../../../../../examples/checkout-persist-creds.js)
- [`examples/no-write-all.js`](../../../../../../examples/no-write-all.js)
- [`examples/permissions-audit.js`](../../../../../../examples/permissions-audit.js)
