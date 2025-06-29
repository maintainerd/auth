package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	token, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	util.Created(c, gin.H{"token": token}, "Registration successful")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		util.Error(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	util.Success(c, gin.H{"token": token}, "Login successful")
}
