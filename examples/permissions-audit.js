const core = require("@actions/core");
const github = require("@actions/github");
const io = require("@actions/io");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function resolveSelectedActions(octokit, owner, repo, permissions) {
  const allowedActions = permissions && permissions.allowed_actions;
  if (allowedActions !== "selected") {
    return {
      data: null,
      status: "skipped-not-selected-policy",
      reason: `selected-actions only applies when allowed_actions == "selected" (got ${allowedActions || "unknown"})`
    };
  }

  return {
    data: octokit.rest.actions.getAllowedActionsRepository({ owner, repo }).data,
    status: "fetched",
    reason: null
  };
}

function trySetAuditOutput(result) {
  try {
    core.setOutput("audit", JSON.stringify(result));
    return { written: true, error: null };
  } catch (err) {
    return { written: false, error: err.message || String(err) };
  }
}

function tryWriteAuditSummary(result) {
  try {
    core.summary.addHeading("GitHub Actions Audit").addRaw(`${JSON.stringify(result, null, 2)}\n`).write();
    return { written: true, error: null };
  } catch (err) {
    return { written: false, error: err.message || String(err) };
  }
}

function buildFindings(result) {
  const auditFindings = [];

  if (result.permissions.enabled === false) {
    auditFindings.push(findings.makeFinding({
      ruleId: "actions-disabled",
      severity: "info",
      message: "GitHub Actions is disabled for this repository.",
      whyItMatters: "A disabled Actions setup may be intentional, but it changes the meaning of other repository policy settings.",
      evidence: {
        enabled: result.permissions.enabled
      },
      remediation: {
        summary: "Confirm that Actions is intentionally disabled before treating other policy findings as actionable."
      }
    }));
  }

  if (result.permissions.allowed_actions === "all") {
    auditFindings.push(findings.makeFinding({
      ruleId: "allowed-actions-not-restricted",
      severity: "high",
      message: "Repository allows all GitHub Actions and reusable workflows.",
      whyItMatters: "Allowing all actions increases supply-chain risk because workflows can consume mutable third-party actions without an allowlist boundary.",
      evidence: {
        allowed_actions: result.permissions.allowed_actions
      },
      remediation: {
        summary: "Prefer `selected` with an explicit allowlist, or `local_only` if the repository only needs local actions.",
        example: "Set repository Actions permissions so `allowed_actions` is `selected` and configure `selected-actions` patterns."
      }
    }));
  }

  if (result.permissions.sha_pinning_required === false) {
    auditFindings.push(findings.makeFinding({
      ruleId: "sha-pinning-not-required",
      severity: "medium",
      message: "Repository does not require full commit SHA pinning for actions.",
      whyItMatters: "Without SHA pinning requirements, mutable tags such as `@v1` can still be used in workflows.",
      evidence: {
        sha_pinning_required: result.permissions.sha_pinning_required
      },
      remediation: {
        summary: "Enable SHA pinning requirements if your policy is to require full commit SHAs for external actions."
      }
    }));
  }

  if (result.workflowPermissions.default_workflow_permissions !== "read") {
    auditFindings.push(findings.makeFinding({
      ruleId: "default-token-not-read-only",
      severity: "high",
      message: "Default workflow token permissions are broader than read-only.",
      whyItMatters: "Broader default token permissions increase the blast radius of compromised workflows and overly broad job defaults.",
      evidence: {
        default_workflow_permissions: result.workflowPermissions.default_workflow_permissions
      },
      remediation: {
        summary: "Set default workflow permissions to `read` and only elevate individual workflows or jobs when necessary."
      }
    }));
  }

  if (result.workflowPermissions.can_approve_pull_request_reviews === true) {
    auditFindings.push(findings.makeFinding({
      ruleId: "actions-can-approve-prs",
      severity: "medium",
      message: "GitHub Actions is allowed to approve pull request reviews.",
      whyItMatters: "Allowing workflow tokens to approve pull requests can widen the impact of automation abuse or compromised workflow executions.",
      evidence: {
        can_approve_pull_request_reviews: result.workflowPermissions.can_approve_pull_request_reviews
      },
      remediation: {
        summary: "Disable workflow approval of pull requests unless there is a well-reviewed and narrowly scoped use case."
      }
    }));
  }

  if (result.permissions.allowed_actions === "selected" && result.selectedActionsStatus === "fetched") {
    const patterns = (result.selectedActions && result.selectedActions.patterns_allowed) || [];
    if (!patterns.length) {
      auditFindings.push(findings.makeFinding({
        ruleId: "selected-actions-empty-allowlist",
        severity: "medium",
        message: "Selected-actions policy is enabled, but the allowlist is empty.",
        whyItMatters: "An empty allowlist may be intentional, but it often means the policy is incomplete and will block or confuse workflow usage.",
        evidence: {
          patterns_allowed: patterns
        },
        remediation: {
          summary: "Populate `patterns_allowed` with the exact actions and reusable workflows your repository trusts."
        }
      }));
    }
  }

  return auditFindings;
}

function renderAuditReport(result) {
  const overallStatus = result.summary.status === "passed"
    ? "ok"
    : result.summary.highestSeverity === "high" || result.summary.highestSeverity === "critical"
      ? "warn"
      : "info";

  const report = ui.report("GitHub Actions Audit")
    .status(overallStatus, `Inspected ${result.repository}`)
    .kv("Repository", result.repository)
    .kv("Workspace", result.workspace || "unknown")
    .kv("Actor", result.actor || "unknown")
    .kv("Event", result.eventName || "unknown")
    .kv("Assessment", result.summary.status)
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none")
    .kv("Allowed actions", result.permissions.allowed_actions || "unknown")
    .kv(
      "Workflow permissions",
      result.workflowPermissions.default_workflow_permissions || "unknown"
    )
    .kv("Workflow count", String(result.workflowCount));

  if (result.selectedActionsStatus === "fetched") {
    report.note("selected-actions policy is active for this repository");
  } else if (result.selectedActionsReason) {
    report.status("skip", result.selectedActionsReason);
  }

  if (result.runnerOutput && !result.runnerOutput.written) {
    report.warn(`Runner output file not written: ${result.runnerOutput.error}`);
  }
  if (result.stepSummary && !result.stepSummary.written) {
    report.warn(`Step summary file not written: ${result.stepSummary.error}`);
  }

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No baseline repository-level policy findings detected.");
      return;
    }

    section.table({
      columns: ["Severity", "Rule", "Message"],
      rows: result.findings.map((finding) => [
        findings.severityLabel(finding.severity),
        finding.ruleId,
        finding.message
      ])
    });
  });

  report.section("Workflows", (section) => {
    if (!result.workflows.length) {
      section.note("No workflow definitions were returned by the GitHub API");
      return;
    }

    section.table({
      columns: ["Name", "Path"],
      rows: result.workflows.map((workflow) => [
        workflow.name || "(unnamed)",
        workflow.path || "(none)"
      ])
    });
  });

  report.section("Local workflow files", (section) => {
    if (!result.localWorkflowFiles.length) {
      section.note("No local workflow files found under .github/workflows");
      return;
    }
    section.list(result.localWorkflowFiles);
  });

  report.render();
}

module.exports = function () {
  const owner = core.getInput("owner") || github.context.repo.owner;
  const repo = core.getInput("repo") || github.context.repo.repo;

  if (!owner || !repo) {
    throw new Error("owner and repo must be available from inputs or github.context");
  }

  const octokit = github.getOctokit();
  const permissions = octokit.rest.actions.getGithubActionsPermissionsRepository({ owner, repo }).data;
  const selectedActions = resolveSelectedActions(octokit, owner, repo, permissions);
  const workflowPermissions = octokit.rest.actions.getWorkflowPermissionsRepository({ owner, repo }).data;
  const workflowsResponse = octokit.rest.actions.listRepoWorkflows({ owner, repo }).data;
  const workflows = workflowsResponse.workflows || [];

  const result = {
    scriptId: "permissions-audit",
    actor: github.context.actor || null,
    eventName: github.context.event_name || null,
    repository: `${owner}/${repo}`,
    workspace: workspaceLib.resolveWorkspace(),
    permissions,
    selectedActions: selectedActions.data,
    selectedActionsStatus: selectedActions.status,
    selectedActionsReason: selectedActions.reason,
    workflowPermissions,
    workflowCount: workflowsResponse.total_count || workflows.length,
    workflows: workflows.map((workflow) => ({
      id: workflow.id,
      name: workflow.name,
      path: workflow.path || null
    })),
    localWorkflowFiles: workspaceLib.tryReadWorkflowFiles(io)
  };
  result.findings = buildFindings(result);
  result.summary = findings.summarizeFindings(result.findings);

  result.runnerOutput = trySetAuditOutput(result);
  result.stepSummary = tryWriteAuditSummary(result);
  renderAuditReport(result);

  return result;
};
