---
Title: Move goja-gha settings resolution fully into Glazed sources
Ticket: GHA-2
Status: closed
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
LastUpdated: 2026-03-11T08:05:59.792426863-04:00
WhatFor: ""
WhenToUse: ""
---

# Move goja-gha settings resolution fully into Glazed sources

## Overview

This ticket analyzes how `goja-gha` should stop resolving command settings through ad hoc environment lookups and instead rely on Glazed's native config/env/args/flags pipeline. The current codebase mixes parser responsibilities with runtime environment projection, especially around `ProcessEnv()` and `github.context`.

The deliverable in this ticket is a detailed design and implementation guide for refactoring that behavior. The guide is written for a new engineer or intern and explains the current state, target architecture, phased migration plan, testing expectations, and key risks.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary Design Doc**: [design-doc/01-glazed-native-settings-resolution-design-for-goja-gha.md](./design-doc/01-glazed-native-settings-resolution-design-for-goja-gha.md)
- **Diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **closed**

This ticket was intentionally closed without implementation when the work shifted toward debugging and observability instead. The follow-up ticket is `GHA-3`, which focuses on logging, request tracing, and runtime debugging support for `goja-gha`.

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
