module.exports = function () {
  return {
    cwd: process.cwd(),
    workspace: process.env.GITHUB_WORKSPACE || null,
    eventPath: process.env.GITHUB_EVENT_PATH || null,
    ok: true
  };
};
