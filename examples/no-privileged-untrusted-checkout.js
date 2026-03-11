const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function triggerKind(document) {
  if ((document.triggerNames || []).includes("pull_request_target")) {
    return "pull_request_target";
  }
  if (document.workflowRun || (document.triggerNames || []).includes("workflow_run")) {
    return "workflow_run";
  }
  return "";
}

function containsUntrustedReference(trigger, value) {
  const normalized = String(value || "").toLowerCase();
  if (!normalized) {
    return false;
  }

  if (trigger === "pull_request_target") {
    return normalized.includes("github.event.pull_request.head.") || normalized.includes("github.head_ref");
  }
  if (trigger === "workflow_run") {
    return normalized.includes("github.event.workflow_run.head_");
  }
  return false;
}

function runStepsForJob(document, jobId) {
  return (document.runSteps || []).filter((step) => step.jobId === jobId);
}

function buildFindings(documents) {
  const collected = [];

  for (const document of documents) {
    const trigger = triggerKind(document);
    if (!trigger) {
      continue;
    }

    for (const checkout of document.checkoutSteps || []) {
      const untrustedRef = containsUntrustedReference(trigger, checkout.ref);
      const untrustedRepository = containsUntrustedReference(trigger, checkout.repository);
      if (!untrustedRef && !untrustedRepository) {
        continue;
      }

      const runSteps = runStepsForJob(document, checkout.jobId);
      const severity = runSteps.length > 0 ? "critical" : "high";

      collected.push(findings.makeFinding({
        ruleId: "no-privileged-untrusted-checkout",
        severity,
        scope: "workflow",
        message: `${document.path} checks out untrusted head content under privileged trigger ${trigger}${runSteps.length > 0 ? " and then executes shell steps in the same job" : ""}.`,
        whyItMatters: "Privileged follow-up workflows can expose write-capable tokens, secrets, or repository mutation authority to attacker-controlled code if they checkout untrusted head refs and then execute them.",
        evidence: {
          path: document.path,
          line: checkout.line,
          trigger,
          jobId: checkout.jobId || null,
          uses: checkout.uses,
          ref: checkout.ref || null,
          repository: checkout.repository || null,
          runStepCount: runSteps.length,
          runStepNames: runSteps.map((step) => step.stepName || `(line ${step.line})`)
        },
        remediation: {
          summary: "Avoid checking out untrusted head refs in privileged workflows. Use pull_request for untrusted code paths, or split the privileged action into a separate narrowly scoped workflow."
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("No Privileged Untrusted Checkout")
    .description(
      "This audit looks for privileged workflows that checkout untrusted head " +
      "content from pull_request_target or workflow_run contexts. It is a " +
      "consolidated high-signal rule for a common token and secret exposure pattern."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No privileged workflows matched the untrusted checkout pattern.");
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

  const result = {
    scriptId: "no-privileged-untrusted-checkout",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: buildFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
