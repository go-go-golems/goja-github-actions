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
