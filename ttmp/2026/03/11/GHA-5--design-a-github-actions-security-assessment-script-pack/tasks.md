# Tasks

## Phase 0: Analysis and ticket scaffolding

- [x] Create `GHA-5` ticket workspace.
- [x] Create the primary design doc and investigation diary.
- [x] Extract the candidate script list from the imported GHA-1 planning notes.
- [x] Document the current runtime surface that security scripts can use today.
- [x] Add a ticket-scoped validation script for `/tmp/geppetto`.

## Phase 1: Stabilize the baseline audit

- [ ] Promote `examples/permissions-audit.js` from “example” to baseline security audit semantics.
- [ ] Add explicit severity evaluation to the permissions audit result.
- [ ] Return a normalized `findings` array from the permissions audit.
- [ ] Add remediation text for weak settings such as `allowed_actions=all`.
- [ ] Decide whether to split repository and organization settings checks into separate scripts now or later.

## Phase 2: Build the first local workflow lint scripts

- [ ] Implement `pin-third-party-actions.js`.
- [ ] Implement `checkout-persist-creds.js`.
- [ ] Implement `no-write-all.js`.
- [ ] Implement `pull-request-target-review.js`.
- [ ] Implement `workflow-run-review.js`.
- [ ] Add fixture workflows covering safe and unsafe patterns for each script.
- [ ] Add CLI integration tests for each new script.

## Phase 3: Introduce shared helpers

- [ ] Add a shared workflow file discovery helper for scripts.
- [ ] Add a shared finding/result builder helper.
- [ ] Add a shared report renderer helper built on `@goja-gha/ui`.
- [ ] Decide whether YAML parsing should remain JS-side or move into a Go-native helper module.
- [ ] If needed, add a native workflow/YAML helper module.

## Phase 4: Settings policy specialization

- [ ] Implement `restricted-default-token.js`.
- [ ] Implement `org-selected-actions.js`.
- [ ] Implement `org-fork-approval.js`.
- [ ] Decide whether organization-wide inventory belongs in script land or a dedicated CLI command.

## Phase 5: Advanced trust-boundary analysis

- [ ] Implement `no-privileged-untrusted-checkout.js`.
- [ ] Implement `reusable-workflow-trust.js`.
- [ ] Implement `no-artifact-bridge.js`.
- [ ] Implement `no-cache-bridge.js`.
- [ ] Implement a trust-label model for privileged, untrusted, and external-input nodes.
- [ ] Add evidence-rich finding output for cross-workflow/dataflow cases.

## Validation and rollout

- [x] Run the baseline validation script against `/tmp/geppetto` and capture JSON and report outputs in the ticket `scripts/` directory.
- [ ] Keep `/tmp/geppetto` as a live smoke target for the baseline audit and future lint scripts.
- [ ] Validate the first lint scripts against fixture repos and a real repo checkout.
- [ ] Document required token scopes for each API-backed script.
- [ ] Add user-facing help/docs once the first three or four scripts exist.
- [ ] Decide when to move scripts from `examples/` into a dedicated `policies/` directory.

## Exit criteria for the first useful release

- [ ] Repository settings audit returns structured findings and a readable report.
- [ ] At least three local workflow lint scripts exist and pass fixture-based tests.
- [ ] Scripts share a consistent JSON result contract.
- [ ] A new engineer can run the scripts against `/tmp/geppetto` without guessing hidden prerequisites.
