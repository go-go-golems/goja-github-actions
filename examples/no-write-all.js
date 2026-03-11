const io = require("@actions/io");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function indentationWidth(line) {
  const match = line.match(/^(\s*)/);
  return match ? match[1].length : 0;
}

function collectFindings(workflowFiles, workspace) {
  const collected = [];

  for (const fileName of workflowFiles) {
    const fullPath = `${workspace}/.github/workflows/${fileName}`;
    const lines = io.readFile(fullPath).split(/\r?\n/);

    let jobsIndent = null;
    let currentJobId = null;
    let currentJobIndent = null;

    for (let i = 0; i < lines.length; i += 1) {
      const line = lines[i];
      if (line.trim() === "") {
        continue;
      }

      const indent = indentationWidth(line);

      if (jobsIndent !== null && indent <= jobsIndent && !/^\s*jobs:\s*$/.test(line)) {
        currentJobId = null;
        currentJobIndent = null;
      }

      const jobsMatch = line.match(/^(\s*)jobs:\s*$/);
      if (jobsMatch) {
        jobsIndent = jobsMatch[1].length;
        currentJobId = null;
        currentJobIndent = null;
        continue;
      }

      if (jobsIndent !== null && indent > jobsIndent) {
        const jobMatch = line.match(/^(\s*)([A-Za-z0-9_-]+):\s*$/);
        if (jobMatch && jobMatch[1].length === jobsIndent + 2) {
          currentJobId = jobMatch[2];
          currentJobIndent = jobMatch[1].length;
          continue;
        }
      }

      const permissionsMatch = line.match(/^\s*permissions:\s*['"]?(write-all)['"]?\s*$/);
      if (!permissionsMatch) {
        continue;
      }

      const scope = currentJobId && currentJobIndent !== null && indent > currentJobIndent
        ? "job"
        : "workflow";

      collected.push(findings.makeFinding({
        ruleId: "no-write-all",
        severity: "high",
        scope: "workflow",
        message: `${scope === "job" ? `Job ${currentJobId}` : "Workflow"} uses permissions: write-all`,
        whyItMatters: "Broad write-all permissions give every permission category write access, which increases the blast radius of compromised workflow execution.",
        evidence: {
          path: `.github/workflows/${fileName}`,
          line: i + 1,
          scope,
          job: scope === "job" ? currentJobId : null,
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
  const workflowFiles = workspaceLib.tryReadWorkflowFiles(io);
  const result = {
    scriptId: "no-write-all",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(workflowFiles, workspace)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
