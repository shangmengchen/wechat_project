const api = require("../../utils/api");
const session = require("../../utils/session");
const pageSync = require("../../utils/pageSync");

Page({
  data: {
    active: "order",
    meal: "lunch",
    selected: [],
    users: normalizeUsers(api.mock.users),
    dishes: normalizeDishes(api.mock.dishes),
    history: normalizeOrders(api.mock.orderHistory)
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    pageSync.registerPageRefresh(this);
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
    const app = getApp();
    if (app && typeof app.clearUnreadCategory === "function") {
      app.clearUnreadCategory("order");
    }
    this.load();
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
      this.setData({ users: normalizeUsers(data.users || api.mock.users) });
    });
    this.loadDishes();
    return this.loadOrders();
  },

  loadOrders() {
    return api.getOrders().then((res) => this.setData({ history: normalizeOrders(res.data || res || api.mock.orderHistory) }));
  },

  loadDishes() {
    return api.getDishes().then((res) => this.setDishes(res.data || res || api.mock.dishes));
  },

  switchTab(e) {
    this.setData({ active: e.currentTarget.dataset.key });
  },

  switchMeal(e) {
    this.setData({ meal: e.currentTarget.dataset.meal, selected: [] });
  },

  toggleDish(e) {
    const id = e.currentTarget.dataset.id;
    const selected = this.data.selected.includes(id)
      ? this.data.selected.filter((item) => item !== id)
      : this.data.selected.concat(id);
    this.setData({ selected }, () => this.setDishes(this.data.dishes));
  },

  randomPick() {
    const available = this.data.dishes.filter((item) => {
      const meal = normalizeMeal(item.meal);
      return item.enabled && (meal === this.data.meal || meal === "any");
    });
    if (!available.length) return;
    const pick = available[Math.floor(Math.random() * available.length)];
    this.setData({ selected: [pick.id] }, () => this.setDishes(this.data.dishes));
  },

  createOrder() {
    const selectedDishes = this.data.dishes.filter((item) => this.data.selected.includes(item.id));
    if (!selectedDishes.length) {
      wx.showToast({ title: "请至少选择一道菜", icon: "none" });
      return;
    }
    api.createOrder({
      meal: this.data.meal,
      picker: `${this.data.users.me.name} 选的`,
      dishes: selectedDishes.map((item) => item.name)
    }).then((ret) => {
      const data = normalizeOrder(ret.data || ret);
      this.setData({
        selected: [],
        history: [data].concat(this.data.history),
        active: "history"
      }, () => this.setDishes(this.data.dishes));
      wx.showToast({ title: "已保存", icon: "success" });
    });
  },

  addDish() {
    wx.showModal({
      title: "添加菜品",
      editable: true,
      placeholderText: "菜品名称",
      success: (res) => {
        const name = (res.content || "").trim();
        if (!res.confirm || !name) return;
        const dish = { name, icon: "🥗", meal: this.data.meal, enabled: true };
        api.createDish(dish).then((ret) => {
          const data = ret.data || { ...dish, id: `local-${Date.now()}` };
          this.setDishes([data].concat(this.data.dishes));
        });
      }
    });
  },

  deleteDish(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: "删除菜品",
      content: "确定删除这个菜品吗？",
      success: (res) => {
        if (!res.confirm) return;
        api.deleteDish(id).then(() => this.setDishes(this.data.dishes.filter((item) => item.id !== id)));
      }
    });
  },

  changeDishEnabled(e) {
    const id = e.currentTarget.dataset.id;
    const enabled = e.detail.value;
    api.updateDishEnabled(id, enabled).then((ret) => {
      const data = ret.data || ret || { enabled };
      this.setDishes(this.data.dishes.map((item) => (item.id === id ? { ...item, ...data } : item)));
    });
  },

  setDishes(dishes) {
    this.setData({
      dishes: normalizeDishes(dishes, this.data.selected)
    });
  }
});

function normalizeDishes(dishes, selected = []) {
  return (dishes || []).map((dish) => ({
    ...dish,
    meal: normalizeMeal(dish.meal),
    mealText: mealText(dish.meal),
    selected: selected.includes(dish.id)
  }));
}

function normalizeOrders(orders) {
  return (orders || []).map(normalizeOrder);
}

function normalizeOrder(order) {
  const item = { ...order };
  item.meal = normalizeMeal(item.meal);
  item.mealText = mealText(item.meal);
  if (item.picker) item.picker = String(item.picker).replace(/\s*picked$/i, " 选的");
  return item;
}

function normalizeMeal(meal) {
  const value = String(meal || "").toLowerCase();
  if (value.includes("breakfast")) return "breakfast";
  if (value.includes("lunch")) return "lunch";
  if (value.includes("dinner")) return "dinner";
  if (value.includes("any")) return "any";
  if (value === "早餐") return "breakfast";
  if (value === "午餐") return "lunch";
  if (value === "晚餐") return "dinner";
  if (value === "通用") return "any";
  return "any";
}

function mealText(meal) {
  const map = {
    breakfast: "早餐",
    lunch: "午餐",
    dinner: "晚餐",
    any: "通用"
  };
  return map[normalizeMeal(meal)] || "通用";
}

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
