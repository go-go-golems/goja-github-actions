# Changelog

## 2026-03-11

- Initial workspace created
- Added the primary design and implementation guide for a GitHub Actions security assessment script pack.
- Added a detailed phased task backlog covering settings audits, local workflow lint rules, helper modules, and advanced trust-boundary analysis.
- Added a ticket-scoped `/tmp/geppetto` baseline validation script and output capture plan.
- Ran the baseline validation against `/tmp/geppetto` and captured both JSON and human-readable outputs under `scripts/`.
- Recorded live baseline findings for `go-go-golems/geppetto`: `allowed_actions=all`, `sha_pinning_required=false`, `default_workflow_permissions=read`, `selectedActionsStatus=skipped-not-selected-policy`, and seven local workflow files under `.github/workflows`.
- Promoted `permissions-audit.js` into a findings-based baseline repository security audit.
- Added shared JavaScript helpers under `lib/` for findings and workspace/workflow discovery.
- Added normalized `summary` and `findings` output to the baseline audit, including remediation text for weak repository settings.
- Revalidated the baseline audit against `/tmp/geppetto`, which now reports two findings: unrestricted allowed actions and missing SHA pinning requirements.
- Implemented `pin-third-party-actions.js` as the first local workflow lint script.
- Added fixture-style CLI integration coverage for the pinning rule and fixed the initial parser bug so `- uses:` lines are detected correctly.
- Validated the pinning rule against `/tmp/geppetto`, which currently reports 22 unpinned action or reusable-workflow references across the local workflow files.
- Implemented `checkout-persist-creds.js` as the second local workflow lint script.
- Added fixture-style CLI integration coverage for the checkout credential rule and fixed the initial parser so it scans whole step blocks instead of only matching `- uses:` lines.
- Validated the checkout credential rule against `/tmp/geppetto`, which now reports 6 checkout steps missing `persist-credentials: false`.
- Implemented `no-write-all.js` as the third local workflow lint script.
- Added fixture-style CLI integration coverage for workflow-level and job-level `permissions: write-all`.
- Validated the write-all rule against `/tmp/geppetto`, which currently passes with zero findings.
- Extended the Go-native workflow parser so local policy scripts can inspect checkout `ref`, checkout `repository`, and shell `run` steps with line metadata.
- Implemented `pull-request-target-review.js` as the fourth local workflow lint script.
- Added fixture-style CLI integration coverage for both a dangerous `pull_request_target` pattern and the human-readable report output.
- Revalidated `/tmp/geppetto`; the new `pull-request-target-review.js` currently passes with zero findings because no local workflow uses `pull_request_target`.
- Extended the workflow parser again so local policy scripts can inspect `workflow_run` trigger details.
- Implemented `workflow-run-review.js` as the fifth local workflow lint script.
- Added fixture-style CLI integration coverage for both a dangerous `workflow_run` follow-up pattern and the human-readable report output.
- Revalidated `/tmp/geppetto`; the new `workflow-run-review.js` currently passes with zero findings because no local workflow uses `workflow_run`.
- Fixed `make gosec` by tightening runner-file permissions and documenting the intentional subprocess execution boundary in the exec module.
- Implemented `reusable-workflow-trust.js` as the next workflow trust rule, focused on external-owner reusable workflows and unpinned reusable workflow refs.
- Added fixture-style CLI integration coverage for reusable workflow trust checks and revalidated `/tmp/geppetto`; the new rule currently passes with zero findings.
- Implemented `no-privileged-untrusted-checkout.js` as the generalized privileged-trigger checkout rule spanning both `pull_request_target` and `workflow_run`.
- Added fixture-style CLI integration coverage for the generalized untrusted-checkout rule and revalidated `/tmp/geppetto`; the rule currently passes with zero findings.
