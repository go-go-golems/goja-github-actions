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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for the Go-native workflow parsing module and the migration of existing audit scripts onto the new API."
LastUpdated: 2026-03-11T13:07:51.791108294-04:00
WhatFor: "Capture why the parser was added, what was implemented in each slice, and how to review or continue the work."
WhenToUse: "Use when continuing `GHA-7`, reviewing implementation history, or validating the migration of JS scripts."
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

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
