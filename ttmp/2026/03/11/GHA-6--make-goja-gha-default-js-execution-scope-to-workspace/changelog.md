# Changelog

## 2026-03-11

- Initial workspace created
- Added a small implementation design for workspace-first JavaScript execution semantics.
- Changed the runtime to use a shared execution root that prefers workspace over raw working directory.
- Updated `process.cwd()`, `@actions/io`, and default `@actions/exec` behavior to use the execution root.
- Simplified `examples/permissions-audit.js` to use relative workflow discovery again.
- Added runtime, IO, and exec tests covering workspace-first behavior.
- Updated public docs to describe the new workspace-first semantics.
- Validated with `go test ./...`, help-page loads, and a real `permissions-audit.js` run against `/tmp/geppetto`.
