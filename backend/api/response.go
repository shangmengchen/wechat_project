package api

import (
	"errors"
	"net/http"

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
	c.JSON(status, gin.H{"code": status, "message": message})
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": message})
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		badRequest(c, "invalid json")
		return false
	}
	return true
}
