package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const code2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

type Client struct {
	appID  string
	secret string
	client *http.Client
}

type Session struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
}

type code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func NewClient(appID, secret string) *Client {
	return &Client{
		appID:  strings.TrimSpace(appID),
		secret: strings.TrimSpace(secret),
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.appID != "" && c.secret != ""
}

func (c *Client) Code2Session(ctx context.Context, code string) (Session, error) {
	if !c.Enabled() {
		return Session{}, errors.New("wechat auth not configured")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return Session{}, errors.New("login code required")
	}
	values := url.Values{}
	values.Set("appid", c.appID)
	values.Set("secret", c.secret)
	values.Set("js_code", code)
	values.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, code2SessionURL+"?"+values.Encode(), nil)
	if err != nil {
		return Session{}, fmt.Errorf("build wechat request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return Session{}, fmt.Errorf("request wechat session: %w", err)
	}
	defer resp.Body.Close()

	var payload code2SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Session{}, fmt.Errorf("decode wechat session: %w", err)
	}
	if payload.ErrCode != 0 {
		return Session{}, fmt.Errorf("wechat auth failed: %d %s", payload.ErrCode, payload.ErrMsg)
	}
	if strings.TrimSpace(payload.OpenID) == "" {
		return Session{}, errors.New("wechat auth missing openid")
	}
	return Session{
		OpenID:     payload.OpenID,
		SessionKey: payload.SessionKey,
		UnionID:    payload.UnionID,
	}, nil
}
