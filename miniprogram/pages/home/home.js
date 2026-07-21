const api = require("../../utils/api");
const session = require("../../utils/session");

Page({
  data: {
    users: api.mock.users,
    dashboard: api.mock.dashboard
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  onPullDownRefresh() {
    this.load().finally(() => wx.stopPullDownRefresh());
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
    const tabPages = ["/pages/home/home", "/pages/memory/memory", "/pages/todos/todos", "/pages/mine/mine"];
    if (tabPages.includes(url)) {
      wx.switchTab({ url });
      return;
    }
    wx.navigateTo({ url });
  }
});
