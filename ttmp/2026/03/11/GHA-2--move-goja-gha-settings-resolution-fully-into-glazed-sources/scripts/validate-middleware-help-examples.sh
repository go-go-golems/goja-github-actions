#!/usr/bin/env bash
set -euo pipefail

ROOT="/home/manuel/workspaces/2026-03-10/gh-actions-goja-validation/goja-github-actions"
CONFIG="$ROOT/ttmp/2026/03/11/GHA-2--move-goja-gha-settings-resolution-fully-into-glazed-sources/scripts/middleware-example-config.yaml"

cd "$ROOT"

echo "== base case =="
GOWORK=off go run ./cmd/goja-gha run \
  --script ./examples/trivial.js \
  --print-parsed-fields |
  yq '.default.script.value'

echo
echo "== runner env mapping =="
GOWORK=off GITHUB_TOKEN=abc123 GITHUB_WORKSPACE=/tmp/ws \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --print-parsed-fields |
  yq '{"workspace": .["github-actions"].workspace.value, "token": .["github-actions"]["github-token"].value, "token_sources": [.["github-actions"]["github-token"].log[].source]}'

echo
echo "== config overrides runner env in current implementation =="
GOWORK=off GITHUB_TOKEN=from-env \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --config-file "$CONFIG" \
    --print-parsed-fields |
  yq '{"value": .["github-actions"]["github-token"].value, "sources": [.["github-actions"]["github-token"].log[].source]}'

echo
echo "== cobra flag overrides runner env =="
GOWORK=off GITHUB_TOKEN=from-env \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --github-token from-flag \
    --print-parsed-fields |
  yq '{"value": .["github-actions"]["github-token"].value, "sources": [.["github-actions"]["github-token"].log[].source]}'

echo
echo "== RUNNER_DEBUG normalization =="
GOWORK=off RUNNER_DEBUG=1 \
  go run ./cmd/goja-gha run \
    --script ./examples/trivial.js \
    --print-parsed-fields |
  yq '{"value": .default.debug.value, "sources": [.default.debug.log[].source], "mapped_values": [.default.debug.log[].metadata."map-value"]}'
