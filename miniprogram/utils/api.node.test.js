const assert = require("assert");
const path = require("path");

const apiModulePath = path.resolve(__dirname, "api.js");

function loadAPI({ requestImpl, storage = {}, globalData = {} }) {
  delete require.cache[apiModulePath];

  global.getApp = () => ({
    globalData: {
      apiBase: "http://127.0.0.1:8080/api/v1",
      token: "",
      currentUserId: "u-test",
      demoMode: false,
      ...globalData
    }
  });

  global.wx = {
    request: requestImpl,
    uploadFile: () => {
      throw new Error("uploadFile should not be called in these tests");
    },
    getStorageSync(key) {
      return storage[key];
    }
  };

  return require(apiModulePath);
}

async function testRequestFailureDoesNotFallbackToMock() {
  const api = loadAPI({
    requestImpl(options) {
      options.fail({ errMsg: "request:fail connect ECONNREFUSED" });
    }
  });

  const result = await api.getTasks();
  assert.strictEqual(result.code, 500, "failed requests should surface backend/network errors");
  assert.ok(
    /ECONNREFUSED|network|请求失败/i.test(result.message || ""),
    `unexpected failure message: ${result.message}`
  );
}

async function testDashboardAlwaysCallsBackendEvenInDemoMode() {
  let calledURL = "";
  const api = loadAPI({
    storage: { demoMode: true },
    globalData: { demoMode: true, currentUserId: "debug-user" },
    requestImpl(options) {
      calledURL = options.url;
      options.success({ data: { code: 0, data: { ok: true } } });
    }
  });

  const result = await api.getDashboard();
  assert.strictEqual(result.code, 0, "dashboard request should still resolve backend payload");
  assert.strictEqual(
    calledURL,
    "http://127.0.0.1:8080/api/v1/dashboard?userId=debug-user",
    "dashboard should call the backend endpoint directly in debug mode"
  );
}

async function main() {
  await testRequestFailureDoesNotFallbackToMock();
  await testDashboardAlwaysCallsBackendEvenInDemoMode();
  console.log("api.node.test.js passed");
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
