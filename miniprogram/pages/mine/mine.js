const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

Page({
  data: {
    users: normalizeUsers(api.mock.users),
    dashboard: api.mock.dashboard,
    expanded: false,
    profileForm: {
      nickname: "",
      wxid: ""
    }
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    pageSync.registerPageRefresh(this);
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  onUnload() {
    pageSync.unregisterPageRefresh(this);
  },

  load() {
    return api.getDashboard().then((res) => {
      const data = res.data || res;
      const users = normalizeUsers(data.users || api.mock.users);
      const me = users.me;
      this.setData({
        users,
        dashboard: data.dashboard || api.mock.dashboard,
        birthdayValue: toDateValue(me.birthday),
        profileForm: {
          nickname: me.name || "",
          wxid: me.wxid || ""
        }
      });
    });
  },

  changeLoveDate(e) {
    api.updateLoveDate(e.detail.value).then(() => {
      wx.showToast({ title: "已更新", icon: "success" });
      this.load();
    });
  },

  changeBirthday(e) {
    this.updateProfile({ birthday: e.detail.value });
  },

  changeNickname(e) {
    this.setData({ "profileForm.nickname": e.detail.value });
  },

  changeWxid(e) {
    this.setData({ "profileForm.wxid": e.detail.value });
  },

  saveProfile() {
    this.updateProfile({
      nickname: (this.data.profileForm.nickname || "").trim(),
      wxid: (this.data.profileForm.wxid || "").trim()
    });
  },

  syncWechatProfile() {
    const app = getApp();
    const applyProfile = (profile) => {
      const nickname = (profile.nickName || "").trim();
      const avatar = profile.avatarUrl || "";
      if (!nickname && !avatar) {
        wx.showToast({ title: "未获取到微信资料", icon: "none" });
        return;
      }
      Promise.resolve(app.syncUserProfile({ nickname, avatar }))
        .then(() => {
          const me = this.data.users.me;
          return api.updateUserProfile(me.id, {
            nickname: nickname || me.name,
            avatar: avatar || me.avatar,
            birthday: this.data.birthdayValue || "",
            wxid: (this.data.profileForm.wxid || me.wxid || "").trim()
          });
        })
        .then(() => {
          wx.showToast({ title: "已同步", icon: "success" });
          this.load();
        });
    };

    if (wx.getUserProfile) {
      wx.getUserProfile({
        desc: "同步昵称和头像",
        success: (res) => applyProfile(res.userInfo || {}),
        fail: () => wx.showToast({ title: "未授权", icon: "none" })
      });
      return;
    }

    wx.showToast({ title: "当前微信版本不支持", icon: "none" });
  },

  unbindPair() {
    wx.showModal({
      title: "解除绑定",
      content: "解除后双方都会回到分享码匹配界面，历史记录会保留但暂时不可见。确认解除吗？",
      confirmText: "确认解除",
      confirmColor: "#ff3867",
      success: (res) => {
        if (!res.confirm) return;
        api.unbindPair().then((ret) => {
          if (ret && ret.code !== undefined && ret.code !== 0) {
            wx.showToast({ title: ret.message || "解除失败", icon: "none" });
            return;
          }
          wx.showToast({ title: "已解除绑定", icon: "success" });
          const app = getApp();
          if (app && typeof app.handlePairUnbound === "function") {
            app.handlePairUnbound({
              initiatorId: app.globalData.currentUserId,
              silent: true,
              wasPaired: true
            });
          } else {
            session.clearPairing();
            wx.reLaunch({ url: "/pages/pair/pair" });
          }
        });
      }
    });
  },

  updateProfile(patch) {
    const me = this.data.users.me;
    const next = {
      nickname: normalizeText(patch.nickname, me.name),
      avatar: normalizeText(patch.avatar, me.avatar),
      birthday: normalizeText(patch.birthday, this.data.birthdayValue || toDateValue(me.birthday)),
      wxid: normalizeText(patch.wxid, this.data.profileForm.wxid || me.wxid)
    };

    api.updateUserProfile(me.id, next).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "更新失败", icon: "none" });
        return;
      }
      wx.showToast({ title: "已更新", icon: "success" });
      this.load();
    });
  },

  toggleExpanded() {
    this.setData({ expanded: !this.data.expanded });
  }
});

function normalizeText(value, fallback) {
  if (value === undefined || value === null || value === "") {
    return fallback || "";
  }
  return value;
}

function normalizeUsers(users) {
  const next = users || api.mock.users;
  return {
    me: normalizeUser(next.me || api.mock.users.me),
    partner: normalizeUser(next.partner || api.mock.users.partner)
  };
}

function normalizeUser(user) {
  const next = { ...user };
  next.avatarText = String(next.name || "?").slice(0, 1);
  if (!next.avatar) next.avatar = "";
  return next;
}

function toDateValue(value) {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value || "")) return value;
  const match = String(value || "").match(/^(\d{1,2})\D+(\d{1,2})/);
  if (!match) return "2000-01-01";
  const month = match[1].padStart(2, "0");
  const day = match[2].padStart(2, "0");
  return `2000-${month}-${day}`;
}
