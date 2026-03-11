# goja-github-actions

`goja-github-actions` is the home of `goja-gha`, a Go CLI that will run GitHub Actions-oriented JavaScript on top of Goja.

The repository is currently in bootstrap mode. The first implementation slice provides:

- a real `goja-gha` binary entrypoint,
- Glazed/Cobra command wiring for `run` and `doctor`,
- default-section runner flags plus a shared GitHub settings section,
- cleaned-up module/build/release metadata.

## Current Status

The runtime itself is not implemented yet. The CLI foundation is being built task by task:

- `goja-gha run` currently validates and prints the resolved bootstrap settings, then reports that runtime execution is not implemented yet.
- `goja-gha doctor` reports the resolved settings through Glazed output so the schema and parser wiring can be exercised early.

## Development

Show the root help:

```bash
go run ./cmd/goja-gha --help
```

Inspect the `run` command:

```bash
go run ./cmd/goja-gha run --help
```

Inspect resolved schema values with structured output:

```bash
go run ./cmd/goja-gha doctor --script ./examples/permissions-audit.js --output json
```

## Roadmap

The implementation plan and detailed backlog live in:

- `ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/design-doc/01-goja-github-actions-design-and-implementation-guide.md`
- `ttmp/2026/03/10/GHA-1--create-goja-bindings-for-github-actions/tasks.md`
