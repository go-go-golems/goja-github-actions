module.exports = async function () {
  console.log("console-log: script starting");
  console.warn("console-warn: warning path");
  console.error("console-error: error path");

  return {
    ok: true,
  };
};
