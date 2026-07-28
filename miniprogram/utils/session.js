function getAppData() {
  const app = getApp();
  return app.globalData || {};
}

function isDemo() {
  return !!wx.getStorageSync("demoMode") || !!getAppData().demoMode;
}

function isPaired() {
  return !!wx.getStorageSync("isPaired") || !!getAppData().isPaired;
}

function canUseCoupleFeatures() {
  return isDemo() || isPaired();
}

function enterDemo() {
  const app = getApp();
  wx.setStorageSync("demoMode", true);
  wx.setStorageSync("isPaired", false);
  app.globalData.demoMode = true;
  app.globalData.isPaired = false;
}

function markPaired() {
  const app = getApp();
  wx.setStorageSync("isPaired", true);
  wx.setStorageSync("demoMode", false);
  app.globalData.isPaired = true;
  app.globalData.demoMode = false;
}

function clearPairing() {
  const app = getApp();
  const userId = (app && app.globalData && app.globalData.currentUserId) || wx.getStorageSync("currentUserId") || "";
  wx.setStorageSync("isPaired", false);
  wx.setStorageSync("demoMode", false);
  if (userId) {
    wx.removeStorageSync(`wechatProfileSynced:${userId}`);
  }
  if (app && app.globalData) {
    app.globalData.isPaired = false;
    app.globalData.demoMode = false;
    app.globalData.syncState = {
      paired: false,
      coupleId: "",
      version: Date.now(),
      updatedAt: new Date().toISOString()
    };
  }
}

function guardCouplePage() {
  if (canUseCoupleFeatures()) return true;
  wx.showToast({ title: "请先完成情侣配对", icon: "none" });
  wx.reLaunch({ url: "/pages/pair/pair" });
  return false;
}

module.exports = {
  canUseCoupleFeatures,
  clearPairing,
  enterDemo,
  guardCouplePage,
  isDemo,
  isPaired,
  markPaired
};
