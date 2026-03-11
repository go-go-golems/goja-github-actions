const workflows = require("@goja-gha/workflows");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function currentRepository() {
  return String(process.env.GITHUB_REPOSITORY || "").trim();
}

function currentOwner() {
  const repository = currentRepository();
  if (!repository.includes("/")) {
    return "";
  }
  return repository.split("/")[0];
}

function isReusableWorkflowReference(spec) {
  const normalized = String(spec || "");
  return normalized.includes("/.github/workflows/");
}

function isLocalReusableWorkflow(spec) {
  return String(spec || "").startsWith("./");
}

function extractRef(spec) {
  const at = String(spec || "").lastIndexOf("@");
  if (at === -1) {
    return "";
  }
  return spec.slice(at + 1).trim();
}

function extractRepository(spec) {
  const withoutRef = String(spec || "").split("@")[0];
  const parts = withoutRef.split("/");
  if (parts.length < 2) {
    return "";
  }
  return `${parts[0]}/${parts[1]}`;
}

function isFullSha(ref) {
  return /^[0-9a-f]{40}$/i.test(ref);
}

function collectFindings(documents) {
  const collected = [];
  const owner = currentOwner();
  const repository = currentRepository();

  for (const document of documents) {
    for (const reference of document.uses || []) {
      if (reference.kind !== "job" || !isReusableWorkflowReference(reference.uses) || isLocalReusableWorkflow(reference.uses)) {
        continue;
      }

      const targetRepository = extractRepository(reference.uses);
      const targetOwner = targetRepository.split("/")[0] || "";
      const ref = extractRef(reference.uses);

      if (!isFullSha(ref)) {
        collected.push(findings.makeFinding({
          ruleId: "reusable-workflow-unpinned",
          severity: "high",
          scope: "workflow",
          message: `${document.path} uses a reusable workflow that is not pinned to a full commit SHA.`,
          whyItMatters: "Mutable refs on reusable workflows can change behavior without review and widen your supply-chain exposure.",
          evidence: {
            path: document.path,
            line: reference.line,
            jobId: reference.jobId || null,
            uses: reference.uses,
            targetRepository
          },
          remediation: {
            summary: "Pin reusable workflows to a full commit SHA instead of a branch or tag.",
            example: `${reference.uses.split("@")[0]}@0123456789abcdef0123456789abcdef01234567`
          }
        }));
      }

      if (targetOwner !== "" && owner !== "" && targetOwner !== owner) {
        collected.push(findings.makeFinding({
          ruleId: "reusable-workflow-external-owner",
          severity: "medium",
          scope: "workflow",
          message: `${document.path} uses a reusable workflow from a different owner: ${targetRepository}.`,
          whyItMatters: "Reusable workflows from outside the current owner boundary deserve explicit review because they inherit execution authority into your workflow run.",
          evidence: {
            path: document.path,
            line: reference.line,
            jobId: reference.jobId || null,
            uses: reference.uses,
            repository,
            targetRepository
          },
          remediation: {
            summary: "Prefer reusable workflows from the same trusted owner, or document why this external workflow is trusted and how it is pinned."
          }
        }));
      }
    }
  }

  return collected;
}

function renderReport(result) {
  const overallStatus = result.summary.status === "passed" ? "ok" : "warn";
  const report = ui.report("Reusable Workflow Trust")
    .description(
      "This audit reviews job-level reusable workflow references and highlights " +
      "two trust concerns: mutable refs and reusable workflows sourced from a " +
      "different owner boundary."
    )
    .status(overallStatus, `Inspected ${result.repository || result.workspace}`)
    .kv("Workspace", result.workspace)
    .kv("Workflow files", String(result.workflowFiles.length))
    .kv("Finding count", String(result.summary.findingCount))
    .kv("Highest severity", result.summary.highestSeverity || "none");

  report.section("Findings", (section) => {
    if (!result.findings.length) {
      section.success("No external or unpinned reusable workflow references matched this rule.");
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
    scriptId: "reusable-workflow-trust",
    repository: currentRepository() || null,
    workspace,
    workflowFiles,
    findings: collectFindings(documents)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
