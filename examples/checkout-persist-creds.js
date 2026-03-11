const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function collectFindings(documents) {
  const collected = [];

  for (const document of documents) {
    for (const step of document.checkoutSteps || []) {
      if (step.persistCredentials === "false") {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "checkout-persist-creds",
        severity: "high",
        scope: "workflow",
        message: `${step.uses} is used without persist-credentials: false`,
        whyItMatters: "Persisted checkout credentials can widen token exposure inside a workflow run and make credential exfiltration easier.",
        evidence: {
          path: document.path,
          line: step.line,
          uses: step.uses,
          jobId: step.jobId || null,
          stepName: step.stepName || null,
          persistCredentials: step.persistCredentials
        },
        remediation: {
          summary: "Add `persist-credentials: false` under the checkout step's `with:` block unless the workflow has a reviewed reason to keep credentials persisted."
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Checkout Persist Credentials")
    .description(
      "This audit checks that every actions/checkout step in your workflow files " +
      "explicitly sets persist-credentials: false. Without this, the GITHUB_TOKEN " +
      "remains on disk in the runner's git config, making credential exfiltration " +
      "easier if any subsequent step is compromised."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("All checkout steps explicitly disable persisted credentials.");
      return;
    }

    section.findings(result.findings, {
      locationFields: ["path", "line", "uses"]
    });
  });

  report.render();
}

module.exports = function () {
  const workspace = workspaceLib.resolveWorkspace();
  const workflowFiles = workflows.listFiles();
  const documents = workflows.parseAll();
  const result = {
    scriptId: "checkout-persist-creds",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
