function resolveWorkspace() {
  return process.env.GITHUB_WORKSPACE || process.cwd() || ".";
}

module.exports = {
  resolveWorkspace
};
