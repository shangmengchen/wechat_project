const api = require("../../utils/api");
const session = require("../../utils/session");

const today = new Date().toISOString().slice(0, 10);

Page({
  data: {
    active: "todo",
    tabs: [
      { key: "todo", text: "待完成", count: 3 },
      { key: "review", text: "待审核", count: 1 },
      { key: "done", text: "已完成", count: 1 }
    ],
    tasks: api.mock.todos,
    showTaskForm: false,
    typeOptions: ["一次性", "每月", "每年"],
    ownerOptions: ["双方", api.mock.users.me.name, api.mock.users.partner.name, "自定义"],
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
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
  },

  load() {
    api.getDashboard().then((res) => {
      const data = res.data || res;
      const users = data.users || api.mock.users;
      this.setData({ ownerOptions: ["双方", users.me.name, users.partner.name, "自定义"] });
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
      wx.showToast({ title: "请输入任务内容", icon: "none" });
      return;
    }
    const ownerOption = this.data.ownerOptions[this.data.ownerIndex];
    const owner = ownerOption === "自定义" ? (form.ownerCustom || "").trim() : ownerOption;
    if (!owner) {
      wx.showToast({ title: "请输入负责人", icon: "none" });
      return;
    }
    const task = {
      title,
      tag: (form.tag || "生活").trim(),
      owner,
      type: this.data.typeOptions[this.data.typeIndex],
      due: form.due,
      reward: ""
    };
    api.createTask(task).then((ret) => {
      if (ret && ret.code !== undefined && ret.code !== 0) {
        wx.showToast({ title: ret.message || "创建失败", icon: "none" });
        return;
      }
      const data = ret.data || { ...task, id: `local-${Date.now()}`, status: "todo" };
      this.setData({ tasks: [data].concat(this.data.tasks), showTaskForm: false }, this.refreshCounts);
    });
  },

  deleteTask(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "删除任务",
      content: "确定删除这条任务吗？",
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
