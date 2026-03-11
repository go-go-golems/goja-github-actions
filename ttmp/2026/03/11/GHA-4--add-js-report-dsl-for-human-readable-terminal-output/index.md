---
Title: Add JS report DSL for human-readable terminal output
Ticket: GHA-4
Status: active
Topics:
    - goja
    - github-actions
    - javascript
    - glazed
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T11:31:08.197212667-04:00
WhatFor: ""
WhenToUse: ""
---

# Add JS report DSL for human-readable terminal output

## Overview

This ticket covers a new JavaScript-facing UI/report DSL for `goja-gha`. The goal is to let scripts generate readable terminal output for humans without hand-formatting strings or giving up structured return values for machine use.

The shipped implementation adds `@goja-gha/ui`, migrates `examples/permissions-audit.js` to use it, and documents the API through a dedicated Glazed help page.

## Key Links

- **Primary Design Doc**: [design-doc/01-js-report-dsl-api.md](./design-doc/01-js-report-dsl-api.md)
- **Diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- **Tasks**: [tasks.md](./tasks.md)
- **Changelog**: [changelog.md](./changelog.md)

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- github-actions
- javascript
- glazed

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
