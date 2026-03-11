#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions"
TICKET_DIR="/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions/ttmp/2026/03/11/GHA-5--design-a-github-actions-security-assessment-script-pack"
SCRIPTS_DIR="$TICKET_DIR/scripts"
REPO_DIR="${1:-/tmp/geppetto}"
JSON_OUT="$SCRIPTS_DIR/geppetto-permissions-audit.json"
REPORT_OUT="$SCRIPTS_DIR/geppetto-permissions-audit.txt"
PR_TARGET_JSON_OUT="$SCRIPTS_DIR/geppetto-pull-request-target-review.json"

if [[ ! -d "$REPO_DIR" ]]; then
  echo "repository checkout not found: $REPO_DIR" >&2
  exit 1
fi

cd "$ROOT"
source "$ROOT/.envrc"

export GITHUB_REPOSITORY="go-go-golems/geppetto"
export GITHUB_WORKSPACE="$REPO_DIR"

GOWORK=off go run ./cmd/goja-gha run \
  --script ./examples/permissions-audit.js \
  --cwd "$REPO_DIR" \
  --workspace "$REPO_DIR" \
  --event-path ./testdata/events/workflow_dispatch.json \
  --json-result >"$JSON_OUT"

GOWORK=off go run ./cmd/goja-gha run \
  --script ./examples/permissions-audit.js \
  --cwd "$REPO_DIR" \
  --workspace "$REPO_DIR" \
  --event-path ./testdata/events/workflow_dispatch.json >"$REPORT_OUT"

GOWORK=off go run ./cmd/goja-gha run \
  --script ./examples/pull-request-target-review.js \
  --cwd "$REPO_DIR" \
  --workspace "$REPO_DIR" \
  --json-result >"$PR_TARGET_JSON_OUT"

jq '{
  scriptId,
  repository,
  workspace,
  summary,
  findings,
  workflowCount,
  localWorkflowFiles,
  selectedActionsStatus,
  allowedActions: .permissions.allowed_actions,
  defaultWorkflowPermissions: .workflowPermissions.default_workflow_permissions
}' "$JSON_OUT"

jq '{
  scriptId,
  repository,
  workspace,
  reviewedWorkflowCount,
  summary,
  findings
}' "$PR_TARGET_JSON_OUT"
