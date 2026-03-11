# Tasks

## Completed

- [x] Design the `@goja-gha/ui` report DSL API and rendering rules.
- [x] Implement the `@goja-gha/ui` native module and wire it into the runtime.
- [x] Add runtime coordination so human report rendering suppresses the default returned-object print in non-JSON runs.
- [x] Migrate `examples/permissions-audit.js` to the report DSL.
- [x] Add module tests for report rendering and JSON-result suppression.
- [x] Add CLI integration coverage for human-readable `permissions-audit.js` output.
- [x] Add a dedicated Glazed help page for the DSL and update the broader JS API docs.

## Follow-up

- [ ] Consider whether `@goja-gha/ui` should support explicit multi-stream rendering such as `stderr` in future releases.
- [ ] Consider whether the renderer should grow a Markdown backend for step summaries or artifact generation.
