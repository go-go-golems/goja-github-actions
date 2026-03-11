function resolveWorkspace() {
  return process.env.GITHUB_WORKSPACE || process.cwd() || ".";
}

function tryReadWorkflowFiles(io) {
  try {
    return io.readdir(".github/workflows");
  } catch (err) {
    return [];
  }
}

module.exports = {
  resolveWorkspace,
  tryReadWorkflowFiles
};
