function registerPageRefresh(page, loaderName = "load") {
  const app = getApp();
  if (!page || !page.route || !app || typeof app.registerPageRefresh !== "function") {
    return;
  }
  const handler = () => {
    if (typeof page[loaderName] === "function") {
      page[loaderName]();
    }
  };
  page.__syncRefreshHandler = handler;
  app.registerPageRefresh(page.route, handler);
}

function unregisterPageRefresh(page) {
  const app = getApp();
  if (!page || !page.route || !app || typeof app.unregisterPageRefresh !== "function") {
    return;
  }
  app.unregisterPageRefresh(page.route);
}

module.exports = {
  registerPageRefresh,
  unregisterPageRefresh
};
