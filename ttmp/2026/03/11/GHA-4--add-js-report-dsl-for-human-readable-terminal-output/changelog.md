# Changelog

## 2026-03-11

- Initial workspace created
## 2026-03-11

- Opened `GHA-4` for a JavaScript report DSL that produces human-readable terminal output.
- Added the new `@goja-gha/ui` native module with a report builder, sections, key/value rows, lists, tables, and status lines.
- Wired the UI renderer into runtime state so human reports suppress the default returned-object print in non-JSON runs.
- Migrated `examples/permissions-audit.js` to the report DSL.
- Added module and CLI integration tests for human report rendering and JSON-result suppression.
- Added a dedicated `js-report-dsl-api` Glazed help page and updated the broader JavaScript API docs.

## 2026-03-11

Implemented report output improvements: collapsed kv blocks, bracket status labels, description() with word-wrap, findings() block with grouping/whyItMatters/remediation/locations, updated all 4 audit scripts, added 7 new tests (commits 114ec11, 7e86110, bd95398, 1f017c9)

### Related Files

- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/checkout-persist-creds.js — Added description and findings() usage
- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/no-write-all.js — Added description and findings() usage
- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/permissions-audit.js — Added description and findings() usage
- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/examples/pin-third-party-actions.js — Added description and findings() usage
- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module.go — Core renderer — collapsed blocks
- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/pkg/modules/ui/module_test.go — 7 new tests for all rendering features

