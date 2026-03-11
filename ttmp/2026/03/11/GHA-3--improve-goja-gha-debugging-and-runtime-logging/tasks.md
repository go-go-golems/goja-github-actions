# Tasks

## Immediate

- [x] Document the existing root logging flags and usage.
- [x] Add structured debug logs around settings decode and runtime bootstrap.
- [x] Add GitHub HTTP request/response tracing with secret-safe fields.
- [x] Confirm and document JavaScript `console.*` behavior during local runs.
- [x] Improve error context around 401 and other GitHub API failures.
- [x] Format wrapped Goja/native runtime errors into cleaner CLI stderr output.
- [x] Make `permissions-audit.js` skip `selected-actions` when the repo policy is not `selected`.
- [x] Make `permissions-audit.js` treat missing local runner output/summary files as best-effort status instead of a fatal error.

## Later

- [x] Add a dedicated debugging help page.
- [ ] Add regression tests for masked logging and request tracing.
- [ ] Add one example script or playbook specifically for debugging GitHub API auth failures.
