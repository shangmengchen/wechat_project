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

function guardCouplePage() {
  if (canUseCoupleFeatures()) return true;
  wx.showToast({ title: "请先完成情侣配对", icon: "none" });
  wx.reLaunch({ url: "/pages/pair/pair" });
  return false;
}

module.exports = {
  canUseCoupleFeatures,
  enterDemo,
  guardCouplePage,
  isDemo,
  isPaired,
  markPaired
};
