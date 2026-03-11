const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function hasWorkflowRunTrigger(document) {
  return !!document.workflowRun || (document.triggerNames || []).includes("workflow_run");
}

function containsWorkflowRunHeadReference(value) {
  const normalized = String(value || "").toLowerCase();
  if (!normalized) {
    return false;
  }
  return normalized.includes("github.event.workflow_run.head_");
}

function isArtifactDownloadReference(reference) {
  const spec = String(reference.uses || "").toLowerCase();
  return spec.startsWith("actions/download-artifact@") || spec.startsWith("dawidd6/action-download-artifact@");
}

function runStepsForJob(document, jobId) {
  return (document.runSteps || []).filter((step) => step.jobId === jobId);
}

function formatWorkflowRunDetails(document) {
  if (!document.workflowRun) {
    return "unknown upstream workflow configuration";
  }
  const upstream = (document.workflowRun.workflows || []).join(", ") || "any";
  const types = (document.workflowRun.types || []).join(", ") || "default";
  return `${upstream} (${types})`;
}

function buildFindings(documents) {
  const collected = [];

  for (const document of documents) {
    if (!hasWorkflowRunTrigger(document)) {
      continue;
    }

    collected.push(findings.makeFinding({
      ruleId: "workflow-run-review",
      severity: "medium",
      scope: "workflow",
      message: `Workflow ${document.path} uses workflow_run and should be reviewed as a follow-up workflow boundary.`,
      whyItMatters: "workflow_run can turn outputs, artifacts, or upstream repository state from another workflow into inputs for a more privileged follow-up workflow.",
      evidence: {
        path: document.path,
        workflowRun: document.workflowRun || null,
        upstream: formatWorkflowRunDetails(document)
      },
      remediation: {
        summary: "Review whether the upstream workflow is trusted and whether the follow-up workflow writes to the repository, publishes artifacts, or uses privileged tokens."
      }
    }));

    for (const reference of document.uses || []) {
      if (!isArtifactDownloadReference(reference)) {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "workflow-run-artifact-bridge",
        severity: "high",
        scope: "workflow",
        message: `${document.path} downloads artifacts in a workflow_run follow-up job.`,
        whyItMatters: "Downloading artifacts in a workflow_run job can bridge data from a less-trusted upstream workflow into a more privileged follow-up execution path.",
        evidence: {
          path: document.path,
          line: reference.line,
          jobId: reference.jobId || null,
          uses: reference.uses
        },
        remediation: {
          summary: "Review whether the upstream workflow is fully trusted before consuming its artifacts in a follow-up workflow."
        }
      }));
    }

    for (const checkout of document.checkoutSteps || []) {
      const untrustedRef = containsWorkflowRunHeadReference(checkout.ref);
      const untrustedRepository = containsWorkflowRunHeadReference(checkout.repository);
      if (!untrustedRef && !untrustedRepository) {
        continue;
      }

      const runSteps = runStepsForJob(document, checkout.jobId);
      const severity = runSteps.length > 0 ? "critical" : "high";
      collected.push(findings.makeFinding({
        ruleId: "workflow-run-head-checkout",
        severity,
        scope: "workflow",
        message: `${document.path} checks out upstream workflow head content under workflow_run${runSteps.length > 0 ? " and then executes shell steps in the same job" : ""}.`,
        whyItMatters: "Checking out upstream head refs in a workflow_run job can import less-trusted repository state into a privileged follow-up workflow. Shell execution after that checkout raises the risk further.",
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
          summary: "Avoid checking out workflow_run head refs in privileged follow-up workflows unless the upstream workflow is trusted and the boundary is intentionally reviewed."
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Workflow Run Review")
    .description(
      "This audit reviews workflow_run follow-up workflows and highlights " +
      "patterns that can bridge artifacts or upstream head state into a " +
      "potentially more privileged execution path."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Reviewed workflows", String(result.reviewedWorkflowCount))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No workflows use workflow_run in a way that matched this review rule.");
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
  const reviewedWorkflows = documents.filter(hasWorkflowRunTrigger);

  const result = {
    scriptId: "workflow-run-review",
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
