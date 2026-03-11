const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function isPullRequestTarget(document) {
  return (document.triggerNames || []).includes("pull_request_target");
}

function containsUntrustedPullRequestReference(value) {
  const normalized = String(value || "").toLowerCase();
  if (!normalized) {
    return false;
  }
  return normalized.includes("github.event.pull_request.head.") || normalized.includes("github.head_ref");
}

function runStepsForJob(document, jobId) {
  return (document.runSteps || []).filter((step) => step.jobId === jobId);
}

function buildFindings(documents) {
  const collected = [];

  for (const document of documents) {
    if (!isPullRequestTarget(document)) {
      continue;
    }

    collected.push(findings.makeFinding({
      ruleId: "pull-request-target-review",
      severity: "medium",
      scope: "workflow",
      message: `Workflow ${document.path} uses the pull_request_target trigger and should be reviewed carefully.`,
      whyItMatters: "pull_request_target runs with the base repository context and can expose secrets or write-capable tokens if the workflow later executes untrusted pull request content.",
      evidence: {
        path: document.path,
        trigger: "pull_request_target"
      },
      remediation: {
        summary: "Use pull_request for untrusted code paths when possible, or keep pull_request_target workflows narrowly scoped to metadata-only operations."
      }
    }));

    for (const checkout of document.checkoutSteps || []) {
      const untrustedRef = containsUntrustedPullRequestReference(checkout.ref);
      const untrustedRepository = containsUntrustedPullRequestReference(checkout.repository);
      if (!untrustedRef && !untrustedRepository) {
        continue;
      }

      const runSteps = runStepsForJob(document, checkout.jobId);
      const hasShellExecution = runSteps.length > 0;
      const severity = hasShellExecution ? "critical" : "high";

      collected.push(findings.makeFinding({
        ruleId: "pull-request-target-untrusted-checkout",
        severity,
        scope: "workflow",
        message: `${document.path} checks out pull request head content under pull_request_target${hasShellExecution ? " and then runs shell steps in the same job" : ""}.`,
        whyItMatters: "Checking out pull request head code under pull_request_target can turn a privileged workflow into an untrusted code execution path. The risk is highest when shell steps run after that checkout.",
        evidence: {
          path: document.path,
          line: checkout.line,
          jobId: checkout.jobId || null,
          uses: checkout.uses,
          ref: checkout.ref || null,
          repository: checkout.repository || null,
          runStepCount: runSteps.length,
          runStepNames: runSteps.map((step) => step.stepName || `(line ${step.line})`)
        },
        remediation: {
          summary: "Avoid checking out pull request head refs in pull_request_target workflows. If you must inspect PR content, use pull_request or move the privileged operation into a separate, tightly controlled workflow.",
          example: "Do not set checkout `with.ref` to `${{ github.event.pull_request.head.sha }}` inside a pull_request_target workflow."
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Pull Request Target Review")
    .description(
      "This audit looks for workflows that use pull_request_target and highlights " +
      "the especially risky pattern where the workflow checks out pull request head " +
      "content and then executes shell commands. That combination can turn a " +
      "privileged workflow into an untrusted code execution path."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Reviewed workflows", String(result.reviewedWorkflowCount))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No workflows use pull_request_target in a way that matched this review rule.");
      return;
    }

    section.findings(result.findings, {
      locationFields: ["path", "line", "jobId"]
    });
  });

  report.render();
}

module.exports = function () {
  const workspace = workspaceLib.resolveWorkspace();
  const documents = workflows.parseAll();
  const workflowFiles = workflows.listFiles();
  const reviewedWorkflows = documents.filter(isPullRequestTarget);

  const result = {
    scriptId: "pull-request-target-review",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    reviewedWorkflowCount: reviewedWorkflows.length,
    findings: buildFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
