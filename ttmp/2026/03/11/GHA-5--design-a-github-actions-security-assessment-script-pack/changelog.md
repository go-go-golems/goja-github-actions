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
