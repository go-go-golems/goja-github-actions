---
Title: GitHub Actions security assessment script pack design and implementation guide
Ticket: GHA-5
Status: active
Topics:
    - github-actions
    - security
    - goja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-gha/cmds/run.go
      Note: CLI execution path and result printing semantics for scripts
    - Path: examples/list-workflows.js
      Note: Existing workflow inventory example that informs future lint scripts
    - Path: examples/permissions-audit.js
      Note: Current baseline security audit script used as the starting point for the pack
    - Path: examples/pin-third-party-actions.js
      Note: First implemented local workflow lint script from the GHA-5 backlog
    - Path: lib/findings.js
      Note: Shared findings severity and summary helper now used by the baseline audit
    - Path: lib/workspace.js
      Note: Shared workspace and workflow-file helper used by the baseline audit
    - Path: pkg/modules/exec/module.go
      Note: Subprocess module available for advanced assessment scripts
    - Path: pkg/modules/github/module.go
      Note: GitHub API module used by settings audit scripts
    - Path: pkg/modules/io/module.go
      Note: Filesystem module used by local workflow lint scripts
    - Path: pkg/modules/ui/module.go
      Note: Report DSL used for human-readable security script output
    - Path: pkg/runtime/script_runner.go
      Note: Runtime entrypoint and promise execution model
    - Path: ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/sources/local/01-imported-planning-notes.md
      Note: Original source of the validator pack ideas and GitHub Actions security requirements
ExternalSources: []
Summary: Detailed design and phased implementation guide for a pack of goja-gha scripts that assess GitHub Actions security posture at the repository and workflow level.
LastUpdated: 2026-03-11T12:22:30.743484419-04:00
WhatFor: Help a new engineer understand the current goja-gha architecture, the security checks proposed in the imported planning notes, and the concrete scripts to build first.
WhenToUse: Use when planning or implementing GitHub Actions security assessment scripts, especially when deciding what can be built today versus what needs new runtime support.
---




# GitHub Actions security assessment script pack design and implementation guide

## Executive summary

The imported planning notes in [01-imported-planning-notes.md](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/sources/local/01-imported-planning-notes.md) already identified the right shape for this project: do not build a single monolithic “mirror the settings UI” script. Instead, build a pack of small, composable security assessment scripts that combine three kinds of evidence:

- repository and organization Actions settings from the GitHub REST API;
- workflow metadata from the GitHub Workflows API and local `.github/workflows/*.yml` files;
- human-readable terminal reports plus structured JSON for automation.

The current `goja-gha` runtime is already good enough for the first wave of scripts. It can:

- authenticate to the GitHub API through `@actions/github`;
- inspect local workflow files through `@actions/io`;
- shell out through `@actions/exec` when needed;
- render readable terminal reports through `@goja-gha/ui`.

That means the first scripts should be workflow and settings assessors that do not require a heavy static-analysis engine yet. The best initial pack is:

1. `permissions-audit.js`
2. `pin-third-party-actions.js`
3. `checkout-persist-creds.js`
4. `no-write-all.js`
5. `pull-request-target-review.js`
6. `workflow-run-review.js`

The more advanced cross-workflow trust checks, such as artifact/cross-cache bridges and privileged-untrusted flow analysis, should be treated as a second phase because they want richer parsing, normalization, and graph-building than the current examples provide.

## Problem statement

GitHub Actions security is split across at least two layers, and they fail in different ways:

- repository or organization settings can be too permissive, such as `allowed_actions=all`, broad default workflow permissions, or weak fork approval rules;
- workflow YAML can create dangerous execution paths even when the defaults look reasonable, such as `pull_request_target` plus checkout of attacker-controlled code, `permissions: write-all`, unpinned third-party actions, or `actions/checkout` without `persist-credentials: false`.

The planning notes correctly call out both classes of risk. They also point out an important product requirement: users need more than a pile of raw JSON responses. They need a tool that can answer questions like:

- “Which of my repos still allow all third-party actions?”
- “Which workflows use `pull_request_target`?”
- “Where do we still have `write-all` or unpinned `uses:` references?”
- “Which findings are simple lints, and which ones are real trust-boundary risks?”

This design turns those questions into a script pack with phased implementation.

## Scope

### In scope

- repository-level security assessment scripts implemented as JavaScript entrypoints under `examples/` first, then promoted into a `policies/` or `scripts/security/` layout later;
- JSON-returning scripts that can also render human-readable reports through `@goja-gha/ui`;
- local repository assessment against a checked-out workspace such as `/tmp/geppetto`;
- API-driven checks against the GitHub Actions permissions and workflows APIs;
- YAML and text-based local workflow scanning.

### Out of scope for the first wave

- automated remediation that edits or commits workflow files;
- organization-wide fleet orchestration across many repositories in one run;
- a full semantic workflow graph engine with complete dataflow modeling;
- support for every GitHub API edge case before the first useful scripts ship.

## Current system architecture

This section explains the parts of `goja-gha` that matter for the security script pack. A new intern should understand this before writing scripts.

### CLI entrypoint and execution flow

The main command path is [cmd/goja-gha/cmds/run.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go). The `run` command:

- decodes runner and GitHub-related settings;
- creates a runtime settings object;
- wires in the native modules;
- executes the JS entrypoint;
- prints either structured JSON or a human report.

Relevant call chain:

```text
goja-gha run
  -> DecodeSettings(...)
  -> NewSettings(...)
  -> RunScriptWithModules(...)
     -> CreateRuntime(...)
     -> Require(entrypoint)
     -> execute exported function / await promise
  -> maybePrintScriptResult(...)
```

Key code references:

- [run.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
- [script_runner.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go)

### Runtime model

The runtime entrypoint logic lives in [script_runner.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go). Important behaviors:

- CommonJS entrypoint resolution uses `Require(entrypoint)`.
- If the module exports a function, `goja-gha` invokes it directly.
- If the function returns a Promise, the runtime polls and awaits settlement.
- The runtime already has debug logs for module resolution, function execution, and Promise lifecycle.

This matters because most security scripts should be ordinary JS functions that return a structured result object, optionally after async subprocess or API work.

### GitHub API surface

The `@actions/github` binding is implemented in [pkg/modules/github/module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go). Today it exposes:

- `github.context`
- `github.getOctokit()`

`github.getOctokit()` pulls the token from:

- an explicit JS argument if supplied;
- otherwise `Settings.GitHubToken`.

It then constructs a client used by `permissions-audit.js`. This is enough for repository-level Actions permissions and workflow metadata checks.

### Local filesystem surface

The `@actions/io` binding in [pkg/modules/io/module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/io/module.go) gives scripts access to:

- `readdir`
- `readFile`
- `writeFile`
- `mkdirP`
- `rmRF`
- `cp`
- `mv`
- `which`

Relative paths now resolve against the workspace-first execution root. In practice that means repo-inspection scripts can treat `process.cwd()` and relative `@actions/io` calls as workspace-relative by default.

### Subprocess surface

The `@actions/exec` binding in [pkg/modules/exec/module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go) can launch commands and capture output. This is useful when a script wants to:

- call `git` for repo metadata;
- optionally call `yq`, `jq`, or other local tools;
- gather repo state not yet exposed by Go bindings.

This should be used carefully. For portable security checks, prefer:

- GitHub REST API for remote state;
- `@actions/io` for local files;
- `@actions/exec` only when the local tool is a clear win and easy to explain.

### Human-readable output surface

The report DSL module is implemented in [pkg/modules/ui/module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go). It provides:

- `ui.report(title)`
- `status`, `success`, `note`, `warn`, `error`
- `kv`, `list`, `table`, `section`
- `render()`

This is the right output surface for security assessment scripts. A user running `goja-gha run --script ...` in a terminal should see a readable summary first, then optional JSON in automation mode.

### Existing baseline script

[permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js) is now the baseline repository security audit and the correct starting reference. It already demonstrates:

- API access through `octokit.rest.actions.*`;
- local workspace inspection with `@actions/io`;
- a normalized `summary` plus `findings` contract;
- human report output with `@goja-gha/ui`;
- graceful handling of the `selected-actions` API when `allowed_actions != "selected"`.

That script is not yet the full script pack, but it now proves the baseline result contract and reporting architecture.

## Why the planning notes are directionally correct

The imported notes in [01-imported-planning-notes.md](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/sources/local/01-imported-planning-notes.md) recommend a validator pack with rules such as:

- `no-write-all.js`
- `pin-third-party-actions.js`
- `checkout-persist-creds.js`
- `no-privileged-untrusted-checkout.js`
- `restricted-default-token.js`
- `org-selected-actions.js`
- `reusable-workflow-trust.js`
- `no-artifact-bridge.js`
- `no-cache-bridge.js`

This is the right shape because:

- some findings come from settings APIs;
- some findings come from static workflow linting;
- some findings require trust-boundary analysis across workflows and triggers.

Trying to merge all of this into a single script would make the first deliverable hard to understand and hard to test.

## Security script taxonomy

The script pack should be organized by evidence type and analysis depth.

### Category A: settings audit scripts

These scripts primarily call GitHub’s Actions settings endpoints.

Examples:

- `permissions-audit.js`
- `restricted-default-token.js`
- `org-selected-actions.js`
- `org-fork-approval.js`

Characteristics:

- high signal;
- low implementation complexity;
- easy to validate with the live GitHub API;
- need good 401/403/409 handling.

### Category B: local workflow lint scripts

These scripts primarily read `.github/workflows/*.yml` from the local workspace and apply syntactic or semi-structural rules.

Examples:

- `pin-third-party-actions.js`
- `checkout-persist-creds.js`
- `no-write-all.js`
- `pull-request-target-review.js`
- `workflow-run-review.js`

Characteristics:

- can be built with current `@actions/io` plus minimal YAML parsing support;
- should produce finding lists with path, job, step, message, severity, and remediation;
- can be validated against `/tmp/geppetto` with no GitHub mutation.

### Category C: semantic trust-boundary scripts

These scripts need a model of privileged versus untrusted execution, and often need to reason across workflows or across jobs.

Examples:

- `no-privileged-untrusted-checkout.js`
- `no-artifact-bridge.js`
- `no-cache-bridge.js`
- `reusable-workflow-trust.js`
- `secret-exposure-paths.js`

Characteristics:

- highest security value;
- highest false-positive risk if implemented too early;
- best done after workflow normalization helpers exist.

## Proposed repository layout

The planning notes suggested a `policies/` layout. That is still the right medium-term direction.

Recommended target structure:

```text
examples/
  permissions-audit.js
  list-workflows.js
  pin-third-party-actions.js
  checkout-persist-creds.js
  no-write-all.js
  pull-request-target-review.js
  workflow-run-review.js

pkg/
  modules/
    github/
    io/
    exec/
    ui/
  runtime/

testdata/
  workflows/
    safe/
    unsafe/
    mixed/

ttmp/.../GHA-5.../
  design-doc/
  reference/
  scripts/
```

Later, after the APIs stabilize, these scripts can move into:

```text
policies/
  core/
  org/
  advanced/
```

The initial `examples/` placement is acceptable because the runtime is still evolving and the repo already uses `examples/` as the place for real entrypoint scripts.

## Output contract

Every security script should return a structured JSON object and optionally render a report.

Recommended common result shape:

```json
{
  "scriptId": "pin-third-party-actions",
  "repository": "go-go-golems/geppetto",
  "workspace": "/tmp/geppetto",
  "summary": {
    "severity": "high",
    "findingCount": 3,
    "status": "findings"
  },
  "findings": [
    {
      "ruleId": "pin-third-party-actions",
      "severity": "high",
      "path": ".github/workflows/push.yml",
      "job": "build",
      "step": "checkout",
      "message": "Third-party action is not pinned to a full commit SHA.",
      "evidence": {
        "uses": "some/action@v1"
      },
      "remediation": {
        "summary": "Pin the action to a full commit SHA.",
        "example": "some/action@0123456789abcdef..."
      }
    }
  ]
}
```

This output shape is valuable because it works in:

- terminal reports;
- JSON-based tests;
- future aggregation commands;
- CI annotations or PR comments later.

## Core implementation building blocks

### 1. Shared workflow file discovery helper

The current `permissions-audit.js` already uses a shared helper in [workspace.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/workspace.js) to resolve workspace and local workflow files. Every local workflow lint script should reuse that approach.

Recommended helper responsibilities:

- resolve workspace;
- list workflow files;
- return absolute and repo-relative paths;
- ignore missing workflow directories cleanly.

### 2. Shared findings/report helper

The report DSL is already strong enough, and the first shared helper now exists in [findings.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/findings.js). The scripts still need a shared pattern for:

- counting findings by severity;
- showing `OK` versus `WARN` versus `ERROR`;
- rendering a finding table or a list of findings;
- returning a consistent JSON shape.

Implemented helper API sketch:

```javascript
function summarizeFindings(findings) {
  // implemented in lib/findings.js
}
```

### 3. YAML parsing support

This is the biggest functional gap for the next scripts.

`@actions/io.readFile()` gives us raw file contents, but several planned rules need structured YAML access, not simple string matching. There are three options:

1. start with text/regex scans for the simplest rules;
2. shell out to `yq` through `@actions/exec`;
3. add a native YAML helper module in Go.

Recommendation:

- start with option 1 only for the quickest checks;
- move to option 3 for anything beyond trivial patterns.

Why not rely on `yq` long term:

- it adds an external dependency;
- syntax differences across versions complicate portability;
- the intern implementing the scripts should not need to become a `yq` query expert just to inspect a workflow AST.

### 4. Trust classification helpers

Advanced rules need a normalized concept of:

- privileged triggers;
- untrusted inputs;
- writable token contexts;
- artifact/cache transfer edges.

Recommended helper model:

```text
Workflow
  -> jobs
  -> steps
  -> triggers
  -> permissions
  -> trust labels

Questions:
  - is this workflow privileged?
  - does it ingest attacker-controlled code?
  - can write tokens or secrets reach that code?
```

This should not be implemented in the first lint scripts. It should follow once the simpler scripts establish the shared file/result/report conventions.

## Detailed script backlog

### 1. `permissions-audit.js`

Purpose:

- inspect repo-level GitHub Actions settings;
- provide a baseline security posture snapshot;
- inventory workflows from the GitHub API and local workspace.

Current state:

- implemented in [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js);
- returns `scriptId`, `summary`, and normalized `findings`;
- uses shared helpers in:
  - [findings.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/findings.js)
  - [workspace.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/lib/workspace.js)

What it should grow into next:

- optional org-policy comparison if org settings are fetched too.
- more repo-level settings rules such as fork approval policy or private-fork workflow settings.

Implementation notes:

- preserve the current `selected-actions` skip behavior;
- keep the report human-readable first;
- maintain the current `summary/findings` shape as the contract for future scripts.

### 2. `pin-third-party-actions.js`

Purpose:

- flag `uses:` references that are not pinned to a full commit SHA.

Why it matters:

- mutable refs such as `@v1` and `@main` are a supply-chain risk;
- GitHub’s security hardening guidance recommends pinning actions to full SHAs.

Inputs:

- local workflow YAML files.

Detection logic:

- parse every step with a `uses:` key;
- ignore local actions such as `./path`;
- optionally allow trusted exceptions such as `actions/checkout` for a first pass if policy requires;
- if the ref is not a 40-character hex SHA, emit a finding.

Pseudocode:

```javascript
for (workflowFile of listWorkflowFiles()) {
  const workflow = parseYaml(workflowFile);
  for (job of workflow.jobs) {
    for (step of job.steps) {
      if (!step.uses) continue;
      if (step.uses.startsWith("./")) continue;
      const ref = extractRef(step.uses);
      if (!looksLikeFullSha(ref)) {
        findings.push(...);
      }
    }
  }
}
```

False positives to watch:

- Docker-image `uses:` versus action references;
- reusable workflow references;
- policies that intentionally allow some GitHub-owned actions by tag.

### 3. `checkout-persist-creds.js`

Purpose:

- detect `actions/checkout` usage without `persist-credentials: false`.

Inputs:

- local workflow YAML files.

Detection logic:

- find steps where `uses` starts with `actions/checkout@`;
- inspect `with.persist-credentials`;
- emit a finding if missing or not explicitly `false`.

Important nuance:

- newer checkout versions changed where credentials are stored;
- the policy still makes sense because `persist-credentials: false` remains the explicit opt-out knob.

### 4. `no-write-all.js`

Purpose:

- detect `permissions: write-all` at workflow or job scope.

Inputs:

- local workflow YAML files.

Detection logic:

- inspect top-level `permissions`;
- inspect per-job `permissions`;
- emit findings for `write-all`;
- optionally emit medium-severity warnings for broad explicit maps too.

### 5. `pull-request-target-review.js`

Purpose:

- flag workflows that use `pull_request_target` and summarize why they need review.

Inputs:

- local workflow YAML files.

Detection logic:

- identify workflows triggered by `pull_request_target`;
- capture whether they also:
  - check out PR code;
  - run shell commands;
  - declare broad permissions;
  - use secrets or tokens.

Recommendation:

- this script should not claim exploitability on day one;
- it should be a review-oriented assessor that highlights risk context clearly.

### 6. `workflow-run-review.js`

Purpose:

- flag workflows triggered by `workflow_run`, especially when they consume outputs from earlier untrusted workflows.

Detection logic:

- identify `workflow_run`;
- summarize the downstream job permissions and steps;
- flag artifact download, checkout, or execution behaviors that could turn into trust-boundary issues.

### 7. `restricted-default-token.js`

Purpose:

- assess whether the repo default `GITHUB_TOKEN` permissions are limited as policy expects.

Inputs:

- `GET /repos/{owner}/{repo}/actions/permissions/workflow`

Detection logic:

- expected secure baseline:
  - `default_workflow_permissions=read`
  - `can_approve_pull_request_reviews=false`

This script is mostly a focused derivative of `permissions-audit.js`, but it is still worth having because it creates a sharp, easy-to-automate policy signal.

### 8. `org-selected-actions.js`

Purpose:

- assess whether org and repo settings enforce `allowed_actions=selected` and a real allowlist.

Inputs:

- org and repo selected-actions endpoints;
- repo inventory if run in org mode later.

Why it should be separate:

- the repository policy question is different from the YAML linting question;
- this script will likely grow into organization-wide auditing.

### 9. `no-privileged-untrusted-checkout.js`

Purpose:

- detect the highest-risk case from the planning notes:
  privileged context plus checkout of untrusted code plus writable token or secrets.

Dependencies:

- structured workflow parsing;
- trigger classification;
- checkout/input/ref understanding;
- permission evaluation.

Recommendation:

- do not build this until the simpler workflow lint scripts exist and a helper layer can normalize workflows into a common model.

### 10. `no-artifact-bridge.js` and `no-cache-bridge.js`

Purpose:

- detect flows where untrusted workflows/jobs can feed artifacts or caches into privileged workflows/jobs.

Why these are later:

- they need cross-run or cross-job reasoning;
- they are conceptually important but easy to get wrong with shallow pattern matching.

## Implementation strategy

### Phase 1: strengthen the baseline

Deliverables:

- keep improving `permissions-audit.js`;
- add shared result/report conventions;
- add workflow discovery helper;
- add at least one additional local workflow lint script.

Goal:

- prove the pack pattern with one API-heavy script and one local YAML-heavy script.

### Phase 2: build the core lint pack

Deliverables:

- `pin-third-party-actions.js`
- `checkout-persist-creds.js`
- `no-write-all.js`
- `pull-request-target-review.js`
- `workflow-run-review.js`

Goal:

- ship the practical, high-value, low-complexity rules first.

### Phase 3: add parsing and normalization helpers

Deliverables:

- native YAML helper support or equivalent;
- workflow normalization utilities;
- shared finding/result helper module.

Goal:

- reduce copy-paste logic and false positives.

### Phase 4: advanced trust analysis

Deliverables:

- `no-privileged-untrusted-checkout.js`
- `no-artifact-bridge.js`
- `no-cache-bridge.js`
- `reusable-workflow-trust.js`

Goal:

- move from linting to workflow security reasoning.

## Recommended helper APIs

### JavaScript helper layer

Suggested helper package shape:

```text
helpers/
  workflow-files.js
  findings.js
  report.js
  yaml.js
  trust.js
```

Suggested helper signatures:

```javascript
function resolveWorkspace() {}
function listWorkflowFiles() {}
function loadWorkflowFile(path) {}
function makeFinding(fields) {}
function summarizeFindings(findings) {}
function renderFindingsReport(title, result) {}
```

### Possible future Go-native modules

If YAML parsing becomes a blocker, add a native module such as:

```text
@goja-gha/workflows
  listFiles()
  parseFile(path)
  parseAll()
  findSteps(predicate)
```

This would be a better long-term design than relying on `yq` for core analysis logic.

## Testing strategy

### Unit-level script tests

Create small workflow fixtures representing:

- safe patterns;
- known-bad patterns;
- mixed patterns;
- edge cases such as reusable workflows or no workflow files.

Each script should be testable against fixtures without live API calls where possible.

### Integration tests

Mirror the pattern already used in [integration/examples_test.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/integration/examples_test.go):

- run the real CLI;
- point it at fixture workspaces;
- parse JSON output;
- assert on summary and finding structure.

### Live repo validation

Use `/tmp/geppetto` as an immediate smoke target for:

- `permissions-audit.js`;
- workflow discovery;
- future local-lint scripts.

Important note:

- the local repo can validate file scanning even when some GitHub APIs are not applicable;
- API-backed scripts still depend on the token having the required permissions.

## Validation performed for this ticket

The ticket includes a reproducible script in `scripts/validate-geppetto-security-baseline.sh`. It runs the current baseline assessment against `/tmp/geppetto`, captures both:

- a JSON result;
- a human-readable terminal report.

This is not yet a full security pack validation. It is a baseline validation of the current architecture against a real checkout.

Observed result snapshot for `/tmp/geppetto`:

- repository: `go-go-golems/geppetto`
- workspace: `/tmp/geppetto`
- API-reported workflow count: `10`
- local `.github/workflows` file count: `7`
- `permissions.allowed_actions`: `all`
- `permissions.sha_pinning_required`: `false`
- `workflowPermissions.default_workflow_permissions`: `read`
- `selectedActionsStatus`: `skipped-not-selected-policy`
- `summary.findingCount`: `2`
- `summary.highestSeverity`: `high`

Those results already justify the next scripts:

- `restricted-default-token.js` would report a mostly acceptable default token baseline for this repo;
- `org-selected-actions.js` or a repo-level selected-actions policy checker would flag `allowed_actions=all`;
- a future pinning rule can verify whether the local workflows actually pin action refs even though the repo setting does not require SHA pinning.

## Risks and tradeoffs

### Risk: overusing string matching

Simple text scans are fast to build but fragile for anything more complex than very narrow rules.

Mitigation:

- use them only for the first pass;
- invest in structured YAML parsing before advanced rules.

### Risk: premature “exploitability” claims

Advanced trust-boundary rules can produce false positives or oversimplify GitHub semantics.

Mitigation:

- make the first advanced scripts review-oriented rather than overly absolute;
- surface evidence and remediation, not just verdicts.

### Risk: too many unrelated output shapes

If every script invents its own result format, aggregation and tests become painful.

Mitigation:

- standardize around the common result shape described above.

## Open questions

1. Should YAML parsing be introduced as a Go-native module before the second script, or after proving one regex/text-based lint?
2. Should the repo keep real security scripts under `examples/` until the APIs settle, or introduce `policies/` immediately?
3. Should organization-level auditing become a separate command later, or stay script-based?

## Recommended next steps

1. Promote `permissions-audit.js` from “example” status to “baseline repo security audit”.
2. Implement `pin-third-party-actions.js` and `checkout-persist-creds.js`.
3. Add shared finding/result helpers before the fourth or fifth script.
4. Introduce structured workflow parsing before `no-privileged-untrusted-checkout.js`.

## Reference file list

- Imported planning source:
  - [01-imported-planning-notes.md](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/sources/local/01-imported-planning-notes.md)
- Existing baseline scripts:
  - [permissions-audit.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js)
  - [list-workflows.js](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/list-workflows.js)
- Runtime and CLI:
  - [run.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/cmd/goja-gha/cmds/run.go)
  - [script_runner.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/runtime/script_runner.go)
- Native modules:
  - [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/github/module.go)
  - [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/io/module.go)
  - [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/exec/module.go)
  - [module.go](/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go)
- GitHub references:
  - GitHub Actions permissions REST API: https://docs.github.com/en/rest/actions/permissions
  - Workflow syntax and permissions: https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax
  - Security hardening for GitHub Actions: https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions
  - Fine-grained PAT permissions reference: https://docs.github.com/en/rest/overview/permissions-required-for-fine-grained-personal-access-tokens
