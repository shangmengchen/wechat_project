const session = require("../../utils/session");

Page({
  data: {
    canEnterHome: false,
    syncingProfile: false,
    syncText: "正在准备同步微信信息..."
  },

  onShow() {
    session.markPaired();
    this.requestWechatProfile();
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

  requestWechatProfile() {
    const app = getApp();
    const userId = (app.globalData && app.globalData.currentUserId) || wx.getStorageSync("currentUserId") || "default";
    const syncedKey = `wechatProfileSynced:${userId}`;

    if (wx.getStorageSync(syncedKey)) {
      this.setData({ canEnterHome: true, syncText: "微信信息已同步，正在进入首页..." });
      this.startHomeTimer();
      return;
    }

    wx.showModal({
      title: "同步微信信息",
      content: "匹配成功后需要读取你的微信头像和昵称，用来完善小程序里的基本资料。",
      confirmText: "同意同步",
      cancelText: "退出",
      success: (res) => {
        if (!res.confirm) {
          this.promptExit();
          return;
        }
        this.getWechatProfile(syncedKey);
      }
    });
  },

  getWechatProfile(syncedKey) {
    if (!wx.getUserProfile) {
      this.promptExit("当前微信版本不支持读取用户信息，请升级微信后再进入。");
      return;
    }

    this.setData({
      syncingProfile: true,
      syncText: "正在同步微信头像和昵称..."
    });

    wx.getUserProfile({
      desc: "用于同步小程序基本资料",
      success: (res) => {
        const profile = res.userInfo || {};
        const nickname = (profile.nickName || "").trim();
        const avatar = profile.avatarUrl || "";
        if (!nickname && !avatar) {
          this.setData({ syncingProfile: false });
          this.promptExit("没有读取到微信信息，暂时无法继续。");
          return;
        }

        Promise.resolve(getApp().syncUserProfile({ nickname, avatar }))
          .then(() => {
            wx.setStorageSync(syncedKey, true);
            this.setData({
              canEnterHome: true,
              syncingProfile: false,
              syncText: "微信信息已同步，正在进入首页..."
            });
            wx.showToast({ title: "已同步微信信息", icon: "success" });
            this.startHomeTimer();
          })
          .catch(() => {
            this.setData({ syncingProfile: false });
            this.promptExit("微信信息同步失败，请稍后重新进入小程序。");
          });
      },
      fail: () => {
        this.setData({ syncingProfile: false });
        this.promptExit();
      }
    });
  },

  promptExit(content = "需要同步微信信息后才能继续使用，请退出后重新进入并授权。") {
    this.stopHomeTimer();
    wx.showModal({
      title: "需要授权",
      content,
      showCancel: false,
      confirmText: "退出小程序",
      success: () => {
        if (wx.exitMiniProgram) {
          wx.exitMiniProgram();
          return;
        }
        wx.reLaunch({ url: "/pages/pair/pair" });
      }
    });
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
