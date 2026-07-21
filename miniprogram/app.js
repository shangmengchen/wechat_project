App({
  globalData: {
    apiBase: "http://127.0.0.1:8080/api/v1",
    token: "",
    currentUserId: "",
    demoMode: false,
    isPaired: false
  },

  onLaunch() {
    const token = wx.getStorageSync("token");
    if (token) {
      this.globalData.token = token;
    }
    let userId = wx.getStorageSync("currentUserId");
    if (!userId) {
      userId = `local-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
      wx.setStorageSync("currentUserId", userId);
    }
    this.globalData.currentUserId = userId;
    this.globalData.demoMode = !!wx.getStorageSync("demoMode");
    this.globalData.isPaired = !!wx.getStorageSync("isPaired");
    this.checkScheduleReminders();
  },

  onShow() {
    this.checkScheduleReminders();
  },

  checkScheduleReminders() {
    if (!this.globalData.demoMode && !this.globalData.isPaired) return;
    const api = require("./utils/api");
    api.getSchedules().then((res) => {
      const schedules = res.data || res || [];
      const due = dueSchedules(schedules);
      if (!due.length) return;
      const today = new Date().toISOString().slice(0, 10);
      const key = `scheduleReminder:${today}:${due.map((item) => item.id).join(",")}`;
      if (wx.getStorageSync(key)) return;
      wx.setStorageSync(key, true);
      wx.showModal({
        title: "定时提醒",
        content: `${due[0].title} 到提醒时间了`,
        showCancel: false
      });
    });
  }
});

function dueSchedules(schedules) {
  const now = new Date();
  const current = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  const day = ["每周日", "每周一", "每周二", "每周三", "每周四", "每周五", "每周六"][now.getDay()];
  return (schedules || []).filter((item) => {
    if (!item.pending || !item.time || item.time > current) return false;
    return item.cycle === "每天" || item.cycle === day;
  });
}
