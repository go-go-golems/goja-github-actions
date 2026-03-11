---
Title: Design a GitHub Actions security assessment script pack
Ticket: GHA-5
Status: active
Topics:
    - github-actions
    - security
    - goja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-11T12:22:30.699513013-04:00
WhatFor: Analyze and plan a script pack for assessing GitHub Actions security posture with goja-gha, starting from the imported GHA-1 planning notes and the current runtime surface.
WhenToUse: Use when deciding which security assessment scripts to build, how to phase them, and what the current implementation can already validate against a real repository checkout.
---

# Design a GitHub Actions security assessment script pack

## Overview

This ticket captures the design for a practical GitHub Actions security assessment script pack built on top of `goja-gha`. The immediate goal is to turn the raw ideas in the original planning notes into a concrete, phased backlog of JavaScript scripts, each with a clear purpose, data source, output shape, and implementation path.

The ticket is intentionally split into two layers:

- a design document that explains the script pack architecture, rule inventory, file/API dependencies, and phased rollout;
- a reproducible validation script that runs the current baseline assessment against `/tmp/geppetto`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- github-actions
- security
- goja

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling, including the `/tmp/geppetto` validation script and captured outputs
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
