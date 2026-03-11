---
Title: Improve goja-gha debugging and runtime logging
Ticket: GHA-3
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
LastUpdated: 2026-03-11T10:09:55.304986504-04:00
WhatFor: ""
WhenToUse: ""
---

# Improve goja-gha debugging and runtime logging

## Overview

This ticket covers debugging and observability improvements for `goja-gha`. The immediate trigger is that a user can hit a failure such as `github api error: status 401: Bad credentials` while running `examples/permissions-audit.js`, but the current CLI gives very little help beyond the final surfaced error.

The goal here is not just "add more logs". The goal is to make it straightforward to answer questions like:

- which token source was used,
- which repository and workspace were resolved,
- which GitHub API base URL and route were called,
- whether the JS script emitted `console.log(...)`,
- whether a failure came from CLI settings, runtime context construction, HTTP request execution, or JavaScript code.
- whether the CLI error shown to the user is actionable without reading a Goja native stack.

This ticket supersedes the paused settings-resolution work in `GHA-2`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary Design Doc**: [design-doc/01-debugging-and-logging-design-for-goja-gha.md](./design-doc/01-debugging-and-logging-design-for-goja-gha.md)
- **Diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

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
