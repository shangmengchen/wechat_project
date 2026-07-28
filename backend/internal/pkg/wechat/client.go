package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	code2SessionURL         = "https://api.weixin.qq.com/sns/jscode2session"
	accessTokenURL          = "https://api.weixin.qq.com/cgi-bin/token"
	subscribeMessageSendURL = "https://api.weixin.qq.com/cgi-bin/message/subscribe/send"
)

type Client struct {
	appID       string
	secret      string
	client      *http.Client
	mu          sync.Mutex
	accessToken string
	tokenExpire time.Time
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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type SubscribeMessage struct {
	ToUser     string
	TemplateID string
	Page       string
	Data       map[string]string
}

type subscribeMessageRequest struct {
	ToUser           string                          `json:"touser"`
	TemplateID       string                          `json:"template_id"`
	Page             string                          `json:"page,omitempty"`
	Data             map[string]subscribeMessageData `json:"data"`
	MiniprogramState string                          `json:"miniprogram_state,omitempty"`
	Lang             string                          `json:"lang,omitempty"`
}

type subscribeMessageData struct {
	Value string `json:"value"`
}

type wechatAPIResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
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

func (c *Client) SendSubscribeMessage(ctx context.Context, message SubscribeMessage) error {
	if !c.Enabled() {
		return errors.New("wechat client not configured")
	}
	message.ToUser = strings.TrimSpace(message.ToUser)
	message.TemplateID = strings.TrimSpace(message.TemplateID)
	if message.ToUser == "" || message.TemplateID == "" {
		return errors.New("subscribe message missing touser or template id")
	}
	token, err := c.AccessToken(ctx)
	if err != nil {
		return err
	}
	payload := subscribeMessageRequest{
		ToUser:           message.ToUser,
		TemplateID:       message.TemplateID,
		Page:             strings.TrimLeft(strings.TrimSpace(message.Page), "/"),
		Data:             map[string]subscribeMessageData{},
		MiniprogramState: "formal",
		Lang:             "zh_CN",
	}
	for key, value := range message.Data {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		payload.Data[key] = subscribeMessageData{Value: value}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal subscribe message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, subscribeMessageSendURL+"?access_token="+url.QueryEscape(token), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build subscribe message request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send subscribe message: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read subscribe message response: %w", err)
	}
	var result wechatAPIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode subscribe message response: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat subscribe message failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

func (c *Client) AccessToken(ctx context.Context) (string, error) {
	if !c.Enabled() {
		return "", errors.New("wechat client not configured")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpire) {
		return c.accessToken, nil
	}
	values := url.Values{}
	values.Set("grant_type", "client_credential")
	values.Set("appid", c.appID)
	values.Set("secret", c.secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accessTokenURL+"?"+values.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("build access token request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request access token: %w", err)
	}
	defer resp.Body.Close()
	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode access token: %w", err)
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("wechat access token failed: %d %s", payload.ErrCode, payload.ErrMsg)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", errors.New("wechat access token missing")
	}
	expires := payload.ExpiresIn
	if expires <= 0 {
		expires = 7200
	}
	c.accessToken = payload.AccessToken
	c.tokenExpire = time.Now().Add(time.Duration(expires-300) * time.Second)
	return c.accessToken, nil
}
