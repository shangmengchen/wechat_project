const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

const cycleOptions = ["每天", "周一", "周二", "周三", "周四", "周五", "周六", "周日"];

Page({
  data: {
    active: "pending",
    schedules: api.mock.schedules,
    pendingCount: 0,
    noticeText: "",
    showScheduleForm: false,
    cycleOptions,
    assigneeOptions: ["双方", api.mock.users.me.name, api.mock.users.partner.name, "轮流", "自定义"],
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
      this.setData({ assigneeOptions: ["双方", users.me.name, users.partner.name, "轮流", "自定义"] });
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
      wx.showToast({ title: "请输入任务内容", icon: "none" });
      return;
    }
    const selectedAssignee = this.data.assigneeOptions[this.data.assigneeIndex];
    const assignee = selectedAssignee === "自定义" ? (form.assigneeCustom || "").trim() : selectedAssignee;
    if (!assignee) {
      wx.showToast({ title: "请输入负责人", icon: "none" });
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
        wx.showToast({ title: ret.message || "创建失败", icon: "none" });
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
      title: "删除提醒",
      content: "确定删除这个定时任务吗？",
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
      title: "定时提醒",
      content: `${due[0].title} 到时间啦`,
      showCancel: false
    });
  },

  setSchedules(schedules) {
    const normalized = (schedules || []).map(normalizeSchedule);
    const pending = normalized.filter((item) => item.pending);
    this.setData({
      schedules: normalized,
      pendingCount: pending.length,
      noticeText: pending.length ? `${pending[0].next} · ${pending.length} 个提醒待完成` : "暂无待完成提醒"
    });
  }
});

function nextText(cycle, time) {
  const text = normalizeCycle(cycle || "每天");
  if (text === "每天") return `今天 ${time}`;
  return `${text} ${time}`;
}

function dueSchedules(schedules) {
  const now = new Date();
  const current = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  const weekdays = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
  const englishWeekdays = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
  const day = weekdays[now.getDay()];
  const englishDay = englishWeekdays[now.getDay()];
  return (schedules || []).filter((item) => {
    const cycle = String(item.cycle || "");
    if (!item.pending || !item.time || item.time > current) return false;
    return normalizeCycle(cycle) === "每天" || cycle === day || cycle === englishDay;
  });
}

function normalizeSchedule(schedule) {
  const item = { ...schedule };
  item.cycle = normalizeCycle(item.cycle);
  item.assignee = normalizeAssignee(item.assignee);
  item.next = item.next ? normalizeNext(item.next) : nextText(item.cycle, item.time);
  return item;
}

function normalizeCycle(cycle) {
  const value = String(cycle || "");
  const lower = value.toLowerCase();
  const map = {
    "every day": "每天",
    daily: "每天",
    monday: "周一",
    tuesday: "周二",
    wednesday: "周三",
    thursday: "周四",
    friday: "周五",
    saturday: "周六",
    sunday: "周日"
  };
  return map[lower] || value;
}

function normalizeAssignee(assignee) {
  const value = String(assignee || "");
  const lower = value.toLowerCase();
  const map = {
    both: "双方",
    rotate: "轮流",
    custom: "自定义"
  };
  return map[lower] || value;
}

function normalizeNext(next) {
  return String(next || "")
    .replace(/^today/i, "今天")
    .replace(/^mon(day)?/i, "周一")
    .replace(/^tue(sday)?/i, "周二")
    .replace(/^wed(nesday)?/i, "周三")
    .replace(/^thu(rsday)?/i, "周四")
    .replace(/^fri(day)?/i, "周五")
    .replace(/^sat(urday)?/i, "周六")
    .replace(/^sun(day)?/i, "周日");
}
