---
Title: Create GOJA bindings for GitHub Actions
Ticket: GHA-1
Status: completed
Topics:
    - goja
    - github-actions
    - javascript
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:01-imported-planning-notes.md
Summary: Ticket index for the implemented goja-gha architecture/design/build effort, including imported planning notes and final deliverables.
LastUpdated: 2026-03-10T23:40:00-04:00
WhatFor: Provide the overview and entrypoint documents for the completed goja-github-actions ticket.
WhenToUse: Use when orienting a reviewer or implementer to the delivered scope, primary documents, and tracked work.
---


# Create GOJA bindings for GitHub Actions

## Overview

This ticket defines the initial architecture for a new tool tentatively named `goja-gha`. The goal is to let engineers write GitHub automation in JavaScript while running on a Go/Goja runtime, with a familiar `@actions/*`-style API surface and a concrete first use case around GitHub Actions permissions auditing and workflow inspection.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary design doc**: `design-doc/01-goja-github-actions-design-and-implementation-guide.md`
- **Diary**: `reference/01-diary.md`
- **Imported planning note**: `sources/local/01-imported-planning-notes.md`

## Status

Current status: **completed**

## Topics

- goja
- github-actions
- javascript

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
