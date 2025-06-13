package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func Created(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func Error(c *gin.Context, status int, err string, details ...string) {
	resp := gin.H{
		"success": false,
		"error":   err,
	}
	if len(details) > 0 {
		resp["details"] = details[0]
	}
	c.JSON(status, resp)
}
