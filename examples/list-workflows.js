const ghaExec = require("@actions/exec");
const github = require("@actions/github");
const workflows = require("@goja-gha/workflows");

module.exports = async function () {
  const git = await ghaExec.exec("git", ["rev-parse", "--show-toplevel"], {
    silent: true
  });

  return {
    gitRoot: git.stdout.trim(),
    repository: github.context.repository || null,
    workflowFiles: workflows.listFiles(),
    workspace: process.env.GITHUB_WORKSPACE || null
  };
};
