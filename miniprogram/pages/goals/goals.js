const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

const today = new Date().toISOString().slice(0, 10);

Page({
  data: {
    active: "active",
    goals: api.mock.goals,
    showGoalForm: false,
    goalForm: {
      title: "",
      period: "",
      targetValue: "",
      currentValue: "0",
      startDate: today,
      targetDate: today
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
    return api.getGoals().then((res) => {
      const goals = normalizeGoals(res.data || res || api.mock.goals);
      this.setData({ goals });
    });
  },

  switchTab(e) {
    this.setData({ active: e.currentTarget.dataset.key });
  },

  goBack() {
    wx.navigateBack({
      fail: () => wx.switchTab({ url: "/pages/home/home" })
    });
  },

  addGoal() {
    const end = new Date();
    end.setDate(end.getDate() + 30);
    this.setData({
      showGoalForm: true,
      goalForm: {
        title: "",
        period: "",
        targetValue: "",
        currentValue: "0",
        startDate: today,
        targetDate: end.toISOString().slice(0, 10)
      }
    });
  },

  closeGoalForm() {
    this.setData({ showGoalForm: false });
  },

  changeGoalTitle(e) {
    this.setData({ "goalForm.title": e.detail.value });
  },

  changeGoalPeriod(e) {
    this.setData({ "goalForm.period": e.detail.value });
  },

  changeGoalTarget(e) {
    this.setData({ "goalForm.targetValue": e.detail.value });
  },

  changeGoalCurrent(e) {
    this.setData({ "goalForm.currentValue": e.detail.value });
  },

  changeGoalStart(e) {
    this.setData({ "goalForm.startDate": e.detail.value });
  },

  changeGoalTargetDate(e) {
    this.setData({ "goalForm.targetDate": e.detail.value });
  },

  submitGoal() {
    const form = this.data.goalForm;
    const title = (form.title || "").trim();
    const targetValue = Number(form.targetValue);
    const currentValue = Number(form.currentValue || 0);
    if (!title) {
      wx.showToast({ title: "请输入目标内容", icon: "none" });
      return;
    }
    if (!targetValue || targetValue <= 0) {
      wx.showToast({ title: "请输入目标数值", icon: "none" });
      return;
    }
    const goal = calculateGoal({
      title,
      period: (form.period || "目标").trim(),
      targetValue,
      currentValue: Math.max(0, currentValue),
      startDate: form.startDate,
      targetDate: form.targetDate
    });
    api.createGoal(goal).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "创建失败", icon: "none" });
        return;
      }
      const data = calculateGoal(ret.data || { ...goal, id: `local-${Date.now()}`, status: "active" });
      this.setData({ goals: [data].concat(this.data.goals), showGoalForm: false });
    });
  },

  editGoalValue(e) {
    const id = e.currentTarget.dataset.id;
    const goal = this.data.goals.find((item) => item.id === id);
    if (!goal) return;
    wx.showModal({
      title: "更新当前数值",
      editable: true,
      placeholderText: `当前 ${goal.currentValue || 0}`,
      success: (res) => {
        if (!res.confirm) return;
        const currentValue = Number(res.content);
        if (Number.isNaN(currentValue) || currentValue < 0) {
          wx.showToast({ title: "请输入有效数值", icon: "none" });
          return;
        }
        api.updateGoalValue(id, currentValue).then((ret) => {
          if (ret && ret.code !== undefined && ret.code !== 0) {
            wx.showToast({ title: ret.message || "更新失败", icon: "none" });
            return;
          }
          const data = calculateGoal(ret.data || { ...goal, currentValue });
          this.setData({ goals: this.data.goals.map((item) => (item.id === id ? { ...item, ...data } : item)) });
        });
      }
    });
  },

  finishGoal(e) {
    const id = e.currentTarget.dataset.id;
    api.updateGoalStatus(id, "done").then((ret) => {
      const data = calculateGoal(ret.data || ret);
      this.setData({
        goals: this.data.goals.map((item) => (item.id === id ? { ...item, ...data, status: "done" } : item))
      });
    });
  },

  deleteGoal(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "删除目标",
      content: "确定删除这个目标吗？",
      success: (res) => {
        if (!res.confirm) return;
        api.deleteGoal(id).then(() => this.setData({ goals: this.data.goals.filter((item) => item.id !== id) }));
      }
    });
  }
});

function normalizeGoals(goals) {
  return (goals || []).map(calculateGoal);
}

function calculateGoal(goal) {
  const item = { ...goal };
  item.targetValue = Number(item.targetValue || 100);
  item.currentValue = Number(item.currentValue || 0);
  item.progress = item.status === "done" ? 100 : clamp(Math.round((item.currentValue / item.targetValue) * 100));
  item.timeProgress = timeProgress(item.startDate, item.targetDate);
  item.remainDays = remainDays(item.targetDate);
  if (item.progress >= 100) {
    item.progress = 100;
    item.status = "done";
    item.remainDays = 0;
  }
  item.status = item.status || "active";
  return item;
}

function timeProgress(startDate, targetDate) {
  const start = parseDate(startDate);
  const target = parseDate(targetDate);
  if (!start || !target) return 0;
  const total = target - start;
  if (total <= 0) return 100;
  return clamp(Math.round(((todayDate() - start) / total) * 100));
}

function remainDays(targetDate) {
  const target = parseDate(targetDate);
  if (!target) return 0;
  return Math.max(0, Math.ceil((target - todayDate()) / 86400000));
}

function parseDate(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value || "")) return null;
  const parts = value.split("-").map(Number);
  return new Date(parts[0], parts[1] - 1, parts[2]);
}

function todayDate() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function clamp(value) {
  if (value < 0) return 0;
  if (value > 100) return 100;
  return value;
}
