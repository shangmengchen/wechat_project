const users = {
  me: {
    id: "u1",
    name: "小雨",
    avatarText: "小",
    birthday: "8月31日",
    wxid: "xiaoyu_2024"
  },
  partner: {
    id: "u2",
    name: "阿杰",
    avatarText: "杰",
    birthday: "9月18日",
    wxid: "ajie_2024"
  }
};

const dashboard = {
  loveDays: 0,
  since: "2023-07-28",
  anniversaries: [],
  stats: [],
  activeGoals: 0,
  goalProgress: 0,
  pendingTasks: 0
};

let pairTicket = null;

function updateDashboard() {
  const loveDate = parseDate(dashboard.since);
  const activeGoals = goals.filter((item) => item.status === "active");
  const pendingTasks = todos.filter((item) => item.status !== "done").length;
  dashboard.loveDays = loveDate ? daysSince(loveDate) : 0;
  dashboard.activeGoals = activeGoals.length;
  dashboard.pendingTasks = pendingTasks;
  dashboard.goalProgress = activeGoals.length
    ? Math.round(activeGoals.reduce((sum, item) => sum + Number(item.progress || 0), 0) / activeGoals.length)
    : 0;
  dashboard.stats = [
    { icon: "📷", value: moments.length, label: "纪念动态" },
    { icon: "📋", value: pendingTasks, label: "待办任务" },
    { icon: "🎯", value: activeGoals.length, label: "进行目标" }
  ];
  dashboard.anniversaries = [
    anniversary("💖", "恋爱纪念日", dashboard.since, "pink")
  ];
  const meBirthday = birthdayAnniversary(users.me, "purple");
  if (meBirthday) dashboard.anniversaries.push(meBirthday);
  const partnerBirthday = birthdayAnniversary(users.partner, "blue");
  if (partnerBirthday) dashboard.anniversaries.push(partnerBirthday);
  dashboard.anniversaries.push({
    icon: "🌹",
    title: "七夕节",
    days: daysUntil(8, 10),
    date: "8月10日",
    tone: "yellow"
  });
}

function updateLoveDate(loveDate) {
  dashboard.since = loveDate || dashboard.since;
  updateDashboard();
  return { id: "c1", loveDate: dashboard.since };
}

function updateUserProfile(id, data = {}) {
  const target = users.me.id === id ? users.me : users.partner.id === id ? users.partner : null;
  if (!target) return {};
  if (data.nickname || data.name) target.name = data.nickname || data.name;
  if (data.birthday) target.birthday = formatDateText(data.birthday);
  if (data.wxid) target.wxid = data.wxid;
  if (data.avatarText) target.avatarText = data.avatarText;
  updateDashboard();
  return {
    id,
    nickname: target.name,
    birthday: target.birthday,
    wxid: target.wxid
  };
}

function generatePairCode(userId) {
  const code = String(Math.floor(100000 + Math.random() * 900000));
  const codeExpireAt = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
  pairTicket = {
    code,
    codeExpireAt,
    userAId: userId || users.me.id
  };
  return {
    id: "c1",
    userAId: pairTicket.userAId,
    userBId: "",
    loveDate: dashboard.since,
    pairCode: code,
    codeExpireAt
  };
}

function confirmPair(data = {}) {
  const code = String(data.code || "");
  if (!pairTicket || pairTicket.code !== code || pairTicket.userAId === data.userId) {
    return { code: 400, message: "分享码输入错误" };
  }
  if (new Date(pairTicket.codeExpireAt).getTime() <= Date.now()) {
    return { code: 400, message: "分享码已过期，请让对方重新生成" };
  }
  if (data.loveDate) updateLoveDate(data.loveDate);
  const result = {
    id: "c1",
    userAId: pairTicket.userAId,
    userBId: data.userId || users.partner.id,
    loveDate: dashboard.since
  };
  pairTicket = null;
  return {
    code: 0,
    message: "ok",
    data: result
  };
}

function createMoment(data) {
  const moment = {
    id: `local-${Date.now()}`,
    author: data.author || users.me.name,
    avatar: data.avatar || users.me.avatarText,
    time: "刚刚",
    tag: data.tag || "日常记录",
    content: data.content,
    image: data.image || "",
    liked: false
  };
  moments.unshift(moment);
  updateDashboard();
  return moment;
}

function deleteMoment(id) {
  removeByID(moments, id);
  updateDashboard();
  return { deleted: true };
}

function updateMomentLiked(id, liked) {
  const item = moments.find((moment) => moment.id === id);
  if (item) item.liked = liked;
  return item || {};
}

function createTask(data) {
  const item = { tag: "生活", ...data, id: `local-${Date.now()}`, status: "todo" };
  todos.unshift(item);
  updateDashboard();
  return item;
}

function deleteTask(id) {
  removeByID(todos, id);
  updateDashboard();
  return { deleted: true };
}

function updateTaskStatus(id, status) {
  const item = todos.find((task) => task.id === id);
  if (item) item.status = status;
  updateDashboard();
  return item || {};
}

function createSchedule(data) {
  const item = {
    cycle: "每天",
    assignee: "双方",
    time: "20:00",
    next: `今天 ${data.time || "20:00"}`,
    ...data,
    id: `local-${Date.now()}`,
    pending: true
  };
  schedules.unshift(item);
  return item;
}

function deleteSchedule(id) {
  removeByID(schedules, id);
  return { deleted: true };
}

function confirmSchedule(id) {
  const item = schedules.find((task) => task.id === id);
  if (item) item.pending = false;
  return item || {};
}

function createDish(data) {
  const item = { ...data, id: `local-${Date.now()}`, enabled: data.enabled !== false };
  dishes.unshift(item);
  return item;
}

function deleteDish(id) {
  removeByID(dishes, id);
  return { deleted: true };
}

function updateDishEnabled(id, enabled) {
  const item = dishes.find((dish) => dish.id === id);
  if (item) item.enabled = enabled;
  return item || {};
}

function createOrder(data) {
  const item = {
    id: `local-${Date.now()}`,
    date: formatDateText(new Date()),
    meal: data.meal,
    picker: data.picker || `${users.me.name}选的`,
    dishes: data.dishes || []
  };
  orderHistory.unshift(item);
  return item;
}

function createGoal(data) {
  const item = calculateGoal({ ...data, id: `local-${Date.now()}`, status: "active" });
  goals.unshift(item);
  updateDashboard();
  return item;
}

function deleteGoal(id) {
  removeByID(goals, id);
  updateDashboard();
  return { deleted: true };
}

function updateGoalStatus(id, status) {
  const item = goals.find((goal) => goal.id === id);
  if (item) {
    item.status = status;
    if (status === "done") item.currentValue = item.targetValue;
    Object.assign(item, calculateGoal(item));
  }
  updateDashboard();
  return item || {};
}

function updateGoalValue(id, currentValue) {
  const item = goals.find((goal) => goal.id === id);
  if (item) {
    item.currentValue = Number(currentValue || 0);
    Object.assign(item, calculateGoal(item));
  }
  updateDashboard();
  return item || {};
}

function birthdayAnniversary(user, tone) {
  if (!user || !user.birthday) return null;
  return anniversary("🎂", `${user.name}生日`, user.birthday, tone);
}

function anniversary(icon, title, value, tone) {
  const date = parseDate(value);
  return {
    icon,
    title,
    days: date ? daysUntil(date.getMonth() + 1, date.getDate()) : 0,
    date: date ? formatDateText(value) : value,
    tone
  };
}

function calculateGoal(goal) {
  const item = { ...goal };
  item.targetValue = Number(item.targetValue || 100);
  item.currentValue = Number(item.currentValue || 0);
  item.startDate = item.startDate || new Date().toISOString().slice(0, 10);
  item.targetDate = item.targetDate || new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10);
  item.progress = item.status === "done" ? 100 : clampPercent(Math.round((item.currentValue / item.targetValue) * 100));
  item.timeProgress = goalTimeProgress(item.startDate, item.targetDate);
  item.remainDays = goalRemainDays(item.targetDate);
  if (item.progress >= 100) {
    item.progress = 100;
    item.status = "done";
    item.remainDays = 0;
  }
  item.status = item.status || "active";
  return item;
}

function goalTimeProgress(startDate, targetDate) {
  const start = parseDate(startDate);
  const target = parseDate(targetDate);
  if (!start || !target) return 0;
  const total = target - start;
  if (total <= 0) return 100;
  return clampPercent(Math.round(((todayDate() - start) / total) * 100));
}

function goalRemainDays(targetDate) {
  const target = parseDate(targetDate);
  if (!target) return 0;
  return Math.max(0, Math.ceil((target - todayDate()) / 86400000));
}

function clampPercent(value) {
  if (value < 0) return 0;
  if (value > 100) return 100;
  return value;
}

function parseDate(value) {
  if (!value) return null;
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const parts = value.split("-").map(Number);
    return new Date(parts[0], parts[1] - 1, parts[2]);
  }
  const match = value.match(/^(\d{1,2})月(\d{1,2})日$/);
  if (match) return new Date(new Date().getFullYear(), Number(match[1]) - 1, Number(match[2]));
  return null;
}

function daysSince(date) {
  const today = todayDate();
  const start = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  return Math.max(0, Math.floor((today - start) / 86400000) + 1);
}

function daysUntil(month, day) {
  const today = todayDate();
  let next = new Date(today.getFullYear(), month - 1, day);
  if (next < today) next = new Date(today.getFullYear() + 1, month - 1, day);
  return Math.floor((next - today) / 86400000);
}

function todayDate() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function formatDateText(value) {
  if (value instanceof Date) return `${value.getMonth() + 1}月${value.getDate()}日`;
  const date = parseDate(value);
  return date ? `${date.getMonth() + 1}月${date.getDate()}日` : value;
}

function removeByID(list, id) {
  const index = list.findIndex((item) => item.id === id);
  if (index >= 0) list.splice(index, 1);
}

const moments = [
  {
    id: "m1",
    author: "小雨",
    avatar: "小",
    time: "2小时前",
    tag: "约会日记",
    content: "今天去了北海公园划船，阳光超好，幸福感满满～",
    image: "https://images.unsplash.com/photo-1507525428034-b723cf961d3e?w=900&auto=format&fit=crop",
    liked: true
  },
  {
    id: "m2",
    author: "阿杰",
    avatar: "杰",
    time: "昨天 20:30",
    tag: "生活小事",
    content: "给小雨做了她最爱的番茄炒蛋，说我做得比她妈妈还好吃哈哈！",
    image: "",
    liked: false
  },
  {
    id: "m3",
    author: "小雨",
    avatar: "小",
    time: "3天前",
    tag: "约会日记",
    content: "一起看了新上映的电影，结局有点意外但很感动💕",
    image: "",
    liked: true
  }
];

const todos = [
  { id: "t1", title: "帮小雨买她想要的无线耳机", owner: "阿杰", type: "一次性", due: "7月25日", reward: "亲亲", status: "todo" },
  { id: "t2", title: "一起做一顿丰盛的晚餐", owner: "双方", type: "一次性", due: "7月22日", reward: "拥抱", status: "todo" },
  { id: "t3", title: "整理衣柜", owner: "小雨", type: "每月", due: "", reward: "", status: "todo" },
  { id: "t4", title: "陪小雨晨跑30分钟", owner: "阿杰", type: "每日", due: "", reward: "夸夸", status: "review" },
  { id: "t5", title: "预约牙医复查", owner: "小雨", type: "一次性", due: "", reward: "", status: "done" }
];

todos.forEach((item, index) => {
  item.tag = item.tag || ["心愿", "约会", "家务", "陪伴", "健康"][index] || "生活";
  item.type = ["一次性", "一次性", "每月", "每月", "每年"][index] || item.type;
  item.due = item.due || ["2026-07-25", "2026-07-22", "2026-07-31", "2026-07-20", "2026-08-01"][index] || "";
});

const schedules = [
  { id: "s1", title: "倒猫砂", cycle: "每2天", assignee: "轮流", time: "20:00", next: "今天 20:00", pending: true },
  { id: "s2", title: "浇阳台的花", cycle: "每周三", assignee: "小雨", time: "19:30", next: "周三 19:30", pending: false }
];

const dishes = [
  { id: "d1", icon: "🍳", name: "番茄炒蛋", meal: "通用", enabled: true },
  { id: "d2", icon: "🥣", name: "皮蛋瘦肉粥", meal: "早餐", enabled: true },
  { id: "d3", icon: "🥩", name: "红烧肉", meal: "晚餐", enabled: true },
  { id: "d4", icon: "🥦", name: "蒜蓉西兰花", meal: "通用", enabled: true },
  { id: "d5", icon: "🥪", name: "三明治", meal: "早餐", enabled: false },
  { id: "d6", icon: "🍲", name: "麻婆豆腐", meal: "午餐", enabled: true }
];

const orderHistory = [
  { id: "o1", date: "7月18日", meal: "晚餐", picker: "阿杰选的", dishes: ["番茄炒蛋", "蒜蓉西兰花"] },
  { id: "o2", date: "7月18日", meal: "午餐", picker: "小雨选的", dishes: ["麻婆豆腐"] },
  { id: "o3", date: "7月17日", meal: "晚餐", picker: "阿杰选的", dishes: ["红烧肉", "番茄炒蛋"] },
  { id: "o4", date: "7月17日", meal: "早餐", picker: "小雨选的", dishes: ["皮蛋瘦肉粥"] }
];

const goals = [
  { id: "g1", title: "一起存下旅行基金", period: "月目标", progress: 68, remainDays: 12, status: "active" },
  { id: "g2", title: "每周一起运动 3 次", period: "周目标", progress: 42, remainDays: 4, status: "active" },
  { id: "g3", title: "读完一本关系沟通书", period: "季度目标", progress: 100, remainDays: 0, status: "done" }
];

goals.forEach((goal, index) => {
  goals[index] = calculateGoal({
    targetValue: 100,
    currentValue: goal.progress || 0,
    startDate: "2026-07-01",
    targetDate: "2026-08-01",
    ...goal
  });
});

updateDashboard();

module.exports = {
  users,
  dashboard,
  updateLoveDate,
  updateUserProfile,
  generatePairCode,
  confirmPair,
  createMoment,
  deleteMoment,
  updateMomentLiked,
  createTask,
  deleteTask,
  updateTaskStatus,
  createSchedule,
  deleteSchedule,
  confirmSchedule,
  createDish,
  deleteDish,
  updateDishEnabled,
  createOrder,
  createGoal,
  deleteGoal,
  updateGoalStatus,
  moments,
  todos,
  schedules,
  dishes,
  orderHistory,
  goals
};
