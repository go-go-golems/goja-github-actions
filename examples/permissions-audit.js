const core = require("@actions/core");
const github = require("@actions/github");
const io = require("@actions/io");
const ui = require("@goja-gha/ui");

function tryReadWorkflowDir() {
  const workspace = process.env.GITHUB_WORKSPACE || process.cwd();
  const workflowDir = `${workspace}/.github/workflows`;
  try {
    return io.readdir(workflowDir);
  } catch (err) {
    return [];
  }
}

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

function renderAuditReport(result) {
  const report = ui.report("GitHub Actions Audit")
    .status("ok", `Inspected ${result.repository}`)
    .kv("Repository", result.repository)
    .kv("Actor", result.actor || "unknown")
    .kv("Event", result.eventName || "unknown")
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
    actor: github.context.actor || null,
    eventName: github.context.event_name || null,
    repository: `${owner}/${repo}`,
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
    localWorkflowFiles: tryReadWorkflowDir()
  };

  result.runnerOutput = trySetAuditOutput(result);
  result.stepSummary = tryWriteAuditSummary(result);
  renderAuditReport(result);

  return result;
};
