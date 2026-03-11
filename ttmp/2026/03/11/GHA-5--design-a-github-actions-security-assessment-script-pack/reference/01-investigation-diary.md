---
Title: Investigation diary
Ticket: GHA-5
Status: active
Topics:
    - github-actions
    - security
    - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/03/11/GHA-5--design-a-github-actions-security-assessment-script-pack/scripts/geppetto-permissions-audit.json
      Note: Captured JSON output from the live geppetto validation
    - Path: ttmp/2026/03/11/GHA-5--design-a-github-actions-security-assessment-script-pack/scripts/geppetto-permissions-audit.txt
      Note: Captured human-readable report output from the live geppetto validation
    - Path: ttmp/2026/03/11/GHA-5--design-a-github-actions-security-assessment-script-pack/scripts/validate-geppetto-security-baseline.sh
      Note: Reproducible validation command for the current baseline against /tmp/geppetto
ExternalSources: []
Summary: Chronological diary of the research and validation work for the GitHub Actions security assessment script pack.
LastUpdated: 2026-03-11T12:22:30.776531586-04:00
WhatFor: Record what was investigated, what was validated, and how the design guide and task list were produced.
WhenToUse: Use when reviewing how the design was derived or when continuing the implementation work in later steps.
---


# Diary

## Goal

Capture the analysis work for the GitHub Actions security assessment script pack so a future engineer can see exactly what evidence was gathered, what assumptions were made, what was validated against `/tmp/geppetto`, and what should happen next.

## Step 1: Create the ticket and map the current surface

This step established a clean ticket workspace and then mapped the current `goja-gha` surface against the imported planning notes. The main goal was to answer a narrow question first: “What scripts could we build now, without inventing new Go runtime features?”

The answer was encouraging. The current repo already has a baseline settings audit in `permissions-audit.js`, a GitHub API client through `@actions/github`, local file access through `@actions/io`, subprocess support through `@actions/exec`, and a human-oriented report DSL through `@goja-gha/ui`. That is enough to define a first real script pack and a sensible phased backlog.

### Prompt Context

**User prompt (verbatim):** "Do the doc, detailed, and add tasks.

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.

then, go ahead, and test in /tmp/geppetto however."

**Assistant interpretation:** Create a new ticketed design packet for the security-assessment script pack, write detailed intern-facing docs and tasks, validate the current baseline against `/tmp/geppetto`, and upload the bundle to reMarkable.

**Inferred user intent:** Convert the earlier brainstorming into an actionable, evidence-backed work packet that can drive implementation and can already be checked against a real repository checkout.

### What I did

- Created ticket `GHA-5`.
- Added the primary design doc.
- Added this diary document.
- Read the imported planning notes from `GHA-1`.
- Inspected the current runtime, module, and example files that define the current implementation surface.

### Why

- The imported notes already had a strong rule inventory, but they needed to be anchored to the actual codebase and split into realistic phases.
- A new engineer needs a document that explains both the current architecture and the proposed scripts, not just a list of names.

### What worked

- The current repo structure made it easy to identify the execution path and module boundaries.
- `permissions-audit.js` turned out to be a strong baseline example rather than a throwaway demo.

### What didn't work

- N/A in this step.

### What I learned

- The pack can start productively with API-backed settings audits and local workflow lints.
- The advanced trust-boundary rules should wait until there is a better normalized workflow model.

### What was tricky to build

- The biggest design judgment was deciding where to draw the line between “build now” and “needs helper infrastructure.” The imported notes include sophisticated trust-boundary checks, but implementing those too early would push the project toward brittle string matching or overly ambitious one-off script logic.

### What warrants a second pair of eyes

- The decision about when to introduce structured YAML parsing.
- The decision about when to move scripts from `examples/` into a more policy-oriented layout.

### What should be done in the future

- Build the first two local lint scripts next so the pack is not just a design.

### Code review instructions

- Start with the design doc and compare it against the imported planning notes.
- Then inspect the current baseline script and modules listed in the design doc’s reference file list.

### Technical details

- Ticket created with:
  - `docmgr ticket create-ticket --ticket GHA-5 --title "Design a GitHub Actions security assessment script pack" --topics github-actions,security,goja`
- Docs created with:
  - `docmgr doc add --ticket GHA-5 --doc-type design-doc --title "GitHub Actions security assessment script pack design and implementation guide"`
  - `docmgr doc add --ticket GHA-5 --doc-type reference --title "Investigation diary"`

## Step 2: Add validation artifacts for `/tmp/geppetto`

After the design outline was stable, the next step was to make the ticket reproducible. Instead of leaving validation as a vague note, I added a dedicated script under the ticket’s `scripts/` directory that runs the current baseline audit against `/tmp/geppetto`, writes both JSON and human-readable outputs, and uses `jq` to print a compact summary.

This matters because the script pack is supposed to be practical, not purely theoretical. A new engineer should be able to run one command from the ticket workspace and see what the baseline audit already tells us about a real repository.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add reproducible validation for the current baseline against `/tmp/geppetto`.

**Inferred user intent:** Make the ticket operational, not just descriptive.

### What I did

- Added a validation shell script under the ticket’s `scripts/` directory.
- Planned to capture both report-mode and `--json-result` outputs inside the same ticket directory.
- Executed the validation script successfully against `/tmp/geppetto`.
- Captured:
  - `scripts/geppetto-permissions-audit.json`
  - `scripts/geppetto-permissions-audit.txt`
- Used `jq` to summarize the result payload.

### Why

- The ticket should contain the exact commands and outputs needed to continue the work.
- Keeping the validation outputs in `scripts/` follows the project convention for ticket-scoped ad hoc artifacts.

### What worked

- `/tmp/geppetto/.github/workflows` exists and is a valid smoke target.
- The baseline script ran cleanly with the current token and repo settings.
- The output captured both GitHub API workflow inventory and local workflow file inventory.

### What didn't work

- N/A in this step.

### What I learned

- `/tmp/geppetto` currently has seven local workflow files under `.github/workflows`, which makes it a useful local-lint validation target even before the advanced scripts exist.
- The GitHub API returned ten workflows for `go-go-golems/geppetto`, while the local checkout currently exposes seven files under `.github/workflows`.
- The live settings snapshot for this repo is:
  - `allowed_actions=all`
  - `sha_pinning_required=false`
  - `default_workflow_permissions=read`
  - `can_approve_pull_request_reviews=false`

### What was tricky to build

- The only subtlety was making the validation script explicit about workspace and repository values so it does not accidentally depend on an unrelated shell state.

### What warrants a second pair of eyes

- The token source and permissions used by the validation environment should remain clear in future follow-up work.

### What should be done in the future

- Once the first local lint scripts exist, extend the validation script to run them too and summarize their findings with `jq`.

### Code review instructions

- Review the ticket `scripts/` directory first.
- Confirm that the outputs written there line up with the current behavior of `permissions-audit.js`.

### Technical details

- Local workflow inventory command used:
  - `find /tmp/geppetto/.github/workflows -maxdepth 1 -type f | sort`
- Validation command used:
  - `/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-5--design-a-github-actions-security-assessment-script-pack/scripts/validate-geppetto-security-baseline.sh`
- `jq` summary returned:

```json
{
  "repository": "go-go-golems/geppetto",
  "workspace": "/tmp/geppetto",
  "workflowCount": 10,
  "localWorkflowFiles": [
    "codeql-analysis.yml",
    "dependency-scanning.yml",
    "lint.yml",
    "push.yml",
    "release.yml",
    "secret-scanning.yml",
    "tag-release-notes.yml"
  ],
  "selectedActionsStatus": "skipped-not-selected-policy",
  "allowedActions": "all",
  "defaultWorkflowPermissions": "read"
}
```
