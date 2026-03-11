---
Title: Report Output Improvement Analysis
Ticket: GHA-4
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go:Core report DSL implementation — rendering and block types
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js:Primary audit script using the report DSL
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js:Checkout credential audit script
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js:Pin third-party actions audit script
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/no-write-all.js:Write-all permissions audit script
    - /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/findings.js:Finding creation and summarization utilities
ExternalSources: []
Summary: Analysis of current report DSL output quality issues and implementation plan for making audit reports more user-friendly with rich finding context, grouping, and prose descriptions.
LastUpdated: 2026-03-11T13:24:26.921471209-04:00
WhatFor: Guide implementation of report output improvements — describes the problems, proposed solutions, and implementation order.
WhenToUse: Use when implementing or reviewing the report DSL output improvements.
---

# Report Output Improvement Analysis

## Problem Statement

The `@goja-gha/ui` report DSL renders audit findings as flat tables that show only severity, rule ID, and a short message. However, the finding data model already carries rich context — `whyItMatters`, `remediation.summary`, `remediation.example`, and detailed `evidence` — that is never surfaced in terminal output.

From a user perspective, the current output fails to answer three critical questions:

1. **What is this audit checking and why should I care?**
2. **Why is this finding important?** What is the actual risk?
3. **What do I do about it?** What does a fix look like?

Additionally, when many findings share the same rule (e.g., 22 pinning violations), the flat table is repetitive and hard to scan.

## Current Output (permissions-audit.js)

```
GitHub Actions Audit
====================

WARN   Inspected go-go-golems/geppetto

Repository            go-go-golems/geppetto

Workspace             /tmp/geppetto

Actor                 unknown

Event                 unknown

Assessment            findings

Finding count         2

Highest severity      high

Allowed actions       all

Workflow permissions  read

Workflow count        10

SKIP   selected-actions only applies when allowed_actions == "selected" (got all)

WARN   Runner output file not written: runner output file path is empty

WARN   Step summary file not written: runner summary file path is empty

Findings
--------
Severity  Rule                            Message
--------  ------------------------------  ----------------------------------------------------------------
HIGH      allowed-actions-not-restricted  Repository allows all GitHub Actions and reusable workflows.
MEDIUM    sha-pinning-not-required        Repository does not require full commit SHA pinning for actions.

Workflows
---------
Name                 Path
-------------------  -----------------------------------------
CodeQL Analysis      .github/workflows/codeql-analysis.yml
...
```

### Issues Identified

#### Formatting issues

1. **Excessive blank lines between kv pairs** — every block gets a blank-line separator, making metadata feel vertically bloated.
2. **No visual grouping** — kv metadata flows directly into status/warning messages with no separation.
3. **Status labels lack framing** — bare `WARN` doesn't stand out; `[WARN]` or `[ OK ]` is more conventional.
4. **Tables lack visual framing** — no column separators, hard to scan for wide tables.
5. **Section headings lack breathing room** — no blank line before section headings.
6. **Title/section underlines are noisy in color mode** — when headings are already bold/colored, the underline is redundant.
7. **No indentation for section content** — section body renders at the same level as top-level content.

#### Content issues (user perspective)

1. **No audit description** — user has no idea what this audit checks or why it exists.
2. **Finding detail is absent** — `whyItMatters` and `remediation` fields exist but are never rendered.
3. **No finding grouping** — 22 identical-rule rows could be grouped by rule with a count header.
4. **No location grouping** — evidence locations could be grouped by file path.
5. **"unknown" values are noisy** — `Actor: unknown`, `Event: unknown` add clutter in local runs.
6. **No severity coloring in tables** — severity column is plain text.

## Proposed Output Design

### Report with description and collapsed kv

```
Checkout Persist Credentials
════════════════════════════

  This audit checks that every actions/checkout step in your
  workflow files explicitly sets persist-credentials: false.
  Without this, the GITHUB_TOKEN remains on disk in the runner's
  git config, making credential exfiltration easier if any
  subsequent step is compromised.

[WARN]  6 findings across 7 workflow files

  Workspace       /tmp/geppetto
  Workflow files  7
  Finding count   6
  Highest sev.    high
```

### Finding group with full context

```
Findings (6)
────────────

checkout-persist-creds                                     6 x HIGH
  actions/checkout is used without persist-credentials: false

  Why it matters
    Persisted checkout credentials can widen token exposure inside
    a workflow run and make credential exfiltration easier.

  Remediation
    Add persist-credentials: false under the checkout step's
    with: block unless the workflow has a reviewed reason to
    keep credentials persisted.

  Locations
    .github/workflows/codeql-analysis.yml
       :19  actions/checkout@v6
    .github/workflows/dependency-scanning.yml
       :9   actions/checkout@v6
    .github/workflows/lint.yml
       :21  actions/checkout@v6
    .github/workflows/push.yml
       :13  actions/checkout@v6
    .github/workflows/release.yml
       :18  actions/checkout@v6
    .github/workflows/secret-scanning.yml
       :14  actions/checkout@v6
```

### Mixed-rule audit (permissions-audit)

```
Findings (2)
────────────

allowed-actions-not-restricted                              1 x HIGH
  Repository allows all GitHub Actions and reusable workflows.

  Why it matters
    Allowing all actions increases supply-chain risk because
    workflows can consume mutable third-party actions without an
    allowlist boundary.

  Remediation
    Prefer selected with an explicit allowlist, or local_only
    if the repository only needs local actions.

sha-pinning-not-required                                  1 x MEDIUM
  Repository does not require full commit SHA pinning for actions.

  Why it matters
    Without SHA pinning requirements, mutable tags such as @v1
    can still be used in workflows.

  Remediation
    Enable SHA pinning requirements if your policy is to require
    full commit SHAs for external actions.
```

## Implementation Plan

### Task 1: Collapse consecutive same-type blocks

**What:** Change `renderBlocks()` so that consecutive kvBlocks (and consecutive statusBlocks) do not get inter-block blank lines. Only insert blank line when transitioning between different block types.

**Files:** `pkg/modules/ui/module.go`

**Why first:** This is the simplest change and immediately removes the biggest visual eyesore. No DSL API changes needed.

### Task 2: Add description() block type

**What:** Add a `descriptionBlock` type that stores a prose string. Add `report.description(text)` method. The renderer word-wraps the text at ~72 chars with 2-space indent.

**Files:** `pkg/modules/ui/module.go`

**Why:** Enables scripts to explain what the audit checks and why it exists. Requires word-wrapping helper which is also needed by Task 4.

### Task 3: Add bracket framing to status labels

**What:** Change `styleStatusLabel()` to render `[WARN]`, `[ OK ]`, `[INFO]`, `[SKIP]`, `[ERR ]` instead of bare `WARN `, `OK   `, etc.

**Files:** `pkg/modules/ui/module.go`

**Why:** Small change, immediately more readable. Conventional CLI badge style.

### Task 4: Add findings() block type with grouping

**What:** Add a `findingsBlock` type that accepts a slice of finding objects and render options. The renderer:
- Groups findings by `ruleId`
- For each group: shows rule header with count and severity, the first finding's message, `whyItMatters` (word-wrapped, indented), `remediation.summary` and optional `remediation.example`
- Groups evidence locations by file path with `:line  uses` entries

Add `section.findings(findingsArray, options)` to the JS API.

**Files:** `pkg/modules/ui/module.go`, all four `examples/*.js` scripts

**Why:** This is the single biggest user-facing improvement. It surfaces all the rich context data that already exists in findings.

### Task 5: Update example scripts to use new DSL features

**What:** Update all four audit scripts to:
- Add `.description()` with prose explaining the audit
- Replace manual `.table()` calls in findings sections with `.findings()`
- Clean up any redundant kv pairs

**Files:** `examples/permissions-audit.js`, `examples/checkout-persist-creds.js`, `examples/pin-third-party-actions.js`, `examples/no-write-all.js`

### Task 6: Update tests

**What:** Update existing tests and add new ones for:
- Collapsed kv blocks (no inter-block blank lines)
- description() rendering with word wrap
- Bracket-framed status labels
- findings() block rendering with grouping, whyItMatters, remediation, location grouping
- Edge cases: empty findings, single finding, findings without whyItMatters

**Files:** `pkg/modules/ui/module_test.go`

## Key Design Decisions

### findings() accepts raw JS finding objects

Rather than adding typed Go structs, `findings()` accepts a Goja array value and extracts fields dynamically (same pattern as `table()` with `exportTableOptions`). This keeps the DSL flexible and avoids coupling the renderer to a specific finding schema.

### Options for findings()

```js
section.findings(findingsArray, {
  groupBy: "ruleId",            // field to group by (default: "ruleId")
  locationFields: ["path", "line", "uses"]  // evidence fields to show in locations
})
```

### Word wrapping is internal to the renderer

The renderer wraps prose at ~72 chars with configurable indent. Scripts pass plain text; the renderer handles layout. This keeps the JS API simple.

### description() renders below title, above first block

Position: after the title underline, before any status/kv/section blocks. Rendered as indented word-wrapped prose with a blank line before and after.
