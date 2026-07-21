const mock = require("./mock");

function request(path, options = {}) {
  const app = getApp();
  return new Promise((resolve) => {
    wx.request({
      url: `${app.globalData.apiBase}${path}`,
      method: options.method || "GET",
      data: options.data || {},
      header: {
        Authorization: app.globalData.token ? `Bearer ${app.globalData.token}` : ""
      },
      success: (res) => resolve(res.data),
      fail: () => {
        const data = fallback(path, options.data || {});
        if (data && typeof data.code === "number") {
          resolve(data);
          return;
        }
        resolve({ code: 0, data, message: "mock fallback" });
      }
    });
  });
}

function uploadImage(filePath) {
  const app = getApp();
  const base = app.globalData.apiBase.replace(/\/api\/v1\/?$/, "");
  return new Promise((resolve) => {
    wx.uploadFile({
      url: `${base}/api/v1/uploads/images`,
      filePath,
      name: "file",
      header: {
        Authorization: app.globalData.token ? `Bearer ${app.globalData.token}` : ""
      },
      success: (res) => {
        try {
          resolve(JSON.parse(res.data));
        } catch (err) {
          resolve({ code: 500, message: "图片上传失败" });
        }
      },
      fail: () => resolve({ code: 0, data: { url: filePath }, message: "mock upload fallback" })
    });
  });
}

function fallback(path, data = {}) {
  path = path.split("?")[0];
  if (path.includes("/pair/code")) return mock.generatePairCode(data.userId);
  if (path.includes("/pair/confirm")) return mock.confirmPair(data);
  if (path.includes("/couple/love-date")) return mock.updateLoveDate(data.loveDate);
  if (path.includes("/users/") && path.includes("/profile")) {
    const id = path.split("/")[2];
    return mock.updateUserProfile(id, data);
  }
  if (path.includes("/moments/") && path.includes("/liked")) return mock.updateMomentLiked(path.split("/")[2], data.liked);
  if (path.includes("/moments/")) return mock.deleteMoment(path.split("/")[2]);
  if (path === "/moments") return data.content ? mock.createMoment(data) : mock.moments;
  if (path.includes("/tasks/") && path.includes("/complete")) return mock.updateTaskStatus(path.split("/")[2], "review");
  if (path.includes("/tasks/") && path.includes("/approve")) return mock.updateTaskStatus(path.split("/")[2], "done");
  if (path.includes("/tasks/") && path.includes("/reject")) return mock.updateTaskStatus(path.split("/")[2], "todo");
  if (path.includes("/tasks/")) return mock.deleteTask(path.split("/")[2]);
  if (path === "/tasks") return data.title ? mock.createTask(data) : mock.todos;
  if (path.includes("/scheduled-tasks/") && path.includes("/confirm")) return mock.confirmSchedule(path.split("/")[2]);
  if (path.includes("/scheduled-tasks/")) return mock.deleteSchedule(path.split("/")[2]);
  if (path === "/scheduled-tasks") return data.title ? mock.createSchedule(data) : mock.schedules;
  if (path.includes("/dishes/") && path.includes("/enabled")) return mock.updateDishEnabled(path.split("/")[2], data.enabled);
  if (path.includes("/dishes/")) return mock.deleteDish(path.split("/")[2]);
  if (path === "/dishes") return data.name ? mock.createDish(data) : mock.dishes;
  if (path === "/orders") return data.dishes ? mock.createOrder(data) : mock.orderHistory;
  if (path.includes("/goals/") && path.includes("/status")) return mock.updateGoalStatus(path.split("/")[2], data.status);
  if (path.includes("/goals/") && path.includes("/value")) return mock.updateGoalValue(path.split("/")[2], data.currentValue);
  if (path.includes("/goals/")) return mock.deleteGoal(path.split("/")[2]);
  if (path === "/goals") return data.title ? mock.createGoal(data) : mock.goals;
  if (path.includes("dashboard")) return { users: mock.users, dashboard: mock.dashboard };
  if (path.includes("me")) return { users: mock.users, dashboard: mock.dashboard };
  return {};
}

module.exports = {
  request,
  uploadImage,
  generatePairCode: (userId) => request("/pair/code", { method: "POST", data: { userId } }),
  confirmPair: (data) => request("/pair/confirm", { method: "POST", data }),
  getDashboard: () => {
    const app = getApp();
    if (app.globalData.demoMode || wx.getStorageSync("demoMode")) {
      return Promise.resolve({ code: 0, data: { users: mock.users, dashboard: mock.dashboard } });
    }
    const userId = app.globalData.currentUserId;
    return request(userId ? `/dashboard?userId=${encodeURIComponent(userId)}` : "/dashboard");
  },
  updateLoveDate: (loveDate) => request("/couple/love-date", { method: "PATCH", data: { loveDate } }),
  updateUserProfile: (id, data) => request(`/users/${id}/profile`, { method: "PATCH", data }),
  getMoments: () => request("/moments"),
  createMoment: (data) => request("/moments", { method: "POST", data }),
  deleteMoment: (id) => request(`/moments/${id}`, { method: "DELETE" }),
  updateMomentLiked: (id, liked) => request(`/moments/${id}/liked`, { method: "PATCH", data: { liked } }),
  getTasks: () => request("/tasks"),
  createTask: (data) => request("/tasks", { method: "POST", data }),
  deleteTask: (id) => request(`/tasks/${id}`, { method: "DELETE" }),
  completeTask: (id) => request(`/tasks/${id}/complete`, { method: "POST" }),
  approveTask: (id) => request(`/tasks/${id}/approve`, { method: "POST" }),
  rejectTask: (id) => request(`/tasks/${id}/reject`, { method: "POST" }),
  getSchedules: () => request("/scheduled-tasks"),
  createSchedule: (data) => request("/scheduled-tasks", { method: "POST", data }),
  deleteSchedule: (id) => request(`/scheduled-tasks/${id}`, { method: "DELETE" }),
  confirmSchedule: (id) => request(`/scheduled-tasks/${id}/confirm`, { method: "POST" }),
  getDishes: () => request("/dishes"),
  createDish: (data) => request("/dishes", { method: "POST", data }),
  deleteDish: (id) => request(`/dishes/${id}`, { method: "DELETE" }),
  updateDishEnabled: (id, enabled) => request(`/dishes/${id}/enabled`, { method: "PATCH", data: { enabled } }),
  getOrders: () => request("/orders"),
  createOrder: (data) => request("/orders", { method: "POST", data }),
  getGoals: () => request("/goals"),
  createGoal: (data) => request("/goals", { method: "POST", data }),
  updateGoalValue: (id, currentValue) => request(`/goals/${id}/value`, { method: "PATCH", data: { currentValue } }),
  updateGoalStatus: (id, status) => request(`/goals/${id}/status`, { method: "PATCH", data: { status } }),
  deleteGoal: (id) => request(`/goals/${id}`, { method: "DELETE" }),
  mock
};
