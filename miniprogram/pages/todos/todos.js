const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

const today = new Date().toISOString().slice(0, 10);

Page({
  data: {
    active: "todo",
    tabs: [
      { key: "todo", text: "Todo", count: 0 },
      { key: "review", text: "Review", count: 0 },
      { key: "done", text: "Done", count: 0 }
    ],
    users: normalizeUsers(api.mock.users),
    tasks: api.mock.todos,
    showTaskForm: false,
    typeOptions: ["one-time", "monthly", "yearly"],
    ownerOptions: ["both", api.mock.users.me.name, api.mock.users.partner.name, "custom"],
    typeIndex: 0,
    ownerIndex: 0,
    taskForm: {
      title: "",
      tag: "",
      due: today,
      ownerCustom: ""
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
    api.getDashboard().then((res) => {
      const data = res.data || res;
      const users = normalizeUsers(data.users || api.mock.users);
      this.setData({
        users,
        ownerOptions: ["both", users.me.name, users.partner.name, "custom"]
      });
    });
    return api.getTasks().then((res) => this.setData({ tasks: res.data || res || api.mock.todos }, this.refreshCounts));
  },

  switchTab(e) {
    this.setData({ active: e.currentTarget.dataset.key });
  },

  complete(e) {
    const id = e.currentTarget.dataset.id;
    api.completeTask(id).then((ret) => this.updateTask(id, ret.data || ret || { status: "review" }));
  },

  approve(e) {
    const id = e.currentTarget.dataset.id;
    api.approveTask(id).then((ret) => this.updateTask(id, ret.data || ret || { status: "done" }));
  },

  reject(e) {
    const id = e.currentTarget.dataset.id;
    api.rejectTask(id).then((ret) => this.updateTask(id, ret.data || ret || { status: "todo" }));
  },

  addTask() {
    this.setData({
      showTaskForm: true,
      typeIndex: 0,
      ownerIndex: 0,
      taskForm: { title: "", tag: "", due: today, ownerCustom: "" }
    });
  },

  closeTaskForm() {
    this.setData({ showTaskForm: false });
  },

  changeTaskTitle(e) {
    this.setData({ "taskForm.title": e.detail.value });
  },

  changeTaskTag(e) {
    this.setData({ "taskForm.tag": e.detail.value });
  },

  changeTaskDate(e) {
    this.setData({ "taskForm.due": e.detail.value });
  },

  changeTaskType(e) {
    this.setData({ typeIndex: Number(e.detail.value) });
  },

  changeTaskOwner(e) {
    this.setData({ ownerIndex: Number(e.detail.value) });
  },

  changeOwnerCustom(e) {
    this.setData({ "taskForm.ownerCustom": e.detail.value });
  },

  submitTask() {
    const form = this.data.taskForm;
    const title = (form.title || "").trim();
    if (!title) {
      wx.showToast({ title: "Task title required", icon: "none" });
      return;
    }
    const ownerOption = this.data.ownerOptions[this.data.ownerIndex];
    const owner = ownerOption === "custom" ? (form.ownerCustom || "").trim() : ownerOption;
    if (!owner) {
      wx.showToast({ title: "Owner required", icon: "none" });
      return;
    }
    const task = {
      title,
      tag: (form.tag || "life").trim(),
      owner,
      type: this.data.typeOptions[this.data.typeIndex],
      due: form.due,
      reward: ""
    };
    api.createTask(task).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "Create failed", icon: "none" });
        return;
      }
      const data = ret.data || { ...task, id: `local-${Date.now()}`, status: "todo" };
      this.setData({ tasks: [data].concat(this.data.tasks), showTaskForm: false }, this.refreshCounts);
    });
  },

  deleteTask(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "Delete task",
      content: "Delete this task?",
      success: (res) => {
        if (!res.confirm) return;
        api.deleteTask(id).then(() => {
          this.setData({ tasks: this.data.tasks.filter((item) => item.id !== id) }, this.refreshCounts);
        });
      }
    });
  },

  updateTask(id, data) {
    this.setData({
      tasks: this.data.tasks.map((item) => (item.id === id ? { ...item, ...data } : item))
    }, this.refreshCounts);
  },

  refreshCounts() {
    const counts = this.data.tasks.reduce((acc, item) => {
      acc[item.status] = (acc[item.status] || 0) + 1;
      return acc;
    }, {});
    this.setData({
      tabs: this.data.tabs.map((tab) => ({ ...tab, count: counts[tab.key] || 0 }))
    });
  }
});

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
