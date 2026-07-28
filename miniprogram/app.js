const NOTICE_STORAGE_KEY = "unreadNotices";
const REMINDER_INTERVAL = 60 * 1000;
const REMINDER_TICK_OFFSET = 500;

App({
  globalData: {
    apiBase: "http://106.52.240.151/api/v1",
    token: "",
    currentUserId: "",
    demoMode: false,
    useMockAPI: false,
    isPaired: false,
    userProfile: {
      nickname: "",
      avatar: ""
    },
    syncState: {
      paired: false,
      coupleId: "",
      version: 0,
      updatedAt: ""
    },
    unreadNotices: {}
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
    this.globalData.useMockAPI = !!wx.getStorageSync("useMockAPI");
    this.globalData.isPaired = !!wx.getStorageSync("isPaired");
    this.globalData.userProfile = normalizeProfile(wx.getStorageSync("userProfile"));
    this.globalData.unreadNotices = this.loadUnreadNotices();
    this.pageRefreshHandlers = {};
    this.ensureSession()
      .then(() => {
        this.startPairPush();
        return this.refreshSyncState();
      })
      .then(() => {
        this.startSyncPolling();
        this.startReminderPolling();
        this.syncUnreadNotices();
        this.checkScheduleReminders();
      });
  },

  onShow() {
    this.startPairPush();
    this.refreshSyncState();
    this.syncUnreadNotices();
    this.startReminderPolling();
    this.checkScheduleReminders();
  },

  onHide() {
    this.stopReminderPolling();
  },

  ensureSession(profilePatch = {}) {
    const nextProfile = normalizeProfile({
      ...(this.globalData.userProfile || {}),
      ...(profilePatch || {})
    });
    this.globalData.userProfile = nextProfile;
    wx.setStorageSync("userProfile", nextProfile);

    if (this.globalData.useMockAPI) {
      return Promise.resolve(null);
    }

    if (Object.keys(profilePatch || {}).length === 0 && this.sessionReadyPromise) {
      return this.sessionReadyPromise;
    }

    this.sessionReadyPromise = new Promise((resolve) => {
      if (!wx.login) {
        resolve(null);
        return;
      }

      wx.login({
        success: (loginRes) => {
          wx.request({
            url: `${this.globalData.apiBase}/auth/login`,
            method: "POST",
            data: {
              userId: this.globalData.currentUserId,
              code: loginRes.code || "",
              nickname: nextProfile.nickname || "",
              avatar: nextProfile.avatar || ""
            },
            success: (res) => {
              const payload = (res.data && res.data.data) || {};
              const user = payload.user || {};
              const token = payload.token || "";

              if (token) {
                this.globalData.token = token;
                wx.setStorageSync("token", token);
                this.startPairPush();
              }

              if (user.id) {
                this.globalData.currentUserId = user.id;
                wx.setStorageSync("currentUserId", user.id);
              }

              const mergedProfile = normalizeProfile({
                nickname: user.nickname || nextProfile.nickname,
                avatar: user.avatar || nextProfile.avatar
              });
              this.globalData.userProfile = mergedProfile;
              wx.setStorageSync("userProfile", mergedProfile);
              resolve(payload);
            },
            fail: () => resolve(null)
          });
        },
        fail: () => resolve(null)
      });
    });

    return this.sessionReadyPromise;
  },

  syncUserProfile(profilePatch = {}) {
    return this.ensureSession(profilePatch).then(() => this.refreshSyncState());
  },

  startPairPush() {
    if (this.globalData.useMockAPI) {
      return;
    }
    const push = require("./utils/push");
    push.start();
  },

  handlePairConfirmed(data = {}) {
    this.globalData.syncState = {
      paired: true,
      coupleId: data.coupleId || (this.globalData.syncState && this.globalData.syncState.coupleId) || "",
      version: Date.now(),
      updatedAt: new Date().toISOString()
    };
    this.globalData.isPaired = true;
    this.globalData.demoMode = false;
    wx.setStorageSync("isPaired", true);
    wx.setStorageSync("demoMode", false);
    this.gotoPairSuccess();
  },

  handlePairUnbound(data = {}) {
    const wasPaired = data.wasPaired !== undefined
      ? !!data.wasPaired
      : !!(this.globalData.isPaired || wx.getStorageSync("isPaired"));
    this.globalData.syncState = {
      paired: false,
      coupleId: "",
      version: Date.now(),
      updatedAt: new Date().toISOString()
    };
    this.globalData.isPaired = false;
    this.globalData.demoMode = false;
    this.globalData.unreadNotices = {};
    wx.setStorageSync("isPaired", false);
    wx.setStorageSync("demoMode", false);
    wx.setStorageSync(NOTICE_STORAGE_KEY, {});
    this.applyTabRedDots();

    const currentUserId = this.globalData.currentUserId || wx.getStorageSync("currentUserId");
    if (currentUserId) {
      wx.removeStorageSync(`wechatProfileSynced:${currentUserId}`);
    }
    const isInitiator = data.initiatorId && data.initiatorId === currentUserId;
    this.gotoPairPage(() => {
      if (!wasPaired || isInitiator || data.silent) {
        return;
      }
      this.showUnboundNotice(data.initiatorId === "admin" ? "管理员已解除你们的绑定关系。" : "对方已解除绑定关系。");
    });
  },

  showUnboundNotice(content) {
    if (this.unboundNoticeShowing) {
      return;
    }
    this.unboundNoticeShowing = true;
    wx.showModal({
      title: "绑定已解除",
      content,
      showCancel: false,
      confirmText: "重新匹配",
      complete: () => {
        this.unboundNoticeShowing = false;
      }
    });
  },

  handleAppNotice(data = {}) {
    const notice = normalizeNotice(data);
    const currentUserId = this.globalData.currentUserId || wx.getStorageSync("currentUserId");
    if (!notice.category || (notice.initiatorId && notice.initiatorId === currentUserId)) {
      return;
    }
    if (notice.recipientId && notice.recipientId !== currentUserId) {
      return;
    }

    this.addUnreadNotice(notice.category);
    this.refreshUnreadOnActivePage();
    this.showNoticeModal(notice);
  },

  showNoticeModal(notice) {
    if (this.noticeModalShowing) {
      this.noticeQueue = this.noticeQueue || [];
      this.noticeQueue.push(notice);
      return;
    }
    this.noticeModalShowing = true;
    wx.showModal({
      title: notice.title || "新的提醒",
      content: notice.content || "对方有新的更新，记得去看看。",
      cancelText: "稍后看",
      confirmText: "去看看",
      success: (res) => {
        if (res.confirm && notice.target) {
          navigateToNoticeTarget(notice.target);
        }
      },
      complete: () => {
        this.noticeModalShowing = false;
        const next = this.noticeQueue && this.noticeQueue.shift();
        if (next) {
          setTimeout(() => this.showNoticeModal(next), 300);
        }
      }
    });
  },

  loadUnreadNotices() {
    return normalizeCounts(wx.getStorageSync(NOTICE_STORAGE_KEY));
  },

  syncUnreadNotices() {
    if (this.globalData.useMockAPI || !this.globalData.isPaired) {
      this.applyTabRedDots();
      return Promise.resolve(this.globalData.unreadNotices || {});
    }
    const api = require("./utils/api");
    return api.getUnreadNotices().then((res) => {
      if (res && res.code !== undefined && res.code !== 0) {
        return this.globalData.unreadNotices || {};
      }
      const payload = (res && res.data) || res || {};
      const counts = payload.counts || countsFromNotices(payload.items || []);
      this.globalData.unreadNotices = normalizeCounts(counts);
      wx.setStorageSync(NOTICE_STORAGE_KEY, this.globalData.unreadNotices);
      this.applyTabRedDots();
      this.refreshUnreadOnActivePage();
      return this.globalData.unreadNotices;
    }).catch(() => {
      this.applyTabRedDots();
      return this.globalData.unreadNotices || {};
    });
  },

  addUnreadNotice(category) {
    const key = normalizeCategory(category);
    if (!key) return;
    const unread = normalizeCounts(this.globalData.unreadNotices || this.loadUnreadNotices());
    unread[key] = (unread[key] || 0) + 1;
    this.globalData.unreadNotices = unread;
    wx.setStorageSync(NOTICE_STORAGE_KEY, unread);
    this.applyTabRedDots();
  },

  clearUnreadCategory(categories) {
    const list = Array.isArray(categories) ? categories : [categories];
    const normalized = list.map(normalizeCategory).filter(Boolean);
    if (!normalized.length) {
      return;
    }
    const unread = normalizeCounts(this.globalData.unreadNotices || this.loadUnreadNotices());
    normalized.forEach((category) => {
      delete unread[category];
    });
    this.globalData.unreadNotices = unread;
    wx.setStorageSync(NOTICE_STORAGE_KEY, unread);
    this.applyTabRedDots();
    this.refreshUnreadOnActivePage();

    if (!this.globalData.useMockAPI && this.globalData.isPaired) {
      const api = require("./utils/api");
      api.markNoticesRead(normalized).catch(() => null);
    }
  },

  getUnreadFlags() {
    const unread = normalizeCounts(this.globalData.unreadNotices || this.loadUnreadNotices());
    return {
      memory: !!unread.memory,
      todos: !!unread.todos,
      schedule: !!unread.schedule,
      order: !!unread.order,
      goals: !!unread.goals,
      home: !!(unread.schedule || unread.order || unread.goals)
    };
  },

  applyTabRedDots() {
    if (!wx.showTabBarRedDot || !wx.hideTabBarRedDot) {
      return;
    }
    const flags = this.getUnreadFlags();
    setTabRedDot(0, flags.home);
    setTabRedDot(1, flags.memory);
    setTabRedDot(2, flags.todos);
    setTabRedDot(3, false);
  },

  refreshUnreadOnActivePage() {
    const pages = getCurrentPages();
    if (!pages || !pages.length) return;
    const page = pages[pages.length - 1];
    if (page && typeof page.refreshUnread === "function") {
      page.refreshUnread();
    }
  },

  gotoPairSuccess() {
    const pages = getCurrentPages();
    const currentPage = pages && pages.length ? pages[pages.length - 1] : null;
    if (currentPage && currentPage.route === "pages/pair-success/pair-success") {
      return;
    }
    if (this.pairSuccessRedirecting) {
      return;
    }
    this.pairSuccessRedirecting = true;
    wx.redirectTo({
      url: "/pages/pair-success/pair-success",
      fail: () => wx.reLaunch({ url: "/pages/pair-success/pair-success" }),
      complete: () => {
        setTimeout(() => {
          this.pairSuccessRedirecting = false;
        }, 500);
      }
    });
  },

  gotoPairPage(afterNavigate) {
    const pages = getCurrentPages();
    const currentPage = pages && pages.length ? pages[pages.length - 1] : null;
    if (currentPage && currentPage.route === "pages/pair/pair") {
      if (typeof afterNavigate === "function") afterNavigate();
      return;
    }
    wx.reLaunch({
      url: "/pages/pair/pair",
      complete: () => {
        if (typeof afterNavigate === "function") afterNavigate();
      }
    });
  },

  registerPageRefresh(route, handler) {
    if (!route || typeof handler !== "function") return;
    this.pageRefreshHandlers[route] = handler;
  },

  unregisterPageRefresh(route) {
    if (!route) return;
    delete this.pageRefreshHandlers[route];
  },

  startSyncPolling() {
    this.stopSyncPolling();
    this.syncTimer = setInterval(() => {
      this.refreshSyncState();
      this.syncUnreadNotices();
    }, 3000);
  },

  stopSyncPolling() {
    if (!this.syncTimer) return;
    clearInterval(this.syncTimer);
    this.syncTimer = null;
  },

  refreshSyncState() {
    if (this.globalData.useMockAPI) {
      return Promise.resolve(this.globalData.syncState);
    }
    const api = require("./utils/api");
    return api.getSyncState().then((res) => {
      const nextState = (res && res.data) || res || {};
      const prevState = this.globalData.syncState || {};
      this.globalData.syncState = {
        paired: !!nextState.paired,
        coupleId: nextState.coupleId || "",
        version: Number(nextState.version || 0),
        updatedAt: nextState.updatedAt || ""
      };
      this.globalData.isPaired = !!nextState.paired;
      wx.setStorageSync("isPaired", !!nextState.paired);
      if (prevState.paired && !nextState.paired) {
        this.handlePairUnbound({ silent: false, wasPaired: true });
        return this.globalData.syncState;
      }
      if (
        !!nextState.paired !== !!prevState.paired ||
        (prevState.version && nextState.version && Number(nextState.version) !== Number(prevState.version))
      ) {
        this.refreshActivePage();
      }
      return this.globalData.syncState;
    }).catch(() => this.globalData.syncState);
  },

  refreshActivePage() {
    const pages = getCurrentPages();
    if (!pages || !pages.length) return;
    const page = pages[pages.length - 1];
    if (!page || !page.route) return;
    const handler = this.pageRefreshHandlers[page.route];
    if (typeof handler === "function") {
      handler();
    }
  },

  startReminderPolling() {
    this.stopReminderPolling();
    this.scheduleNextReminderTick();
  },

  stopReminderPolling() {
    if (!this.reminderTimer) return;
    clearTimeout(this.reminderTimer);
    this.reminderTimer = null;
  },

  scheduleNextReminderTick() {
    const now = new Date();
    const delay = Math.max(
      500,
      REMINDER_INTERVAL - (now.getSeconds() * 1000 + now.getMilliseconds()) + REMINDER_TICK_OFFSET
    );
    this.reminderTimer = setTimeout(() => {
      this.reminderTimer = null;
      this.checkScheduleReminders({ catchUp: false });
      this.scheduleNextReminderTick();
    }, delay);
  },

  checkScheduleReminders(options = {}) {
    if (!this.globalData.demoMode && !this.globalData.isPaired) return;
    const api = require("./utils/api");
    api.getSchedules().then((res) => {
      const schedules = res.data || res || [];
      const due = dueSchedules(schedules, options);
      if (!due.length) return;
      const today = new Date().toISOString().slice(0, 10);
      const key = `scheduleReminder:${today}:${due.map((item) => item.id).join(",")}`;
      if (wx.getStorageSync(key)) return;
      wx.setStorageSync(key, true);
      wx.showModal({
        title: "定时任务提醒",
        content: due.length > 1 ? `${due[0].title} 等 ${due.length} 个任务到时间啦` : `${due[0].title} 到时间啦`,
        showCancel: false,
        confirmText: "去处理",
        success: (modalRes) => {
          if (modalRes.confirm) {
            navigateToNoticeTarget("/pages/schedule/schedule");
          }
        }
      });
    });
  }
});

function normalizeProfile(profile) {
  return {
    nickname: (profile && profile.nickname) || "",
    avatar: (profile && profile.avatar) || ""
  };
}

function normalizeNotice(data = {}) {
  return {
    id: data.id || "",
    recipientId: data.recipientId || "",
    initiatorId: data.initiatorId || "",
    category: normalizeCategory(data.category),
    title: data.title || "新的提醒",
    content: data.content || "",
    target: data.target || targetByCategory(data.category)
  };
}

function normalizeCategory(category) {
  const value = String(category || "").toLowerCase();
  const map = {
    moment: "memory",
    memory: "memory",
    task: "todos",
    todo: "todos",
    todos: "todos",
    scheduledtask: "schedule",
    schedule: "schedule",
    dish: "order",
    order: "order",
    goal: "goals",
    goals: "goals"
  };
  return map[value] || "";
}

function targetByCategory(category) {
  const key = normalizeCategory(category);
  const map = {
    memory: "/pages/memory/memory",
    todos: "/pages/todos/todos",
    schedule: "/pages/schedule/schedule",
    order: "/pages/order/order",
    goals: "/pages/goal/goals"
  };
  return map[key] || "";
}

function normalizeCounts(counts) {
  const next = {};
  Object.keys(counts || {}).forEach((category) => {
    const key = normalizeCategory(category);
    const value = Number(counts[category] || 0);
    if (key && value > 0) {
      next[key] = (next[key] || 0) + value;
    }
  });
  return next;
}

function countsFromNotices(items) {
  return (items || []).reduce((counts, item) => {
    const key = normalizeCategory(item.category);
    if (key) counts[key] = (counts[key] || 0) + 1;
    return counts;
  }, {});
}

function setTabRedDot(index, visible) {
  const method = visible ? wx.showTabBarRedDot : wx.hideTabBarRedDot;
  method({
    index,
    fail: () => null
  });
}

function navigateToNoticeTarget(url) {
  const target = String(url || "");
  if (!target) return;
  const tabPages = ["/pages/home/home", "/pages/memory/memory", "/pages/todos/todos", "/pages/mine/mine"];
  if (tabPages.includes(target)) {
    wx.switchTab({ url: target });
    return;
  }
  wx.navigateTo({
    url: target,
    fail: () => wx.switchTab({ url: "/pages/home/home" })
  });
}

function dueSchedules(schedules, options = {}) {
  const catchUp = options.catchUp !== false;
  const now = new Date();
  const current = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  const weekdayNames = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
  const englishWeekdayNames = ["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"];
  const weekday = weekdayNames[now.getDay()];
  const englishWeekday = englishWeekdayNames[now.getDay()];

  return (schedules || []).filter((item) => {
    const cycle = String(item.cycle || "").toLowerCase();
    if (!item.pending || !item.time) return false;
    if (catchUp ? item.time > current : item.time !== current) return false;
    if (cycle === "every day" || cycle === "daily" || cycle === "每天") return true;
    return cycle.includes(weekday) || cycle.includes(englishWeekday);
  });
}
