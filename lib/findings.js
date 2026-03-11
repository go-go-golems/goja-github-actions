const severityOrder = {
  critical: 4,
  high: 3,
  medium: 2,
  low: 1,
  info: 0
};

function normalizeSeverity(severity) {
  const normalized = String(severity || "info").trim().toLowerCase();
  if (Object.prototype.hasOwnProperty.call(severityOrder, normalized)) {
    return normalized;
  }
  return "info";
}

function makeFinding(finding) {
  return {
    ruleId: String(finding.ruleId || "unknown").trim(),
    severity: normalizeSeverity(finding.severity),
    scope: String(finding.scope || "repository").trim(),
    message: String(finding.message || "").trim(),
    whyItMatters: finding.whyItMatters || null,
    evidence: finding.evidence || {},
    remediation: finding.remediation || null
  };
}

function severityLabel(severity) {
  return normalizeSeverity(severity).toUpperCase();
}

function compareSeverity(left, right) {
  return severityOrder[normalizeSeverity(left)] - severityOrder[normalizeSeverity(right)];
}

function summarizeFindings(findings) {
  const counts = {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    info: 0
  };

  let highestSeverity = "info";
  for (const finding of findings) {
    const severity = normalizeSeverity(finding.severity);
    counts[severity] += 1;
    if (compareSeverity(severity, highestSeverity) > 0) {
      highestSeverity = severity;
    }
  }

  if (!findings.length) {
    return {
      status: "passed",
      findingCount: 0,
      highestSeverity: null,
      counts
    };
  }

  return {
    status: compareSeverity(highestSeverity, "medium") >= 0 ? "findings" : "review",
    findingCount: findings.length,
    highestSeverity,
    counts
  };
}

module.exports = {
  makeFinding,
  severityLabel,
  summarizeFindings
};
