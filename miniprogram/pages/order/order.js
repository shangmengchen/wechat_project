const api = require("../../utils/api");
const session = require("../../utils/session");

Page({
  data: {
    active: "order",
    meal: "午餐",
    selected: [],
    dishes: api.mock.dishes,
    history: api.mock.orderHistory
  },

  onLoad() {
    if (!session.guardCouplePage()) return;
    this.load();
  },

  onShow() {
    if (!session.guardCouplePage()) return;
  },

  load() {
    this.loadDishes();
    return this.loadOrders();
  },

  loadOrders() {
    return api.getOrders().then((res) => this.setData({ history: res.data || res || api.mock.orderHistory }));
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
    const available = this.data.dishes.filter((item) => item.enabled && (item.meal === this.data.meal || item.meal === "通用"));
    if (!available.length) return;
    const pick = available[Math.floor(Math.random() * available.length)];
    this.setData({ selected: [pick.id] }, () => this.setDishes(this.data.dishes));
  },

  createOrder() {
    const selectedDishes = this.data.dishes.filter((item) => this.data.selected.includes(item.id));
    if (!selectedDishes.length) {
      wx.showToast({ title: "先选一道菜", icon: "none" });
      return;
    }
    api.createOrder({
      meal: this.data.meal,
      picker: `${api.mock.users.me.name}选的`,
      dishes: selectedDishes.map((item) => item.name)
    }).then((ret) => {
      const data = ret.data || ret;
      this.setData({
        selected: [],
        history: [data].concat(this.data.history),
        active: "history"
      }, () => this.setDishes(this.data.dishes));
      wx.showToast({ title: "已记录", icon: "success" });
    });
  },

  addDish() {
    wx.showModal({
      title: "添加菜品",
      editable: true,
      placeholderText: "输入菜品名称",
      success: (res) => {
        const name = (res.content || "").trim();
        if (!res.confirm || !name) return;
        const dish = { name, icon: "🍽️", meal: this.data.meal, enabled: true };
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
      dishes: dishes.map((dish) => ({
        ...dish,
        selected: this.data.selected.includes(dish.id)
      }))
    });
  }
});
