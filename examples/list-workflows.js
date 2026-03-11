const ghaExec = require("@actions/exec");
const github = require("@actions/github");
const io = require("@actions/io");

function tryReadWorkflowDir() {
  try {
    return io.readdir(".github/workflows");
  } catch (err) {
    return [];
  }
}

module.exports = async function () {
  const git = await ghaExec.exec("git", ["rev-parse", "--show-toplevel"], {
    silent: true
  });

  return {
    gitRoot: git.stdout.trim(),
    repository: github.context.repository || null,
    workflowFiles: tryReadWorkflowDir(),
    workspace: process.env.GITHUB_WORKSPACE || null
  };
};
