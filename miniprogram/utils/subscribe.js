const api = require("./api");

const COOLDOWN_MS = 6 * 60 * 60 * 1000;
const STORAGE_PREFIX = "subscribePromptedAt";

function request(category = "notice") {
  if (!wx.requestSubscribeMessage) {
    return Promise.resolve(false);
  }
  const key = `${STORAGE_PREFIX}:${category}`;
  const lastPromptedAt = Number(wx.getStorageSync(key) || 0);
  if (Date.now() - lastPromptedAt < COOLDOWN_MS) {
    return Promise.resolve(false);
  }
  return api.getSubscribeTemplates().then((res) => {
    if (res && res.code !== undefined && res.code !== 0) {
      return false;
    }
    const data = (res && res.data) || res || {};
    const ids = templateIds(data, category);
    if (!ids.length) {
      return false;
    }
    return new Promise((resolve) => {
      wx.requestSubscribeMessage({
        tmplIds: ids,
        complete: () => {
          wx.setStorageSync(key, Date.now());
          resolve(true);
        }
      });
    });
  }).catch(() => false);
}

function templateIds(data, category) {
  const ids = [];
  if (category === "schedule" && data.schedule) {
    ids.push(data.schedule);
  }
  if (data.notice) {
    ids.push(data.notice);
  }
  return Array.from(new Set(ids.filter(Boolean)));
}

module.exports = {
  request
};
