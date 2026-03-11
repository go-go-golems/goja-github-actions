const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function isLocalReference(spec) {
  return spec.startsWith("./");
}

function isDockerReference(spec) {
  return spec.startsWith("docker://");
}

function extractRef(spec) {
  const at = spec.lastIndexOf("@");
  if (at === -1) {
    return "";
  }
  return spec.slice(at + 1).trim();
}

function isFullSha(ref) {
  return /^[0-9a-f]{40}$/i.test(ref);
}

function collectFindings(documents) {
  const collected = [];

  for (const document of documents) {
    for (const reference of document.uses || []) {
      if (isLocalReference(reference.uses) || isDockerReference(reference.uses)) {
        continue;
      }

      const ref = extractRef(reference.uses);
      if (isFullSha(ref)) {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "pin-third-party-actions",
        severity: "high",
        scope: "workflow",
        message: `Action or reusable workflow is not pinned to a full commit SHA: ${reference.uses}`,
        whyItMatters: "Mutable refs such as tags and branches can change without notice, which increases supply-chain risk in GitHub Actions workflows.",
        evidence: {
          path: document.path,
          line: reference.line,
          uses: reference.uses,
          kind: reference.kind,
          jobId: reference.jobId || null,
          stepName: reference.stepName || null
        },
        remediation: {
          summary: "Pin external actions and reusable workflows to a full commit SHA.",
          example: `${reference.uses.split("@")[0]}@0123456789abcdef0123456789abcdef01234567`
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Pin Third-Party Actions")
    .description(
      "This audit checks that all third-party GitHub Actions and reusable " +
      "workflows are pinned to full commit SHAs rather than mutable tags or " +
      "branch names. Mutable refs can change without notice, which increases " +
      "supply-chain risk."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("All detected action and reusable workflow references are pinned to full commit SHAs.");
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
    scriptId: "pin-third-party-actions",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
