const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

Page({
  data: {
    users: api.mock.users,
    dashboard: api.mock.dashboard,
    unread: {}
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    pageSync.registerPageRefresh(this);
    this.refreshUnread();
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    this.refreshUnread();
    this.load();
  },

  onUnload() {
    pageSync.unregisterPageRefresh(this);
  },

  onPullDownRefresh() {
    this.load().finally(() => wx.stopPullDownRefresh());
  },

  refreshUnread() {
    const app = getApp();
    if (app && typeof app.getUnreadFlags === "function") {
      this.setData({ unread: app.getUnreadFlags() });
    }
  },

  load() {
    return api.getDashboard().then((res) => {
      const data = res.data || res;
      this.setData({
        users: data.users || api.mock.users,
        dashboard: data.dashboard || api.mock.dashboard
      });
    });
  },

  go(e) {
    const url = e.currentTarget.dataset.url;
    if (!url) return;
    this.clearUnreadByURL(url);
    const tabPages = ["/pages/home/home", "/pages/memory/memory", "/pages/todos/todos", "/pages/mine/mine"];
    if (tabPages.includes(url)) {
      wx.switchTab({ url });
      return;
    }
    wx.navigateTo({ url });
  },

  clearUnreadByURL(url) {
    const app = getApp();
    if (!app || typeof app.clearUnreadCategory !== "function") return;
    const map = {
      "/pages/memory/memory": "memory",
      "/pages/todos/todos": "todos",
      "/pages/schedule/schedule": "schedule",
      "/pages/order/order": "order",
      "/pages/goal/goals": "goals"
    };
    if (map[url]) {
      app.clearUnreadCategory(map[url]);
    }
  }
});
