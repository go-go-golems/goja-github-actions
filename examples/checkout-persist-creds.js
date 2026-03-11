const io = require("@actions/io");
const ui = require("@goja-gha/ui");
const findings = require("lib/findings.js");
const workspaceLib = require("lib/workspace.js");

function indentationWidth(line) {
  const match = line.match(/^(\s*)/);
  return match ? match[1].length : 0;
}

function parseCheckoutSteps(fileName, content) {
  const lines = content.split(/\r?\n/);
  const steps = [];

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    const stepStartMatch = line.match(/^(\s*)-\s*(.*)$/);
    if (!stepStartMatch) {
      continue;
    }

    const stepIndent = stepStartMatch[1].length;
    const stepLines = [line];
    let stepEnd = i + 1;

    for (; stepEnd < lines.length; stepEnd += 1) {
      const nextLine = lines[stepEnd];
      if (nextLine.trim() === "") {
        stepLines.push(nextLine);
        continue;
      }

      const nextIndent = indentationWidth(nextLine);
      if (/^\s*-\s*/.test(nextLine) && nextIndent === stepIndent) {
        break;
      }

      stepLines.push(nextLine);
    }

    let uses = null;
    let persistCredentialsValue = null;
    let stepName = null;

    for (const stepLine of stepLines) {
      const nameMatch = stepLine.match(/^\s*(?:-\s*)?name:\s*['"]?([^'"]+)['"]?\s*$/);
      if (nameMatch && !stepName) {
        stepName = nameMatch[1].trim();
      }

      const usesMatch = stepLine.match(/^\s*(?:-\s*)?uses:\s*['"]?(actions\/checkout@[^'"]+)['"]?\s*$/);
      if (usesMatch) {
        uses = usesMatch[1].trim();
      }

      const persistMatch = stepLine.match(/^\s*persist-credentials:\s*['"]?([^'"]+)['"]?\s*$/);
      if (persistMatch) {
        persistCredentialsValue = persistMatch[1].trim().toLowerCase();
      }
    }

    if (!uses) {
      i = stepEnd - 1;
      continue;
    }

    steps.push({
      fileName,
      line: i + 1,
      uses,
      stepName,
      persistCredentialsValue
    });

    i = stepEnd - 1;
  }

  return steps;
}

function collectFindings(workflowFiles, workspace) {
  const collected = [];

  for (const fileName of workflowFiles) {
    const fullPath = `${workspace}/.github/workflows/${fileName}`;
    const content = io.readFile(fullPath);
    const steps = parseCheckoutSteps(fileName, content);

    for (const step of steps) {
      if (step.persistCredentialsValue === "false") {
        continue;
      }

      collected.push(findings.makeFinding({
        ruleId: "checkout-persist-creds",
        severity: "high",
        scope: "workflow",
        message: `${step.uses} is used without persist-credentials: false`,
        whyItMatters: "Persisted checkout credentials can widen token exposure inside a workflow run and make credential exfiltration easier.",
        evidence: {
          path: `.github/workflows/${step.fileName}`,
          line: step.line,
          uses: step.uses,
          stepName: step.stepName,
          persistCredentials: step.persistCredentialsValue
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
  const workflowFiles = workspaceLib.tryReadWorkflowFiles(io);
  const result = {
    scriptId: "checkout-persist-creds",
    repository: process.env.GITHUB_REPOSITORY || null,
    workspace,
    workflowFiles,
    findings: collectFindings(workflowFiles, workspace)
  };

  result.summary = findings.summarizeFindings(result.findings);
  renderReport(result);
  return result;
};
