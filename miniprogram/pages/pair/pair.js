const api = require("../../utils/api");
const session = require("../../utils/session");

const SHARE_TTL_MS = 20 * 60 * 1000;

Page({
  data: {
    mode: "input",
    code: "",
    generatedCode: "",
    shareTime: "",
    shareExpireAt: "",
    shareStatusText: "",
    shareRuleText: "",
    canRegenerate: true,
    regenerateButtonText: "重新生成分享码",
    keys: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]
  },

  onShow() {
    this.startShareTimer();
    this.refreshShareTime();
  },

  onHide() {
    this.stopShareTimer();
  },

  onUnload() {
    this.stopShareTimer();
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
    if (hasActiveShareCode(this.data)) {
      this.setData({
        mode: "generate",
        ...buildShareState(this.data.generatedCode, this.data.shareExpireAt)
      });
      wx.showToast({ title: "当前分享码仍在 20 分钟有效期内", icon: "none" });
      return;
    }

    const app = getApp();
    api.generatePairCode(app.globalData.currentUserId).then((ret) => {
      if (!isOk(ret)) {
        wx.showToast({ title: ret.message || "生成分享码失败", icon: "none" });
        return;
      }
      const data = ret.data || ret;
      this.setData({
        generatedCode: data.pairCode || "",
        shareExpireAt: data.codeExpireAt || "",
        mode: "generate",
        ...buildShareState(data.pairCode || "", data.codeExpireAt || "")
      });
    });
  },

  confirmPair() {
    if (this.data.code.length !== 6) {
      wx.showToast({ title: "请输入 6 位分享码", icon: "none" });
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
  },

  startShareTimer() {
    this.stopShareTimer();
    this.shareTimer = setInterval(() => {
      this.refreshShareTime();
    }, 1000);
  },

  stopShareTimer() {
    if (!this.shareTimer) return;
    clearInterval(this.shareTimer);
    this.shareTimer = null;
  },

  refreshShareTime() {
    const nextState = buildShareState(this.data.generatedCode, this.data.shareExpireAt);
    this.setData(nextState);
  }
});

function isOk(ret) {
  return !ret || ret.code === undefined || ret.code === 0;
}

function pairErrorText(ret = {}) {
  return ret.message || "分享码输入错误";
}

function hasActiveShareCode(data = {}) {
  if (!data.generatedCode || !data.shareExpireAt) return false;
  return remainingMs(data.shareExpireAt) > 0;
}

function buildShareState(generatedCode, shareExpireAt) {
  if (!generatedCode || !shareExpireAt) {
    return {
      shareTime: "",
      shareStatusText: "",
      shareRuleText: "",
      canRegenerate: true,
      regenerateButtonText: "重新生成分享码"
    };
  }

  const remain = remainingMs(shareExpireAt);
  if (remain <= 0) {
    return {
      shareTime: "分享码已过期",
      shareStatusText: "已过期",
      shareRuleText: "可以重新生成新的分享码，对方需要使用最新的 6 位分享码完成配对。",
      canRegenerate: true,
      regenerateButtonText: "重新生成分享码"
    };
  }

  return {
    shareTime: countdownText(remain),
    shareStatusText: "有效中",
    shareRuleText: "分享码在 20 分钟内有效，有效期内不会重复生成，请直接把当前这组分享码发给对方。",
    canRegenerate: false,
    regenerateButtonText: "20 分钟内不可重复生成"
  };
}

function remainingMs(expireAt) {
  const target = new Date(expireAt).getTime();
  if (!target) return 0;
  return Math.max(0, target - Date.now());
}

function countdownText(remain) {
  const totalSeconds = Math.max(0, Math.floor(remain / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `有效期剩余 ${pad2(minutes)}:${pad2(seconds)}`;
}

function pad2(value) {
  return String(value).padStart(2, "0");
}
