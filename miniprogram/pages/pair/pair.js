const api = require("../../utils/api");
const session = require("../../utils/session");

Page({
  data: {
    mode: "input",
    code: "",
    generatedCode: "",
    shareTime: "",
    keys: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]
  },

  switchMode(e) {
    this.setData({ mode: e.currentTarget.dataset.mode });
  },

  tapKey(e) {
    if (this.data.code.length >= 6) return;
    this.setData({ code: `${this.data.code}${e.currentTarget.dataset.key}` });
  },

  deleteKey() {
    this.setData({ code: this.data.code.slice(0, -1) });
  },

  generateCode() {
    const app = getApp();
    api.generatePairCode(app.globalData.currentUserId).then((ret) => {
      if (!isOk(ret)) {
        wx.showToast({ title: ret.message || "生成分享码失败", icon: "none" });
        return;
      }
      const data = ret.data || ret;
      this.setData({
        generatedCode: data.pairCode || "",
        shareTime: expireText(data.codeExpireAt),
        mode: "generate"
      });
    });
  },

  confirmPair() {
    if (this.data.code.length !== 6) {
      wx.showToast({ title: "请输入6位分享码", icon: "none" });
      return;
    }
    const app = getApp();
    api.confirmPair({
      userId: app.globalData.currentUserId,
      code: this.data.code,
      loveDate: new Date().toISOString().slice(0, 10)
    }).then((ret) => {
      if (!isOk(ret)) {
        wx.showToast({ title: pairErrorText(ret), icon: "none" });
        return;
      }
      session.markPaired();
      wx.switchTab({ url: "/pages/home/home" });
    });
  },

  enterDemo() {
    session.enterDemo();
    wx.switchTab({ url: "/pages/home/home" });
  }
});

function isOk(ret) {
  return !ret || ret.code === undefined || ret.code === 0;
}

function pairErrorText(ret = {}) {
  return ret.message || "分享码输入错误";
}

function expireText(value) {
  const expire = value ? new Date(value).getTime() : Date.now() + 24 * 60 * 60 * 1000;
  const remain = Math.max(0, expire - Date.now());
  const hours = Math.floor(remain / 3600000);
  const minutes = Math.floor((remain % 3600000) / 60000);
  return `有效期 ${hours}小时${minutes}分钟`;
}
