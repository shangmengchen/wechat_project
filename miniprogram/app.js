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
    }
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
    this.pageRefreshHandlers = {};
    this.sessionReadyPromise = this.ensureSession()
      .then(() => this.refreshSyncState())
      .then(() => {
        this.startSyncPolling();
        this.checkScheduleReminders();
      });
  },

  onShow() {
    this.refreshSyncState();
    this.checkScheduleReminders();
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
      if (prevState.version && nextState.version && Number(nextState.version) !== Number(prevState.version)) {
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

  checkScheduleReminders() {
    if (!this.globalData.demoMode && !this.globalData.isPaired) return;
    const api = require("./utils/api");
    api.getSchedules().then((res) => {
      const schedules = res.data || res || [];
      const due = dueSchedules(schedules);
      if (!due.length) return;
      const today = new Date().toISOString().slice(0, 10);
      const key = `scheduleReminder:${today}:${due.map((item) => item.id).join(",")}`;
      if (wx.getStorageSync(key)) return;
      wx.setStorageSync(key, true);
      wx.showModal({
        title: "Schedule Reminder",
        content: `${due[0].title} is due now.`,
        showCancel: false
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

function dueSchedules(schedules) {
  const now = new Date();
  const current = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  const weekdayNames = ["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"];
  const weekday = weekdayNames[now.getDay()];

  return (schedules || []).filter((item) => {
    const cycle = String(item.cycle || "").toLowerCase();
    if (!item.pending || !item.time || item.time > current) return false;
    if (cycle === "every day" || cycle === "daily" || cycle === "\u6bcf\u5929") return true;
    return cycle.includes(weekday);
  });
}
