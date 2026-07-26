package api

import (
	"errors"
	"net/http"

	"couple-mini/backend/internal/pkg/httpmw"
	applog "couple-mini/backend/internal/pkg/logger"
	"couple-mini/backend/internal/repo"

	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func respond(c *gin.Context, data any, err error) {
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, data)
}

func fail(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if errors.Is(err, repo.ErrNotFound) {
		status = http.StatusNotFound
		message = "resource not found"
	}
	if errors.Is(err, repo.ErrInvalidPairCode) {
		status = http.StatusBadRequest
		message = "invalid pair code"
	}
	if errors.Is(err, repo.ErrPairCodeExpired) {
		status = http.StatusBadRequest
		message = "pair code expired"
	}
	if errors.Is(err, repo.ErrAlreadyPaired) {
		status = http.StatusBadRequest
		message = "already paired"
	}
	if errors.Is(err, repo.ErrUnauthorized) {
		status = http.StatusUnauthorized
		message = "unauthorized"
	}
	logRequestError(c, err, status, message)
	c.JSON(status, gin.H{"code": status, "message": message})
}

func badRequest(c *gin.Context, message string) {
	logRequestError(c, errors.New(message), http.StatusBadRequest, message)
	c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": message})
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "invalid json"})
		logRequestError(c, err, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

func logRequestError(c *gin.Context, err error, status int, message string) {
	entry := applog.L().With(
		"request_id", httpmw.GetRequestID(c),
		"status", status,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	if status >= http.StatusInternalServerError {
		entry.Error("request failed", "message", message, "error", err.Error())
		return
	}
	entry.Warn("request rejected", "message", message, "error", err.Error())
}
