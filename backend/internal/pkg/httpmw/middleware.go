package httpmw

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/pkg/auth"
	"couple-mini/backend/internal/pkg/adminview"
	applog "couple-mini/backend/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

const (
	requestIDKey    = "request_id"
	currentUserIDKey = "current_user_id"
	currentOpenIDKey = "current_openid"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		adminview.ObserveRequest(c.Writer.Status())
		entry := applog.Access().With(
			"request_id", GetRequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"query", c.Request.URL.RawQuery,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"remote_addr", c.Request.RemoteAddr,
			"x_forwarded_for", c.GetHeader("X-Forwarded-For"),
			"user_agent", c.Request.UserAgent(),
			"response_bytes", c.Writer.Size(),
		)

		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			entry.Error("http request completed", "errors", c.Errors.String())
		case c.Writer.Status() >= http.StatusBadRequest:
			entry.Warn("http request completed", "errors", c.Errors.String())
		default:
			entry.Info("http request completed")
		}
	}
}

func AdminBasicAuth() gin.HandlerFunc {
	cfg := configs.GetGlobalConfig().AdminConfig
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		user, pass, ok := c.Request.BasicAuth()
		if ok &&
			subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.Password)) == 1 {
			c.Next()
			return
		}
		c.Header("WWW-Authenticate", `Basic realm="Couple Mini Admin"`)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "unauthorized",
			})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		cfg := configs.GetGlobalConfig()
		claims, err := auth.ParseToken(cfg.AuthConfig.TokenSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "unauthorized",
			})
			return
		}
		c.Set(currentUserIDKey, claims.UserID)
		c.Set(currentOpenIDKey, claims.OpenID)
		c.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				applog.L().Error("panic recovered",
					"request_id", GetRequestID(c),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func GetCurrentUserID(c *gin.Context) string {
	if value, ok := c.Get(currentUserIDKey); ok {
		if userID, ok := value.(string); ok {
			return userID
		}
	}
	return ""
}

func GetCurrentOpenID(c *gin.Context) string {
	if value, ok := c.Get(currentOpenIDKey); ok {
		if openID, ok := value.(string); ok {
			return openID
		}
	}
	return ""
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
