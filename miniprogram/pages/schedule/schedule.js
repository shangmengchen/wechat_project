const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

const cycleOptions = ["Every day", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

Page({
  data: {
    active: "pending",
    schedules: api.mock.schedules,
    pendingCount: 0,
    noticeText: "",
    showScheduleForm: false,
    cycleOptions,
    assigneeOptions: ["Both", api.mock.users.me.name, api.mock.users.partner.name, "Rotate", "Custom"],
    cycleIndex: 0,
    assigneeIndex: 0,
    scheduleForm: {
      title: "",
      time: "20:00",
      assigneeCustom: ""
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
    this.checkDueReminders();
  },

  onUnload() {
    pageSync.unregisterPageRefresh(this);
  },

  goBack() {
    wx.navigateBack({
      fail: () => wx.switchTab({ url: "/pages/home/home" })
    });
  },

  load() {
    api.getDashboard().then((res) => {
      const data = res.data || res;
      const users = data.users || api.mock.users;
      this.setData({ assigneeOptions: ["Both", users.me.name, users.partner.name, "Rotate", "Custom"] });
    });
    return api.getSchedules().then((res) => this.setSchedules(res.data || res || api.mock.schedules));
  },

  switchTab(e) {
    this.setData({ active: e.currentTarget.dataset.key });
  },

  confirm(e) {
    const id = e.currentTarget.dataset.id;
    api.confirmSchedule(id).then((ret) => {
      const data = ret.data || ret || { pending: false };
      this.setSchedules(this.data.schedules.map((item) => (item.id === id ? { ...item, ...data } : item)));
    });
  },

  addSchedule() {
    this.setData({
      showScheduleForm: true,
      cycleIndex: 0,
      assigneeIndex: 0,
      scheduleForm: { title: "", time: "20:00", assigneeCustom: "" }
    });
  },

  closeScheduleForm() {
    this.setData({ showScheduleForm: false });
  },

  changeScheduleTitle(e) {
    this.setData({ "scheduleForm.title": e.detail.value });
  },

  changeScheduleTime(e) {
    this.setData({ "scheduleForm.time": e.detail.value });
  },

  changeCycle(e) {
    this.setData({ cycleIndex: Number(e.detail.value) });
  },

  changeAssignee(e) {
    this.setData({ assigneeIndex: Number(e.detail.value) });
  },

  changeAssigneeCustom(e) {
    this.setData({ "scheduleForm.assigneeCustom": e.detail.value });
  },

  submitSchedule() {
    const form = this.data.scheduleForm;
    const title = (form.title || "").trim();
    if (!title) {
      wx.showToast({ title: "Task title required", icon: "none" });
      return;
    }
    const selectedAssignee = this.data.assigneeOptions[this.data.assigneeIndex];
    const assignee = selectedAssignee === "Custom" ? (form.assigneeCustom || "").trim() : selectedAssignee;
    if (!assignee) {
      wx.showToast({ title: "Assignee required", icon: "none" });
      return;
    }
    const task = {
      title,
      cycle: this.data.cycleOptions[this.data.cycleIndex],
      assignee,
      time: form.time,
      next: nextText(this.data.cycleOptions[this.data.cycleIndex], form.time)
    };
    api.createSchedule(task).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "Create failed", icon: "none" });
        return;
      }
      const data = ret.data || { ...task, id: `local-${Date.now()}`, pending: true };
      this.setSchedules([data].concat(this.data.schedules));
      this.setData({ showScheduleForm: false });
    });
  },

  deleteSchedule(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "Delete reminder",
      content: "Delete this scheduled task?",
      success: (res) => {
        if (!res.confirm) return;
        api.deleteSchedule(id).then(() => {
          this.setSchedules(this.data.schedules.filter((item) => item.id !== id));
        });
      }
    });
  },

  checkDueReminders() {
    const due = dueSchedules(this.data.schedules);
    if (!due.length) return;
    const today = new Date().toISOString().slice(0, 10);
    const key = `scheduleReminder:${today}:${due.map((item) => item.id).join(",")}`;
    if (wx.getStorageSync(key)) return;
    wx.setStorageSync(key, true);
    wx.showModal({
      title: "Reminder",
      content: `${due[0].title} is due now`,
      showCancel: false
    });
  },

  setSchedules(schedules) {
    const normalized = (schedules || []).map((item) => ({ ...item, next: item.next || nextText(item.cycle, item.time) }));
    const pending = normalized.filter((item) => item.pending);
    this.setData({
      schedules: normalized,
      pendingCount: pending.length,
      noticeText: pending.length ? `${pending[0].next} · ${pending.length} task(s) pending` : "No pending reminders"
    });
  }
});

function nextText(cycle, time) {
  if ((cycle || "").toLowerCase().includes("every day")) return `Today ${time}`;
  return `${cycle || "Every day"} ${time}`;
}

function dueSchedules(schedules) {
  const now = new Date();
  const current = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  const weekdays = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
  const day = weekdays[now.getDay()];
  return (schedules || []).filter((item) => {
    const cycle = String(item.cycle || "");
    if (!item.pending || !item.time || item.time > current) return false;
    return cycle === "Every day" || cycle === day;
  });
}
