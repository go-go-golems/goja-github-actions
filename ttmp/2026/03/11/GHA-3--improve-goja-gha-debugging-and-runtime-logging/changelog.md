# Changelog

## 2026-03-11

- Initial workspace created
- Opened to replace the paused Glazed settings-resolution work with a practical debugging and observability track.
- Added secret-safe debug logs in the run command, runtime execution path, GitHub client construction, and GitHub HTTP request layer.
- Added a debugging help page and a validated console-debug script showing how to use `--log-level debug`, `--log-format text`, and JavaScript `console.*` together.
- Fixed `github.getOctokit()` so an omitted JavaScript argument falls back to the runtime token instead of behaving like a bogus call-argument token.
- Added CLI-side error formatting so wrapped `GoError` and GitHub API failures render as readable stderr blocks instead of exposing raw native wrapper text by default.
- Switched the `run` subcommand off the Glazed `cobra.CheckErr()` path so formatted errors reach the terminal cleanly.
- Updated `permissions-audit.js` to skip the `selected-actions` endpoint unless `allowed_actions == "selected"`.
- Updated `permissions-audit.js` to treat missing local runner output/summary files as best-effort status reported in JSON instead of a fatal error.

## 2026-03-11

Opened a debugging and observability ticket after confirming that root logging support already exists but subsystem-specific logs are missing.

### Related Files

- /home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-3--improve-goja-gha-debugging-and-runtime-logging/design-doc/01-debugging-and-logging-design-for-goja-gha.md — Primary design document for the new ticket
