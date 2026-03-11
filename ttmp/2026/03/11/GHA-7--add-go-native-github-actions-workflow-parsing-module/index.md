---
Title: Add Go-native GitHub Actions workflow parsing module
Ticket: GHA-7
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
Summary: "Build a Go-native @goja-gha/workflows module that parses local workflow YAML into normalized JS-facing structures with file and line metadata, then migrate the existing workflow audit scripts to use it."
LastUpdated: 2026-03-11T13:07:51.722750035-04:00
WhatFor: "Reduce fragile ad hoc YAML parsing in JavaScript audit scripts and establish a reusable workflow-analysis foundation for future security checks."
WhenToUse: "Use this ticket when working on local GitHub Actions workflow inspection, rule authoring, or parser-backed JS helper APIs."
---

# Add Go-native GitHub Actions workflow parsing module

## Overview

`goja-gha` already has a useful repository-level security baseline, but the local workflow scripts are still parsing YAML as plain text. That approach was enough for a first pass, but it is brittle: small formatting differences in workflow YAML change what the scripts detect, and each script duplicates its own parser logic.

This ticket replaces those text scanners with a Go-native workflow parsing layer exposed to JavaScript as `@goja-gha/workflows`. The initial goal is pragmatic rather than fully general: parse the data that the current audit scripts need, preserve file and line metadata, and make the API stable enough that future workflow security rules can build on it without re-reading YAML by hand.

This ticket also includes a migration step for the existing scripts. The parser is not useful unless the current scripts actually consume it, and the user explicitly asked that the prior scripts move to the new API as well.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Current focus:

- define and document the first JS-facing workflow parser API,
- implement the Go service package and module adapter,
- migrate the existing local audit scripts,
- validate the result on `/tmp/geppetto`.

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
