const api = require("../../utils/api");
const session = require("../../utils/session");

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
    regenerateButtonText: "Regenerate code",
    keys: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]
  },

  onShow() {
    if (session.isPaired()) {
      wx.switchTab({ url: "/pages/home/home" });
      return;
    }
    this.startShareTimer();
    this.refreshShareTime();
    this.syncPairStatus();
  },

  onHide() {
    this.stopShareTimer();
    this.stopPairStatusPolling();
  },

  onUnload() {
    this.stopShareTimer();
    this.stopPairStatusPolling();
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
      wx.showToast({ title: "Current code is still valid", icon: "none" });
      return;
    }

    const app = getApp();
    api.generatePairCode(app.globalData.currentUserId).then((ret) => {
      if (!isOk(ret)) {
        wx.showToast({ title: ret.message || "Generate failed", icon: "none" });
        return;
      }
      const data = ret.data || ret;
      this.setData({
        generatedCode: data.pairCode || "",
        shareExpireAt: data.codeExpireAt || "",
        mode: "generate",
        ...buildShareState(data.pairCode || "", data.codeExpireAt || "")
      });
      this.startPairStatusPolling();
    });
  },

  confirmPair() {
    if (this.data.code.length !== 6) {
      wx.showToast({ title: "Enter the 6-digit code", icon: "none" });
      return;
    }

    api.confirmPair({
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
    if (hasActiveShareCode(this.data)) {
      this.startPairStatusPolling();
      return;
    }
    this.stopPairStatusPolling();
  },

  startPairStatusPolling() {
    if (this.pairStatusTimer || !hasActiveShareCode(this.data) || session.isPaired()) return;
    this.pairStatusTimer = setInterval(() => {
      this.syncPairStatus();
    }, 3000);
  },

  stopPairStatusPolling() {
    if (!this.pairStatusTimer) return;
    clearInterval(this.pairStatusTimer);
    this.pairStatusTimer = null;
  },

  syncPairStatus() {
    if (!hasActiveShareCode(this.data) || session.isPaired()) return;
    api.getSyncState().then((ret) => {
      const data = ret.data || ret || {};
      if (!data.paired) return;
      session.markPaired();
      this.stopPairStatusPolling();
      wx.showToast({ title: "Paired", icon: "success" });
      setTimeout(() => {
        wx.switchTab({ url: "/pages/home/home" });
      }, 300);
    });
  }
});

function isOk(ret) {
  return !ret || ret.code === undefined || ret.code === 0;
}

function pairErrorText(ret = {}) {
  return ret.message || "Invalid pair code";
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
      regenerateButtonText: "Regenerate code"
    };
  }

  const remain = remainingMs(shareExpireAt);
  if (remain <= 0) {
    return {
      shareTime: "Code expired",
      shareStatusText: "Expired",
      shareRuleText: "Generate a new code and send the latest 6-digit code to your partner.",
      canRegenerate: true,
      regenerateButtonText: "Generate new code"
    };
  }

  return {
    shareTime: countdownText(remain),
    shareStatusText: "Active",
    shareRuleText: "The code stays valid for 20 minutes. During that time, send the current code directly to your partner.",
    canRegenerate: false,
    regenerateButtonText: "Valid for 20 minutes"
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
  return `Expires in ${pad2(minutes)}:${pad2(seconds)}`;
}

function pad2(value) {
  return String(value).padStart(2, "0");
}
