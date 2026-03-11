#!/usr/bin/env bash

set -euo pipefail

ROOT="/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions"
TICKET_DIR="$ROOT/ttmp/2026/03/11/GHA-7--add-go-native-github-actions-workflow-parsing-module"
OUTPUT_DIR="$TICKET_DIR/scripts"
WORKSPACE="/tmp/geppetto"
REPOSITORY="go-go-golems/geppetto"

if [[ ! -d "$WORKSPACE/.github/workflows" ]]; then
  echo "expected workflow directory at $WORKSPACE/.github/workflows" >&2
  exit 1
fi

if [[ -f "$ROOT/.envrc" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.envrc"
  set +a
fi

export GOWORK=off
export GITHUB_WORKSPACE="$WORKSPACE"
export GITHUB_REPOSITORY="$REPOSITORY"

cd "$ROOT"

go run ./cmd/goja-gha run \
  --script ./examples/list-workflows.js \
  --cwd /tmp \
  --workspace "$WORKSPACE" \
  --json-result \
  > "$OUTPUT_DIR/geppetto-list-workflows.json"

go run ./cmd/goja-gha run \
  --script ./examples/pin-third-party-actions.js \
  --cwd /tmp \
  --workspace "$WORKSPACE" \
  --json-result \
  > "$OUTPUT_DIR/geppetto-pin-third-party-actions.json"

go run ./cmd/goja-gha run \
  --script ./examples/checkout-persist-creds.js \
  --cwd /tmp \
  --workspace "$WORKSPACE" \
  --json-result \
  > "$OUTPUT_DIR/geppetto-checkout-persist-creds.json"

go run ./cmd/goja-gha run \
  --script ./examples/no-write-all.js \
  --cwd /tmp \
  --workspace "$WORKSPACE" \
  --json-result \
  > "$OUTPUT_DIR/geppetto-no-write-all.json"

go run ./cmd/goja-gha run \
  --script ./examples/permissions-audit.js \
  --cwd /tmp \
  --workspace "$WORKSPACE" \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result \
  > "$OUTPUT_DIR/geppetto-permissions-audit.json"

jq -n \
  --arg workspace "$WORKSPACE" \
  --arg repository "$REPOSITORY" \
  --slurpfile list "$OUTPUT_DIR/geppetto-list-workflows.json" \
  --slurpfile pin "$OUTPUT_DIR/geppetto-pin-third-party-actions.json" \
  --slurpfile checkout "$OUTPUT_DIR/geppetto-checkout-persist-creds.json" \
  --slurpfile writeall "$OUTPUT_DIR/geppetto-no-write-all.json" \
  --slurpfile audit "$OUTPUT_DIR/geppetto-permissions-audit.json" \
  '{
    workspace: $workspace,
    repository: $repository,
    workflowFiles: ($list[0].workflowFiles | length),
    pinThirdPartyActions: $pin[0].summary,
    checkoutPersistCreds: $checkout[0].summary,
    noWriteAll: $writeall[0].summary,
    permissionsAudit: $audit[0].summary
  }' \
  > "$OUTPUT_DIR/geppetto-workflow-script-summary.json"

jq -r '
  [
    "Repository: \(.repository)",
    "Workspace: \(.workspace)",
    "Workflow files: \(.workflowFiles)",
    "pin-third-party-actions: \(.pinThirdPartyActions.findingCount) findings (\(.pinThirdPartyActions.highestSeverity // "none"))",
    "checkout-persist-creds: \(.checkoutPersistCreds.findingCount) findings (\(.checkoutPersistCreds.highestSeverity // "none"))",
    "no-write-all: \(.noWriteAll.findingCount) findings (\(.noWriteAll.highestSeverity // "none"))",
    "permissions-audit: \(.permissionsAudit.findingCount) findings (\(.permissionsAudit.highestSeverity // "none"))"
  ] | .[]
' "$OUTPUT_DIR/geppetto-workflow-script-summary.json" \
  > "$OUTPUT_DIR/geppetto-workflow-script-summary.txt"

cat "$OUTPUT_DIR/geppetto-workflow-script-summary.txt"
