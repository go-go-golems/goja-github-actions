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
    - Path: examples/checkout-persist-creds.js
      Note: Second local lint script implemented in Step 5
    - Path: examples/pin-third-party-actions.js
      Note: First local lint script implemented in Step 4
    - Path: integration/examples_test.go
      Note: |-
        Integration coverage for the findings-based permissions audit
        Fixture-style integration coverage for the first local lint script
    - Path: lib/findings.js
      Note: Shared helper introduced during the first implementation slice
    - Path: lib/workspace.js
      Note: Shared workspace helper introduced during the first implementation slice
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

## Step 3: Turn the baseline audit into a findings-based repository security audit

This step promoted `permissions-audit.js` from a raw settings dump into the first real repository security assessment script. The script now returns a stable `scriptId`, `summary`, and `findings` contract, and it renders those findings in a human-readable table. I also added the first shared JavaScript helper files under `lib/` so later scripts can reuse the same workspace and findings logic.

The important architectural change is that the repo now has a shared helper layer for security scripts. That means the next workflow-lint scripts do not need to invent their own severity ranking or result-shape conventions.

### Prompt Context

**User prompt (verbatim):** "ok, continue working on thos tasks then"

**Assistant interpretation:** Resume GHA-5 implementation and start completing the planned task list instead of stopping at the design packet.

**Inferred user intent:** Move from planning into incremental implementation while keeping ticket bookkeeping and the diary up to date.

### What I did

- Added:
  - [findings.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/findings.js)
  - [workspace.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/workspace.js)
- Updated [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) to:
  - set `scriptId`,
  - compute normalized `findings`,
  - compute `summary`,
  - attach remediation text,
  - render a `Findings` section in the human report.
- Updated [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) to assert the new JSON/result shape and human-report content.
- Re-ran the ticket validation script and refreshed the captured outputs under `scripts/`.

### Why

- The GHA-5 backlog explicitly called for a structured findings contract before the first useful release.
- The later workflow-lint scripts need a shared severity and summary model, and `permissions-audit.js` was the right place to prove it first.

### What worked

- The new `lib/` helper layout works cleanly with the current CommonJS/module-root setup.
- Integration tests passed without needing Go runtime changes.
- The live `/tmp/geppetto` audit now reports two concrete baseline findings instead of only echoing settings.

### What didn't work

- The ticket validation script originally summarized only the raw policy fields, so it had to be updated to include the new `scriptId`, `summary`, and `findings`.

### What I learned

- `lib/` is a practical place for shared JS helpers in this repo because the runtime’s module roots already include the repo root for scripts under `examples/`.
- A normalized findings contract immediately improves both CLI output and testability.

### What was tricky to build

- The main design choice was deciding which repo-level settings should count as actual findings now versus later. I kept the first pass intentionally narrow: unrestricted allowed actions, missing SHA pinning requirements, non-read-only default token, and Actions approving PR reviews. That keeps the baseline useful without overclaiming on settings we are not yet modeling deeply.

### What warrants a second pair of eyes

- The exact severity assignments for repo-level settings findings.
- Whether `permissions-audit.js` should eventually cover more organization-level comparison logic or stay repo-focused.

### What should be done in the future

- Build `pin-third-party-actions.js` next, using the same `summary/findings` contract.
- Add a shared report helper once the second or third script exists.

### Code review instructions

- Start with [findings.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/findings.js) and [workspace.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/workspace.js).
- Then read [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js).
- Finally review [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) for the expected result shape.

### Technical details

- Validation commands:
  - `GOWORK=off go test ./integration`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/permissions-audit.js --cwd /tmp --workspace /tmp/geppetto --event-path ./testdata/events/workflow_dispatch.json --json-result | jq '{scriptId, summary, findings, allowedActions: .permissions.allowed_actions, shaPinningRequired: .permissions.sha_pinning_required}'`
- Live `/tmp/geppetto` result:

```json
{
  "scriptId": "permissions-audit",
  "summary": {
    "counts": {
      "critical": 0,
      "high": 1,
      "info": 0,
      "low": 0,
      "medium": 1
    },
    "findingCount": 2,
    "highestSeverity": "high",
    "status": "findings"
  },
  "findings": [
    {
      "ruleId": "allowed-actions-not-restricted",
      "severity": "high"
    },
    {
      "ruleId": "sha-pinning-not-required",
      "severity": "medium"
    }
  ],
  "allowedActions": "all",
  "shaPinningRequired": false
}
```

## Step 4: Implement the first local workflow lint rule

This step delivered `pin-third-party-actions.js`, the first real local workflow lint script from the GHA-5 backlog. It scans `.github/workflows/*.yml`, extracts `uses:` references, ignores local and Docker-based references, and flags anything not pinned to a full 40-character commit SHA. It reuses the shared `lib/findings.js` and `lib/workspace.js` helpers introduced in the previous step.

The first implementation had a bug: it matched only `uses:` at the start of a line and missed the common `- uses:` step form. That showed up immediately in the integration test. After fixing the parser, the script behaved correctly and produced a much larger and more realistic finding set on `/tmp/geppetto`.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue implementing the next GHA-5 tasks and keep moving through the backlog.

**Inferred user intent:** Build the script pack incrementally, validate each slice, and keep the ticket documentation continuation-friendly.

### What I did

- Added [pin-third-party-actions.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js).
- Added integration coverage in [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) for:
  - JSON output
  - human-readable report output
- Ran the script against `/tmp/geppetto` in both `--json-result` and human-report modes.

### Why

- This was the highest-value and lowest-complexity local lint rule from the design backlog.
- It also exercises the newly added shared helper layer in a second real script, which is important before adding more lint rules.

### What worked

- The script fits naturally into the current runtime surface; no Go runtime changes were needed.
- The geppetto workflows produced a realistic first result set with many unpinned refs.
- The human report format is already readable enough to use directly in a terminal.

### What didn't work

- The initial parser regex only matched `uses:` and missed `- uses:` lines.
- Exact test failure:

```text
summary.findingCount = 0, want 2
```

and then:

```text
human report missing "pin-third-party-actions"
```

- The first failure exposed the parser bug; the second showed that the human report needed a `Rule` column to make the output self-describing.

### What I learned

- Even a “simple” YAML-lint rule benefits from fixture-based tests immediately, because line-shape assumptions are easy to get wrong.
- `/tmp/geppetto` currently has 22 unpinned action or reusable-workflow references across its local workflow files.

### What was tricky to build

- The parser deliberately stays shallow for now, but that means the matching rules must be chosen carefully. The right compromise for this first pass was:
  - detect plain `uses:` and `- uses:` lines,
  - ignore local `./...` actions,
  - ignore `docker://...`,
  - treat any remaining non-SHA ref as a finding.

### What warrants a second pair of eyes

- Whether GitHub-owned actions such as `actions/checkout` should eventually have a different severity or policy mode than other external actions.
- Whether reusable workflow references should remain under the same rule or split into a dedicated rule later.

### What should be done in the future

- Implement `checkout-persist-creds.js` next.
- Add more workflow fixtures so the next lint rules do not keep re-embedding YAML inline in the test file.

### Code review instructions

- Start with [pin-third-party-actions.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js).
- Then review the two new integration tests in [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go).
- Finally compare the live `/tmp/geppetto` output against the repo’s actual workflow files.

### Technical details

- Validation commands:
  - `GOWORK=off go test ./integration`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/pin-third-party-actions.js --cwd /tmp --workspace /tmp/geppetto --json-result | jq '{summary, findingCount: (.findings|length), firstThree: [.findings[:3][] | {path: .evidence.path, line: .evidence.line, uses: .evidence.uses}]}'`
- Live `/tmp/geppetto` result summary:

```json
{
  "summary": {
    "counts": {
      "critical": 0,
      "high": 22,
      "info": 0,
      "low": 0,
      "medium": 0
    },
    "findingCount": 22,
    "highestSeverity": "high",
    "status": "findings"
  },
  "findingCount": 22
}
```

## Step 5: Implement the checkout credential hardening rule

This step delivered `checkout-persist-creds.js`, the second local workflow lint script in the GHA-5 backlog. It scans workflow step blocks, detects `actions/checkout@...`, and emits a finding when the step does not explicitly set `persist-credentials: false`. This rule is narrower than the pinning rule, but it is important because it targets a concrete hardening control that was called out in the original planning notes.

The first parser version was too shallow. It only matched `- uses:` directly on the step-start line and missed the common form where a step starts with `- name:` and puts `uses:` on the next line. I rewrote the parser to inspect whole step blocks, which fixed both the integration fixture and the real `/tmp/geppetto` results.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue down the GHA-5 task list with the next local lint rule.

**Inferred user intent:** Build out the initial core lint pack methodically, validating each rule against both fixtures and a real repository.

### What I did

- Added [checkout-persist-creds.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js).
- Added integration coverage in [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go) for:
  - JSON output
  - human-readable report output
- Ran the script against `/tmp/geppetto` in `--json-result` and human-report modes.

### Why

- `persist-credentials: false` is one of the clearest workflow-level hardening checks from the original planning notes.
- It complements the new pinning rule well: together they already cover two important supply-chain and token-exposure controls.

### What worked

- The rule produces concrete, path-and-line-specific findings.
- The geppetto smoke is high-signal because it finds real missing hardening settings, not synthetic edge cases.
- The updated block parser is flexible enough for the current workflow styles in the repo.

### What didn't work

- The initial parser only recognized checkout when `uses:` appeared on the same `- ...` line.
- Exact integration failure:

```text
summary.findingCount = 0, want 1
```

- That bug also understated the real geppetto result, showing only 3 findings before the parser was fixed.

### What I learned

- Workflow lint rules that reason about step-local configuration need a step-block parser, not just line-by-line regexes.
- `/tmp/geppetto` currently has 6 checkout steps missing `persist-credentials: false`.

### What was tricky to build

- The subtle part was deciding where a step block ends. The current compromise is pragmatic:
  - detect lines that start a step with `- ...`,
  - collect lines until the next step at the same indentation,
  - inspect the accumulated block for `name:`, `uses:`, and `persist-credentials:`.

### What warrants a second pair of eyes

- Whether this rule should eventually treat missing `persist-credentials` differently from explicitly `true`.
- Whether a later YAML helper should replace the current block-scanning logic.

### What should be done in the future

- Implement `no-write-all.js` next.
- Start moving repeated workflow-scanning logic into a shared helper once the third lint rule lands.

### Code review instructions

- Start with [checkout-persist-creds.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js).
- Then inspect the checkout-specific integration tests in [examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go).
- Finally compare the live `/tmp/geppetto` results against the corresponding checkout steps in the repository workflows.

### Technical details

- Validation commands:
  - `GOWORK=off go test ./integration`
  - `source .envrc && GOWORK=off go run ./cmd/goja-gha run --script ./examples/checkout-persist-creds.js --cwd /tmp --workspace /tmp/geppetto --json-result | jq '{summary, findingCount: (.findings|length), findings: [.findings[] | {path: .evidence.path, line: .evidence.line, stepName: .evidence.stepName, uses: .evidence.uses}]}'`
- Live `/tmp/geppetto` result summary:

```json
{
  "summary": {
    "counts": {
      "critical": 0,
      "high": 6,
      "info": 0,
      "low": 0,
      "medium": 0
    },
    "findingCount": 6,
    "highestSeverity": "high",
    "status": "findings"
  },
  "findingCount": 6
}
```
