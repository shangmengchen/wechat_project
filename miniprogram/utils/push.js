let socketTask = null;
let reconnectTimer = null;
let reconnectAttempts = 0;
let manualClose = false;
let activeToken = "";

function start() {
  const app = getApp();
  if (!app || !app.globalData || app.globalData.useMockAPI || !wx.connectSocket) {
    return;
  }

  const token = app.globalData.token || wx.getStorageSync("token") || "";
  if (!token) {
    return;
  }

  if (socketTask && activeToken === token) {
    return;
  }

  stop();
  manualClose = false;
  activeToken = token;

  const task = wx.connectSocket({
    url: eventURL(app.globalData.apiBase),
    header: {
      Authorization: `Bearer ${token}`
    }
  });

  socketTask = task;

  if (!task) {
    scheduleReconnect();
    return;
  }

  task.onOpen(() => {
    reconnectAttempts = 0;
  });

  task.onMessage((message) => {
    const event = parseEvent(message && message.data);
    if (!event || event.type !== "pair:confirmed") {
      return;
    }
    const currentApp = getApp();
    if (currentApp && typeof currentApp.handlePairConfirmed === "function") {
      currentApp.handlePairConfirmed(event.data || {});
    }
  });

  task.onClose(() => {
    if (task !== socketTask) {
      return;
    }
    socketTask = null;
    if (!manualClose) {
      scheduleReconnect();
    }
  });

  task.onError(() => {
    if (task !== socketTask) {
      return;
    }
    socketTask = null;
    task.close({});
    if (!manualClose) {
      scheduleReconnect();
    }
  });
}

function stop() {
  manualClose = true;
  clearReconnectTimer();
  if (socketTask) {
    socketTask.close({});
  }
  socketTask = null;
  activeToken = "";
}

function scheduleReconnect() {
  if (reconnectTimer) {
    return;
  }
  reconnectAttempts += 1;
  const delay = Math.min(10000, 800 * reconnectAttempts);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    start();
  }, delay);
}

function clearReconnectTimer() {
  if (!reconnectTimer) {
    return;
  }
  clearTimeout(reconnectTimer);
  reconnectTimer = null;
}

function eventURL(apiBase = "") {
  const base = String(apiBase || "").replace(/\/$/, "");
  return `${base.replace(/^https:\/\//, "wss://").replace(/^http:\/\//, "ws://")}/events`;
}

function parseEvent(raw) {
  if (!raw) {
    return null;
  }
  if (typeof raw === "object" && !(raw instanceof ArrayBuffer)) {
    return raw;
  }
  try {
    return JSON.parse(String(raw));
  } catch (err) {
    return null;
  }
}

module.exports = {
  start,
  stop,
  eventURL,
  parseEvent
};
