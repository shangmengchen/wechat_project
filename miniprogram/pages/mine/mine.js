const api = require("../../utils/api");
const session = require("../../utils/session");

Page({
  data: {
    users: api.mock.users,
    dashboard: api.mock.dashboard,
    expanded: false
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  load() {
    return api.getDashboard().then((res) => {
      const data = res.data || res;
      const users = data.users || api.mock.users;
      this.setData({
        users,
        dashboard: data.dashboard || api.mock.dashboard,
        birthdayValue: toDateValue(users.me.birthday)
      });
    });
  },

  changeLoveDate(e) {
    const loveDate = e.detail.value;
    api.updateLoveDate(loveDate).then(() => {
      wx.showToast({ title: "已更新", icon: "success" });
      this.load();
    });
  },

  changeBirthday(e) {
    const birthday = e.detail.value;
    const me = this.data.users.me;
    api.updateUserProfile(me.id, {
      nickname: me.name,
      birthday,
      wxid: me.wxid
    }).then(() => {
      wx.showToast({ title: "已更新", icon: "success" });
      this.load();
    });
  },

  toggleExpanded() {
    this.setData({ expanded: !this.data.expanded });
  }
});

function toDateValue(value) {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value || "")) return value;
  const match = String(value || "").match(/^(\d{1,2})月(\d{1,2})日$/);
  if (!match) return "2000-01-01";
  const month = match[1].padStart(2, "0");
  const day = match[2].padStart(2, "0");
  return `2000-${month}-${day}`;
}
