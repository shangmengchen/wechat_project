const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");
const subscribe = require("../../utils/subscribe");

Page({
  data: {
    active: "feed",
    tabs: [
      { key: "feed", text: "动态" },
      { key: "timeline", text: "时间线" },
      { key: "dates", text: "纪念日" }
    ],
    users: normalizeUsers(api.mock.users),
    moments: api.mock.moments,
    anniversaries: api.mock.dashboard.anniversaries,
    monthTitle: "",
    showComposer: false,
    momentDraft: {
      content: "",
      image: ""
    }
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    pageSync.registerPageRefresh(this);
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    const app = getApp();
    if (app && typeof app.clearUnreadCategory === "function") {
      app.clearUnreadCategory("memory");
    }
    this.load();
  },

  onUnload() {
    pageSync.unregisterPageRefresh(this);
  },

  load() {
    api.getMoments().then((res) => {
      this.setData({
        moments: res.data || res || api.mock.moments,
        monthTitle: currentMonthTitle()
      });
    });
    api.getDashboard().then((res) => {
      const data = res.data || res;
      const dashboard = data.dashboard || api.mock.dashboard;
      this.setData({
        users: normalizeUsers(data.users || api.mock.users),
        anniversaries: dashboard.anniversaries || api.mock.dashboard.anniversaries
      });
    });
  },

  switchTab(e) {
    this.setData({ active: e.currentTarget.dataset.key });
  },

  toggleComposer() {
    this.setData({ showComposer: !this.data.showComposer });
  },

  changeMomentContent(e) {
    this.setData({ "momentDraft.content": e.detail.value });
  },

  chooseDraftImage() {
    chooseImage().then((filePath) => {
      if (!filePath) return "";
      return api.uploadImage(filePath).then((ret) => {
        if (!ret || ret.code === 0) {
          const data = ret.data || ret;
          this.setData({ "momentDraft.image": data.url || filePath });
          return;
        }
        wx.showToast({ title: ret.message || "上传失败", icon: "none" });
      });
    });
  },

  removeDraftImage() {
    this.setData({ "momentDraft.image": "" });
  },

  submitMoment() {
    subscribe.request("notice");
    const content = (this.data.momentDraft.content || "").trim();
    if (!content) {
      wx.showToast({ title: "请输入内容", icon: "none" });
      return;
    }
    const moment = {
      author: this.data.users.me.name,
      avatar: this.data.users.me.avatarText,
      tag: "日常记录",
      content,
      image: this.data.momentDraft.image
    };
    api.createMoment(moment).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "发布失败", icon: "none" });
        return;
      }
      const data = ret.data || ret || { ...moment, id: `local-${Date.now()}`, time: "刚刚", liked: false };
      this.setData({
        moments: [data].concat(this.data.moments),
        showComposer: false,
        momentDraft: { content: "", image: "" }
      });
    });
  },

  deleteMoment(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "删除动态",
      content: "确定删除这条纪念动态吗？",
      success: (res) => {
        if (!res.confirm) return;
        api.deleteMoment(id).then(() => {
          this.setData({ moments: this.data.moments.filter((item) => item.id !== id) });
        });
      }
    });
  },

  toggleLike(e) {
    const id = e.currentTarget.dataset.id;
    const target = this.data.moments.find((item) => item.id === id);
    if (!target) return;
    const liked = !target.liked;
    api.updateMomentLiked(id, liked).then((ret) => {
      const data = ret.data || ret;
      this.setData({
        moments: this.data.moments.map((item) => (item.id === id ? { ...item, liked: data.liked !== undefined ? data.liked : liked } : item))
      });
    });
  }
});

function chooseImage() {
  return new Promise((resolve) => {
    if (wx.chooseMedia) {
      wx.chooseMedia({
        count: 1,
        mediaType: ["image"],
        sourceType: ["album", "camera"],
        success: (res) => resolve(res.tempFiles && res.tempFiles[0] ? res.tempFiles[0].tempFilePath : ""),
        fail: () => resolve("")
      });
      return;
    }
    wx.chooseImage({
      count: 1,
      sourceType: ["album", "camera"],
      success: (res) => resolve(res.tempFilePaths && res.tempFilePaths[0] ? res.tempFilePaths[0] : ""),
      fail: () => resolve("")
    });
  });
}

function currentMonthTitle() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
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
  return next;
}
