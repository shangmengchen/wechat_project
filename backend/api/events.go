package api

import (
	"net/http"
	"strings"

	"couple-mini/backend/configs"
	"couple-mini/backend/internal/pkg/auth"

	"github.com/gin-gonic/gin"
)

func (api *API) PairEvents(c *gin.Context) {
	if api.push == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "push service unavailable",
		})
		return
	}

	token := eventToken(c)
	claims, err := auth.ParseToken(configs.GetGlobalConfig().AuthConfig.TokenSecret, token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
		})
		return
	}

	api.push.Serve(c, claims.UserID)
}

func eventToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}

	token := strings.TrimSpace(c.Query("token"))
	if token != "" {
		return token
	}

	for _, protocol := range strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",") {
		protocol = strings.TrimSpace(protocol)
		if strings.HasPrefix(protocol, "Bearer.") {
			return strings.TrimPrefix(protocol, "Bearer.")
		}
	}
	return ""
}
