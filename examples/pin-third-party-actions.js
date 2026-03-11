const io = require("@actions/io");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function parseUsesEntries(fileName, content) {
  const entries = [];
  const lines = content.split(/\r?\n/);

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    const match = line.match(/^\s*(?:-\s*)?uses:\s*['"]?([^'"]+)['"]?\s*$/);
    if (!match) {
      continue;
    }
    entries.push({
      fileName,
      line: i + 1,
      uses: match[1].trim()
    });
  }

  return entries;
}

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

function collectFindings(workflowFiles, workspace) {
  const collected = [];

  for (const fileName of workflowFiles) {
    const fullPath = `${workspace}/.github/workflows/${fileName}`;
    const content = io.readFile(fullPath);
    const entries = parseUsesEntries(fileName, content);

    for (const entry of entries) {
      if (isLocalReference(entry.uses) || isDockerReference(entry.uses)) {
        continue;
      }

      const ref = extractRef(entry.uses);
      if (isFullSha(ref)) {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "pin-third-party-actions",
        severity: "high",
        scope: "workflow",
        message: `Action or reusable workflow is not pinned to a full commit SHA: ${entry.uses}`,
        whyItMatters: "Mutable refs such as tags and branches can change without notice, which increases supply-chain risk in GitHub Actions workflows.",
        evidence: {
          path: `.github/workflows/${entry.fileName}`,
          line: entry.line,
          uses: entry.uses
        },
        remediation: {
          summary: "Pin external actions and reusable workflows to a full commit SHA.",
          example: `${entry.uses.split("@")[0]}@0123456789abcdef0123456789abcdef01234567`
        }
      }));
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Pin Third-Party Actions")
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

    section.table({
      columns: ["Severity", "Rule", "Path", "Line", "Uses"],
      rows: result.findings.map((finding) => [
        findings.severityLabel(finding.severity),
        finding.ruleId,
        finding.evidence.path || "",
        String(finding.evidence.line || ""),
        finding.evidence.uses || ""
      ])
    });
  });

  report.render();
}

module.exports = function () {
  const workspace = workspaceLib.resolveWorkspace();
  const workflowFiles = workspaceLib.tryReadWorkflowFiles(io);
  const result = {
    scriptId: "pin-third-party-actions",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(workflowFiles, workspace)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
