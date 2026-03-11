---
Title: Make goja-gha default JS execution scope to workspace
Ticket: GHA-6
Status: active
Topics:
    - github-actions
    - goja
    - glazed
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T12:35:05.248793626-04:00
WhatFor: Track the change that makes JavaScript execution, relative IO, and default subprocess execution resolve against the GitHub workspace by default.
WhenToUse: Use when reviewing or extending workspace-vs-cwd behavior in goja-gha.
---

# Make goja-gha default JS execution scope to workspace

## Overview

This ticket aligns `goja-gha` with the expected GitHub Actions mental model: scripts should execute inside the repository workspace by default. After this change, `process.cwd()`, `@actions/io` relative paths, and default `@actions/exec` command execution all resolve against the workspace when one is available.

The previous split between `workspace` and `WorkingDirectory` is still preserved internally for explicit override cases, but the default JavaScript-facing execution scope is now workspace-first.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- github-actions
- goja
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
