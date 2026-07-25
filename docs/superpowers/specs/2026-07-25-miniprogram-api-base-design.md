# Mini Program API Base Design

## Goal

Make the mini program frontend request the deployed backend through a single configurable API base, using the current public IP now and allowing an easy switch to domain + HTTPS later.

## Current State

- `miniprogram/app.js` hardcodes `apiBase` as `http://127.0.0.1:8080/api/v1`
- `miniprogram/utils/api.js` and `wx.uploadFile` both derive request URLs from `app.globalData.apiBase`
- WeChat DevTools disables URL checking, which helps local debugging but does not remove real-device or release constraints

## Chosen Approach

Create a small dedicated config module for backend endpoints and make `app.js` read `apiBase` from it.

### Why this approach

- Keeps the request address in exactly one place
- Avoids touching page-level business code
- Supports the current public-IP deployment immediately
- Makes the future move to domain + HTTPS a one-line config change

## Design

### Config structure

Add a config module under `miniprogram` that exposes:

- a current default API base pointing to `http://106.52.240.151/api/v1`
- a reserved production API base slot for future `https://<domain>/api/v1`

The module should keep the shape simple and readable rather than introducing a full environment system.

### App integration

`miniprogram/app.js` should import the config module and populate `globalData.apiBase` from it instead of hardcoding the value inline.

### Request layer impact

`miniprogram/utils/api.js` should remain behaviorally unchanged:

- `wx.request` continues using `app.globalData.apiBase`
- `wx.uploadFile` continues deriving its upload host from the same base URL

This keeps all existing callers stable.

## Non-goals

- No page logic changes
- No request/response contract changes
- No production domain or HTTPS rollout in this task
- No removal of mock fallback behavior in this task

## Risks and Constraints

- Public IP over HTTP is acceptable for short-term DevTools debugging, but it is not the final recommended release setup for WeChat Mini Program
- Real-device or production use should move to a legal HTTPS domain configured in the WeChat platform

## Verification

- Confirm `apiBase` is no longer hardcoded in `miniprogram/app.js`
- Confirm request and upload logic still read from `app.globalData.apiBase`
- Confirm the configured default base is `http://106.52.240.151/api/v1`
- Confirm the future switch to domain + HTTPS requires changing only the config module
