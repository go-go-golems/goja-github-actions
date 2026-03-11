---
Title: "CLI Middleware And Env Resolution"
Slug: "cli-middleware-and-env-resolution"
Short: "Understand how pkg/cli/middleware.go builds the current parse pipeline, why runner env works, and how to inspect precedence with --print-parsed-fields."
Topics:
- goja
- github-actions
- javascript
- glazed
Commands:
- run
Flags:
- print-parsed-fields
- config-file
- github-token
- workspace
- debug
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page explains how `pkg/cli/middleware.go` works today. It is the right page to read when you want to answer questions like "why did `GITHUB_TOKEN` become `github-actions.github-token`?", "why did config beat the environment here?", or "what does `--print-parsed-fields` actually show me?".

The short version is that `goja-gha` uses Glazed as the parsing engine, but it does not use Glazed's default env pipeline. Instead, `pkg/cli/middleware.go` installs a custom middleware list that translates selected GitHub runner environment variables into Glazed field values before settings are decoded.

## Why This File Exists

`goja-gha` needs to support two naming worlds at the same time:

- Glazed fields such as `event-path`, `debug`, `workspace`, and `github-token`
- GitHub runner environment variables such as `GITHUB_EVENT_PATH`, `RUNNER_DEBUG`, `GITHUB_WORKSPACE`, and `GITHUB_TOKEN`

Without a bridge, those GitHub-specific names would not automatically land in the Glazed settings structs. `pkg/cli/middleware.go` is that bridge.

Why this matters in practice:

- it defines the current precedence rules,
- it explains why raw `GITHUB_*` variables work without any `GOJA_GHA_*` prefix,
- it is the file to inspect first when `--print-parsed-fields` shows an unexpected source.

## The Entry Point

The file starts with `NewParserConfig()`, which returns a `glazedcli.CobraParserConfig`:

```go
func NewParserConfig() glazedcli.CobraParserConfig {
    return glazedcli.CobraParserConfig{
        AppName:           AppName,
        MiddlewaresFunc:   NewMiddlewaresFunc(os.LookupEnv),
        ShortHelpSections: []string{},
    }
}
```

The most important detail is `MiddlewaresFunc`. That field overrides Glazed's default middleware chain. So even though `AppName` is set to `goja-gha`, the command is not using the normal built-in `FromEnv("GOJA_GHA")` path. It is using the custom chain built by `NewMiddlewaresFunc(...)`.

## The Middleware List

`NewMiddlewaresFunc(...)` returns this list:

```go
return []sources.Middleware{
    sources.FromCobra(cmd, fields.WithSource("cobra")),
    sources.FromArgs(args, fields.WithSource("arguments")),
    sources.FromFiles(configFiles, ...),
    sources.FromMap(RunnerEnvValuesFromLookup(lookupEnv), fields.WithSource(SourceRunnerEnv)),
    sources.FromMapAsDefault(DefaultFieldValues(), fields.WithSource(SourceDefaults)),
    sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
}, nil
```

That list is the entire policy for where values come from.

## The Important Rule: Glazed Executes Middlewares In Reverse Order

This is the part that usually causes confusion the first time you read the file.

Glazed middleware composition means the list is applied in reverse execution order. So the item that appears earlier in the list wins over the items that appear later in the list.

For this specific file, that means:

```text
listed first                               highest precedence
  FromCobra
  FromArgs
  FromFiles
  FromMap(RunnerEnvValuesFromLookup)
  FromMapAsDefault(DefaultFieldValues)
  FromDefaults
listed last                                lowest precedence
```

And because execution is reversed internally, the effective precedence becomes:

```text
lowest precedence
  schema defaults
  app-supplied defaults from DefaultFieldValues()
  mapped runner env from RunnerEnvValuesFromLookup()
  config files
  positional args
  cobra flags
highest precedence
```

That ordering is not accidental. It comes directly from how the list is written in `pkg/cli/middleware.go`.

This also explains the subtle point you need to remember while debugging:

- config files come before runner env in the list,
- so config files run after runner env in reverse execution order,
- so config files override runner env in the current implementation.

That is current `goja-gha` behavior. It is not a generic Glazed rule.

## What Each Middleware Does

This section explains what each middleware contributes and why the chain is structured this way.

### `FromDefaults`

This reads default values declared directly on field definitions. For example, `debug` and `json-result` default to `false`.

Why it matters:

- it gives a stable baseline for fields that define schema defaults,
- it runs at the bottom of the precedence stack.

### `FromMapAsDefault(DefaultFieldValues())`

This injects app-owned defaults that are not declared on the field definitions themselves. In the current code, the main example is `cwd: "."`.

Why it matters:

- it lets the application supply defaults outside the field definition,
- it still behaves like a low-precedence fallback.

One small implementation detail is worth knowing: in current `--print-parsed-fields` output, `cwd` may appear with source `none` rather than a custom source label. That comes from how this default is merged today, not from Cobra or the environment.

### `FromMap(RunnerEnvValuesFromLookup(...))`

This is the custom GitHub bridge. It looks up selected environment variables, translates them into section/field names, and injects them as parsed values.

Today it maps, among others:

- `GITHUB_EVENT_PATH -> default.event-path`
- `GITHUB_ACTION_PATH -> default.action-path`
- `GITHUB_ENV -> default.runner-env-file`
- `GITHUB_OUTPUT -> default.runner-output-file`
- `GITHUB_PATH -> default.runner-path-file`
- `GITHUB_STEP_SUMMARY -> default.runner-summary-file`
- `RUNNER_DEBUG -> default.debug`
- `GITHUB_WORKSPACE -> github-actions.workspace`
- `GITHUB_TOKEN -> github-actions.github-token`
- `GH_TOKEN -> github-actions.github-token`

Why it matters:

- it is the reason GitHub runner variables work without special CLI flags,
- it is the layer that normalizes booleans like `RUNNER_DEBUG=1` into `true`,
- it is the main thing that makes this parser more complex than a default Glazed command.

### `FromFiles(configFiles, ...)`

This loads the resolved config files and merges them as higher-precedence values than the mapped runner environment.

Why it matters:

- a config file can override `GITHUB_TOKEN` and `GITHUB_WORKSPACE` in the current implementation,
- that is often surprising until you remember the reverse execution rule.

### `FromArgs`

This reads positional arguments if the command defines them.

Why it matters:

- in `run`, most important values are flags rather than positional args,
- but it still sits above config and env in the stack.

### `FromCobra`

This reads normal Cobra flags such as `--script`, `--github-token`, `--workspace`, and `--debug`.

Why it matters:

- flags are the highest-precedence source in the current chain,
- this is why explicit CLI overrides beat both config and runner env.

## How The Runner Env Mapping Is Built

The custom env bridge lives in two helper functions.

### `RunnerEnvValuesFromLookup`

This function loops through the mappings from `pkg/cli/defaults.go`, checks whether an env var exists, and builds a Glazed-shaped nested map:

```text
map[sectionSlug]map[fieldName]value
```

Conceptually:

```go
valuesBySection := map[string]map[string]interface{}{}

for _, mapping := range RunnerEnvMappings() {
    for _, envKey := range mapping.EnvKeys {
        value, ok := lookupEnv(envKey)
        if !ok || strings.TrimSpace(value) == "" {
            continue
        }

        valuesBySection[mapping.SectionSlug][mapping.FieldName] =
            normalizeEnvValue(mapping.FieldName, value)
        break
    }
}
```

Why it matters:

- it converts raw process env into something Glazed can merge,
- it supports fallbacks like `GITHUB_TOKEN` then `GH_TOKEN`,
- it ensures only the first matching env key wins for a mapped field.

### `normalizeEnvValue`

This normalizes special cases before the values are injected. Right now the most important special case is `debug`:

- `"1"`, `"true"`, `"yes"`, and `"on"` become `true`
- other strings stay strings for non-boolean fields

Why it matters:

- it explains why `RUNNER_DEBUG=1` becomes a real boolean in parsed output instead of the literal string `"1"`.

## `ResolveConfigFiles`

The config file part of the parser is much simpler than the env mapping part.

`ResolveConfigFiles(...)` decodes the standard Glazed command settings section and then calls `glazedconfig.ResolveAppConfigPath(AppName, commandSettings.ConfigFile)`.

In practice that means:

- if `--config-file` is set, it uses that,
- otherwise it asks Glazed to resolve the application config path for `goja-gha`,
- the resolved file list is then passed into `FromFiles(...)`.

So this file is not reimplementing config discovery from scratch. It is mostly reimplementing env mapping.

## How To Read `--print-parsed-fields`

`--print-parsed-fields` is the fastest way to debug this parser. It shows:

- the final value,
- the ordered log of where that value came from,
- some source-specific metadata.

For each field you usually care about three things:

1. `value`
2. `log[].source`
3. `log[].metadata`

Common sources you will see:

| Source | Meaning |
|---|---|
| `defaults` | schema-defined field default |
| `runner-env` | mapped GitHub runner env injected by `RunnerEnvValuesFromLookup()` |
| `config` | config file value loaded by `FromFiles(...)` |
| `arguments` | positional args |
| `cobra` | explicit CLI flag |
| `none` | current fallback/default merge with no stronger source annotation |

Why the log order matters:

- it shows all contributing writes, not just the winner,
- the last applicable higher-precedence write determines the final `value`.

## Real-Life Examples

All examples below were validated against the current implementation. They use `yq` to extract just the relevant part of `--print-parsed-fields` output so you do not have to scan the full YAML dump.

### Example 1: Runner env populates GitHub settings

This example shows that `GITHUB_TOKEN` and `GITHUB_WORKSPACE` are not read by Cobra or a default Glazed env prefix. They are mapped by the custom runner-env middleware.

```bash
GOWORK=off GITHUB_TOKEN=abc123 GITHUB_WORKSPACE=/tmp/ws \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --print-parsed-fields |
  yq '{"workspace": .["github-actions"].workspace.value, "token": .["github-actions"]["github-token"].value, "token_sources": [.["github-actions"]["github-token"].log[].source]}'
```

Expected output:

```yaml
workspace: /tmp/ws
token: abc123
token_sources:
  - runner-env
```

What this proves:

- `GITHUB_WORKSPACE` becomes `github-actions.workspace`
- `GITHUB_TOKEN` becomes `github-actions.github-token`
- the source is `runner-env`, not `cobra` and not default Glazed `env`

### Example 2: Config beats runner env in the current implementation

Create a config file like this:

```yaml
default:
  cwd: /tmp/from-config
github-actions:
  github-token: from-config
```

Then run:

```bash
GOWORK=off GITHUB_TOKEN=from-env \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --config-file ./ttmp/2026/03/11/GHA-2--move-goja-gha-settings-resolution-fully-into-glazed-sources/scripts/middleware-example-config.yaml \
    --print-parsed-fields |
  yq '{"value": .["github-actions"]["github-token"].value, "sources": [.["github-actions"]["github-token"].log[].source]}'
```

Expected output:

```yaml
value: from-config
sources:
  - runner-env
  - config
```

What this proves:

- both sources contributed,
- config won,
- the reason is middleware list order plus reverse execution order.

This is the easiest example to use when you want to confirm that current `goja-gha` precedence is file-specific policy rather than generic Glazed default behavior.

### Example 3: Flag beats runner env

```bash
GOWORK=off GITHUB_TOKEN=from-env \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --github-token from-flag \
    --print-parsed-fields |
  yq '{"value": .["github-actions"]["github-token"].value, "sources": [.["github-actions"]["github-token"].log[].source]}'
```

Expected output:

```yaml
value: from-flag
sources:
  - runner-env
  - cobra
```

What this proves:

- runner env set an initial value,
- Cobra flag overwrote it,
- flags are the highest-precedence source in the current chain.

### Example 4: `RUNNER_DEBUG=1` becomes a boolean

```bash
GOWORK=off RUNNER_DEBUG=1 \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --print-parsed-fields |
  yq '{"value": .default.debug.value, "sources": [.default.debug.log[].source], "mapped_values": [.default.debug.log[].metadata."map-value"]}'
```

Expected output:

```yaml
value: true
sources:
  - defaults
  - runner-env
mapped_values:
  - null
  - true
```

What this proves:

- the field starts from the schema default `false`,
- the runner env layer overwrites it,
- normalization happens before merge, so the mapped value is real boolean `true`.

## A Concrete Mental Model

If you want one reliable way to think about this file, use this sequence:

```text
1. Build a middleware list
2. Remember Glazed executes it in reverse order
3. Ask which source wrote the field first
4. Ask which later, higher-precedence source overwrote it
5. Confirm with --print-parsed-fields
```

That mental model is enough to explain almost every current parser surprise.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `GITHUB_TOKEN` is ignored | the variable is empty or unset | run with `env | grep GITHUB_TOKEN` or inspect `--print-parsed-fields` |
| config beats env and you expected the opposite | `FromFiles(...)` appears before runner env in the middleware list, so it executes after it | inspect the middleware order in `pkg/cli/middleware.go` and confirm with Example 2 |
| `RUNNER_DEBUG=1` does not show `"1"` | the value is normalized to boolean `true` | inspect `.default.debug` rather than expecting a raw string |
| `cwd` shows source `none` | current app-default merge does not annotate it the same way as stronger sources | treat it as a low-precedence fallback, not as a flag or env write |
| full parsed output is too noisy | Glazed sections include many output-formatting fields | pipe through `yq` and extract only the field you care about |

## See Also

- `goja-gha help developer-guide` for the broader architecture and package map
- `goja-gha help design-and-internals` for deeper runtime and binding decisions
- `goja-gha help user-guide` for end-user command usage instead of parser internals
