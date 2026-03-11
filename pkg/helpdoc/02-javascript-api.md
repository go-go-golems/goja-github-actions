---
Title: "JavaScript API"
Slug: "javascript-api"
Short: "Reference for process globals and the @actions/core, @actions/github, @actions/io, @actions/exec, and @goja-gha/ui modules exposed by goja-gha."
Topics:
- goja
- github-actions
- javascript
Commands:
- run
Flags:
- script
- json-result
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This reference explains the JavaScript surface area exposed by `goja-gha`. It is written for script authors first, but it also helps reviewers verify what the runtime promises and what it intentionally does not promise.

The most important rule is that the API surface is intentionally narrow. `goja-gha` exposes the pieces needed for GitHub automation, not a general Node standard library or npm-compatible ecosystem. When you see a missing API, assume "not implemented yet" rather than "available because Node would have it."

## Runtime Globals

This section covers the globals that exist before you require any module. They are implemented primarily in `pkg/runtime/globals.go`.

### `process.env`

`process.env` starts from the resolved runtime environment, which is built from:

- ambient environment variables,
- decoded workspace and runner settings,
- local mutations from `core.exportVariable()` and `core.addPath()`.

Why this matters: `process.env` is not just a passive view of the host process. It is part of the runtime state and can change during script execution.

### `process.cwd()`

Returns the resolved working directory. This is the same value the runtime uses for relative file operations and the default subprocess cwd.

### `process.stdout.write(value)` and `process.stderr.write(value)`

Write directly to the CLI process streams.

### `process.exitCode`

Holds the exit code requested by the script. `@actions/core.setFailed()` sets this to `1`.

## `@actions/core`

This section covers the highest-value GitHub Actions primitives. The implementation lives in `pkg/modules/core`.

### Inputs

Available functions:

| Function | Behavior |
|---|---|
| `getInput(name, options?)` | Reads `INPUT_<NAME>` and trims surrounding whitespace |
| `getBooleanInput(name, options?)` | Parses GitHub-style boolean inputs |
| `getMultilineInput(name, options?)` | Splits a multiline input into an array |

Example:

```js
const core = require("@actions/core");

module.exports = function () {
  const name = core.getInput("name", { required: true });
  const dryRun = core.getBooleanInput("dry-run");
  const paths = core.getMultilineInput("paths");
  return { name, dryRun, paths };
};
```

### Outputs And Environment Mutation

Available functions:

| Function | Behavior |
|---|---|
| `setOutput(name, value)` | Appends to the resolved runner output file |
| `exportVariable(name, value)` | Appends to the runner env file and updates runtime `process.env` |
| `addPath(path)` | Appends to the runner path file and prepends to runtime `PATH` |

Why this matters: local behavior and runner behavior stay aligned enough that one script can usually be debugged in both places.

### Logging And Workflow Commands

Available functions:

- `debug(message)`
- `info(message)`
- `notice(message)`
- `warning(message)`
- `error(message)`
- `setSecret(value)`
- `startGroup(name)`
- `endGroup()`
- `group(name, fn)`
- `setFailed(message)`

`setFailed(message)` has two effects:

1. it records a failure message in runtime state,
2. it sets `process.exitCode = 1`.

### Step Summaries

`core.summary` is a builder object. Current methods:

- `addRaw(text)`
- `addHeading(text)`
- `write()`
- `clear()`

Example:

```js
core.summary
  .addHeading("Audit Result")
  .addRaw("All required workflows are present.\n")
  .write();
```

## `@actions/github`

This section covers the runtime GitHub context and the first GitHub REST helpers. The implementation lives in `pkg/contextdata`, `pkg/githubapi`, and `pkg/modules/github`.

### `github.context`

`github.context` is a plain JavaScript object normalized from runtime settings, runner env, and the optional event payload file.

Important fields:

| Field | Source |
|---|---|
| `actor` | `GITHUB_ACTOR` |
| `event_name` | `GITHUB_EVENT_NAME` |
| `event_path` | resolved event-path setting |
| `repository` | `GITHUB_REPOSITORY` or event payload |
| `sha` | `GITHUB_SHA` |
| `workspace` | resolved workspace setting |
| `repo.owner` / `repo.repo` | parsed from repository or event payload |
| `payload` | decoded event JSON |

### `getOctokit(token?, options?)`

Creates the GitHub client object. If `token` is omitted, `undefined`, `null`, or an empty string, the runtime falls back to the resolved `github-token`. If `options.baseUrl` is empty, the runtime falls back to `GITHUB_API_URL` or `https://api.github.com`.

Example:

```js
const github = require("@actions/github");

module.exports = function () {
  const octokit = github.getOctokit();
  return octokit.request("GET /repos/{owner}/{repo}/actions/workflows", {
    owner: github.context.repo.owner,
    repo: github.context.repo.repo
  });
};
```

Why this matters: script authors can usually call `github.getOctokit()` with no arguments during local and runner-backed executions, as long as the runtime has already resolved a token. That fallback path was explicitly validated during debugging work on `permissions-audit.js`.

### Generic Request Helpers

Available methods:

- `octokit.request(route, params)`
- `octokit.paginate(route, params)`

Returned shape:

```js
{
  status: 200,
  data: ...,
  headers: {
    "Link": ["<...>; rel=\"next\""]
  }
}
```

If the API returns an error status, the runtime throws a Go-backed `APIError` with a human-readable message.

### Curated `rest.actions.*` Helpers

Available helpers:

- `octokit.rest.actions.getGithubActionsPermissionsRepository({ owner, repo })`
- `octokit.rest.actions.getAllowedActionsRepository({ owner, repo })`
- `octokit.rest.actions.getWorkflowPermissionsRepository({ owner, repo })`
- `octokit.rest.actions.listRepoWorkflows({ owner, repo })`

These helpers exist because the first real application is a permissions/workflow audit. They are intentionally thin wrappers over the generic request layer.

For `permissions-audit.js`, these helpers are not equivalent from a permissions standpoint:

- `listRepoWorkflows(...)` typically needs repository `Actions: Read`
- the repository permissions helpers also need repository `Administration: Read`

That means a fine-grained PAT can succeed on workflow listing and still fail on the permissions endpoints with `403 Resource not accessible by personal access token`.

One more behavioral detail matters for real scripts: `getAllowedActionsRepository(...)` is not universally applicable. GitHub returns `409 Conflict` when the repository policy is not `allowed_actions == "selected"`. The shipped `permissions-audit.js` example now fetches `getGithubActionsPermissionsRepository(...)` first, inspects `allowed_actions`, and skips `getAllowedActionsRepository(...)` when the policy mode is not `selected`.

## `@goja-gha/ui`

This section covers the report builder module implemented in `pkg/modules/ui`.

The purpose of `@goja-gha/ui` is to let scripts describe human-readable terminal output declaratively instead of manually formatting strings. It is intentionally small in v1.

Top-level module functions:

- `ui.report(title)`
- `ui.enabled()`

Report-builder methods:

- `status(kind, text)`
- `success(text)`
- `note(text)`
- `warn(text)`
- `error(text)`
- `kv(label, value)`
- `list(items)`
- `table({ columns, rows })`
- `section(title, fn)`
- `render()`

Example:

```js
const ui = require("@goja-gha/ui");

module.exports = function () {
  ui.report("Audit")
    .status("ok", "Inspection complete")
    .kv("Repository", "acme/widgets")
    .section("Workflows", (section) => {
      section.table({
        columns: ["Name", "Path"],
        rows: [["CI", ".github/workflows/ci.yml"]]
      });
    })
    .render();

  return { ok: true };
};
```

Why this matters: the renderer coordinates with the CLI. In normal runs it emits a human report and suppresses the automatic returned-object print. In `--json-result` runs it becomes a no-op so JSON output stays machine-readable.

For the full DSL reference, use:

- `goja-gha help js-report-dsl-api`

## `@actions/io`

This section covers the file-system helper module implemented in `pkg/modules/io`.

Available functions:

| Function | Behavior |
|---|---|
| `readdir(path)` | Returns sorted entry names |
| `readFile(path)` | Returns file contents as a string |
| `writeFile(path, content)` | Creates parents and writes text |
| `mkdirP(path)` | Creates directories recursively |
| `rmRF(path)` | Removes a path recursively |
| `cp(src, dst, options?)` | Copies files or directories |
| `mv(src, dst)` | Moves or renames a path |
| `which(tool, check)` | Resolves a binary through `PATH` |

Paths resolve relative to the runtime working directory unless they are already absolute.

Example:

```js
const io = require("@actions/io");

module.exports = function () {
  io.mkdirP(".cache/results");
  io.writeFile(".cache/results/report.txt", "ok\n");
  return io.readdir(".cache/results");
};
```

## `@actions/exec`

This section covers asynchronous subprocess execution, which is the sharpest part of the JS API because it has to cross the Goja owner-thread boundary safely.

Available function:

- `exec(command, args?, options?)`

Returned value:

- a Promise that resolves to `{ exitCode, stdout, stderr }`

Supported options:

| Option | Meaning |
|---|---|
| `cwd` | override working directory |
| `env` | environment overrides |
| `ignoreReturnCode` | resolve instead of reject on non-zero exit |
| `silent` | suppress streaming to the CLI stdout/stderr |
| `captureOutput` | reserved for compatibility; output is currently always collected |
| `listeners.stdout` / `listeners.stderr` | callback hooks for streamed chunks |

Example:

```js
const ghaExec = require("@actions/exec");

module.exports = async function () {
  const result = await ghaExec.exec("go", ["env", "GOOS"], {
    silent: true
  });

  return {
    exitCode: result.exitCode,
    stdout: result.stdout.trim()
  };
};
```

Why this matters: the Promise resolves on the Goja owner thread through `runtimeowner.Runner.Post(...)`. That is what makes JS callbacks and Promise settlement safe.

## Common Patterns

This section shows how the modules are meant to combine in real scripts.

### Pattern: Read Inputs, Call GitHub, Emit Output

```js
const core = require("@actions/core");
const github = require("@actions/github");

module.exports = function () {
  const owner = core.getInput("owner") || github.context.repo.owner;
  const repo = core.getInput("repo") || github.context.repo.repo;
  const octokit = github.getOctokit();
  const workflows = octokit.rest.actions.listRepoWorkflows({ owner, repo }).data;

  core.setOutput("workflow-count", String(workflows.total_count));
  return workflows;
};
```

### Pattern: Inspect Local Files And Run A Helper Command

```js
const io = require("@actions/io");
const ghaExec = require("@actions/exec");

module.exports = async function () {
  const files = io.readdir(".github/workflows");
  const git = await ghaExec.exec("git", ["status", "--short"], { silent: true });
  return { files, gitStatus: git.stdout };
};
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `require("@actions/...")` fails | Module name is wrong or module was not registered | Use the exact module names documented here |
| `github.context.repo.owner` is empty | Repository metadata did not resolve from env or payload | Set `GITHUB_REPOSITORY` or provide a realistic event payload |
| `octokit.request(...)` throws `401 Bad credentials` | The token is missing, invalid, expired, or not what the runtime used | Run with `--log-level debug --log-format text` and inspect the `Creating Octokit client` and `Received GitHub API response` lines |
| `octokit.request(...)` throws `403 Resource not accessible by personal access token` | The token is valid but under-scoped for the endpoint | For repository Actions permissions endpoints, add fine-grained PAT repo permissions `Actions: Read` and `Administration: Read` |
| `octokit.request(...)` throws another API error | Route, auth, or base URL is wrong | Check token, route placeholders, `GITHUB_API_URL`, and the debug request logs |
| `io.readdir(...)` fails on a relative path | The working directory is not what you expect | Confirm `process.cwd()` or pass `--cwd` |
| `exec(...)` rejects | The command failed or the binary was not found | Re-run with `silent: false` or inspect `stderr` |

## See Also

- `goja-gha help user-guide`
- `goja-gha help developer-guide`
- `goja-gha help design-and-internals`
