const core = require("@actions/core");
const github = require("@actions/github");
const io = require("@actions/io");

function tryReadWorkflowDir() {
  try {
    return io.readdir(".github/workflows");
  } catch (err) {
    return [];
  }
}

module.exports = function () {
  const owner = core.getInput("owner") || github.context.repo.owner;
  const repo = core.getInput("repo") || github.context.repo.repo;

  if (!owner || !repo) {
    throw new Error("owner and repo must be available from inputs or github.context");
  }

  const octokit = github.getOctokit();
  const permissions = octokit.rest.actions.getGithubActionsPermissionsRepository({ owner, repo }).data;
  const selectedActions = octokit.rest.actions.getAllowedActionsRepository({ owner, repo }).data;
  const workflowPermissions = octokit.rest.actions.getWorkflowPermissionsRepository({ owner, repo }).data;
  const workflowsResponse = octokit.rest.actions.listRepoWorkflows({ owner, repo }).data;
  const workflows = workflowsResponse.workflows || [];

  const result = {
    actor: github.context.actor || null,
    eventName: github.context.event_name || null,
    repository: `${owner}/${repo}`,
    permissions,
    selectedActions,
    workflowPermissions,
    workflowCount: workflowsResponse.total_count || workflows.length,
    workflows: workflows.map((workflow) => ({
      id: workflow.id,
      name: workflow.name,
      path: workflow.path || null
    })),
    localWorkflowFiles: tryReadWorkflowDir()
  };

  core.setOutput("audit", JSON.stringify(result));
  core.summary.addHeading("GitHub Actions Audit").addRaw(`${JSON.stringify(result, null, 2)}\n`).write();

  return result;
};
