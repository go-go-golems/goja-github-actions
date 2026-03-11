const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function collectFindings(documents) {
  const collected = [];

  for (const document of documents) {
    for (const permission of document.permissions || []) {
      if (permission.kind !== "scalar" || permission.value !== "write-all") {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "no-write-all",
        severity: "high",
        scope: permission.scope || "workflow",
        message: `${permission.scope === "job" ? `Job ${permission.jobId}` : "Workflow"} uses permissions: write-all`,
        whyItMatters: "Broad write-all permissions give every permission category write access, which increases the blast radius of compromised workflow execution.",
        evidence: {
          path: document.path,
          line: permission.line,
          scope: permission.scope,
          job: permission.scope === "job" ? permission.jobId : null,
          permissions: "write-all"
        },
        remediation: {
          summary: "Replace `write-all` with a minimal explicit permission map or with `read-all` when write access is not required."
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("No Write-All")
    .description(
      "This audit checks that no workflow or job in your repository uses the " +
      "overly broad permissions: write-all setting. Broad write-all permissions " +
      "give every permission category write access, increasing the blast radius " +
      "of compromised workflow execution."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No workflow or job uses permissions: write-all.");
      return;
    }

    section.findings(result.findings, {
      locationFields: ["path", "line", "scope"]
    });
  });

  report.render();
}

module.exports = function () {
  const workspace = workspaceLib.resolveWorkspace();
  const workflowFiles = workflows.listFiles();
  const documents = workflows.parseAll();
  const result = {
    scriptId: "no-write-all",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
