const session = require("../../utils/session");

Page({
  onShow() {
    session.markPaired();
    this.startHomeTimer();
  },

  onHide() {
    this.stopHomeTimer();
  },

  onUnload() {
    this.stopHomeTimer();
  },

  goHome() {
    this.stopHomeTimer();
    wx.switchTab({ url: "/pages/home/home" });
  },

  startHomeTimer() {
    this.stopHomeTimer();
    this.homeTimer = setTimeout(() => {
      this.goHome();
    }, 1800);
  },

  stopHomeTimer() {
    if (!this.homeTimer) return;
    clearTimeout(this.homeTimer);
    this.homeTimer = null;
  }
});
