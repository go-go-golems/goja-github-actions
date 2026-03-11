const core = require("@actions/core");

module.exports = function () {
  const name = core.getInput("name", { required: true });
  const flag = core.getBooleanInput("flag");
  const items = core.getMultilineInput("items");

  core.setOutput("result", `${name}:${items.length}`);
  core.exportVariable("GOJA_GHA_NAME", name);
  core.addPath("/tmp/goja-gha-bin");
  core.summary.addHeading("Core Example").addRaw(`flag=${flag}\n`).write();

  return {
    flag,
    items,
    name,
    path: process.env.PATH,
    workspace: process.env.GITHUB_WORKSPACE || null
  };
};
