const core = require("@actions/core");

module.exports = function () {
  const name = core.getInput("name") || "world";
  const message = `hello ${name}`;

  core.setOutput("message", message);
  core.summary.addHeading("Set Output Example").addRaw(`${message}\n`).write();

  return {
    message
  };
};
