const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

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
    generatingCode: false,
    keys: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]
  },

  onShow() {
    if (session.isPaired()) {
      this.enterPairedHome(false);
      return;
    }
    pageSync.registerPageRefresh(this, "handleSyncRefresh");
    this.startShareTimer();
    this.refreshShareTime();
    this.syncPairStatus();
  },

  onHide() {
    pageSync.unregisterPageRefresh(this);
    this.stopShareTimer();
    this.stopPairStatusPolling();
  },

  onUnload() {
    pageSync.unregisterPageRefresh(this);
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
    if (this.data.generatingCode) return;
    if (hasActiveShareCode(this.data)) {
      this.setData({
        mode: "generate",
        ...buildShareState(this.data.generatedCode, this.data.shareExpireAt)
      });
      wx.showToast({ title: "当前分享码仍有效", icon: "none" });
      return;
    }

    const app = getApp();
    this.setData({ generatingCode: true });
    api.generatePairCode(app.globalData.currentUserId).then((ret) => {
      this.setData({ generatingCode: false });
      if (!isOk(ret)) {
        wx.showToast({ title: ret.message || "生成失败", icon: "none" });
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
      wx.showToast({ title: "请输入 6 位分享码", icon: "none" });
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
      this.enterPairedHome(false);
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
    if (session.isPaired()) {
      this.enterPairedHome(false);
      return;
    }
    if (this.pairStatusTimer || !hasActiveShareCode(this.data)) return;
    this.pairStatusTimer = setInterval(() => {
      this.syncPairStatus();
    }, 1000);
  },

  stopPairStatusPolling() {
    if (!this.pairStatusTimer) return;
    clearInterval(this.pairStatusTimer);
    this.pairStatusTimer = null;
  },

  syncPairStatus() {
    if (session.isPaired()) {
      this.enterPairedHome(false);
      return;
    }
    if (!hasActiveShareCode(this.data)) return;
    api.getSyncState().then((ret) => {
      const data = ret.data || ret || {};
      if (!data.paired) return;
      this.enterPairedHome(true);
    });
  },

  handleSyncRefresh() {
    const app = getApp();
    const state = (app.globalData && app.globalData.syncState) || {};
    if (state.paired || session.isPaired()) {
      this.enterPairedHome(true);
      return;
    }
    this.syncPairStatus();
  },

  enterPairedHome(showToast = true) {
    if (this.pairRedirecting) return;
    this.pairRedirecting = true;
    session.markPaired();
    this.stopShareTimer();
    this.stopPairStatusPolling();
    pageSync.unregisterPageRefresh(this);
    if (showToast) {
      wx.showToast({ title: "配对成功", icon: "success" });
    }
    setTimeout(() => {
      wx.redirectTo({
        url: "/pages/pair-success/pair-success",
        fail: () => wx.reLaunch({ url: "/pages/pair-success/pair-success" }),
        complete: () => {
          this.pairRedirecting = false;
        }
      });
    }, showToast ? 300 : 0);
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
      shareRuleText: "请重新生成分享码，并把最新的 6 位数字发给对方。",
      canRegenerate: true,
      regenerateButtonText: "生成新的分享码"
    };
  }

  return {
    shareTime: countdownText(remain),
    shareStatusText: "有效中",
    shareRuleText: "分享码 20 分钟内有效，请在有效期内把当前分享码发给对方。",
    canRegenerate: false,
    regenerateButtonText: "20 分钟内有效"
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
  return `剩余 ${pad2(minutes)}:${pad2(seconds)}`;
}

function pad2(value) {
  return String(value).padStart(2, "0");
}
