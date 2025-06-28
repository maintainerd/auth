package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/services"
	"github.com/maintainerd/auth/internal/utils"
)

type AuthHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	token, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	utils.Created(c, gin.H{"token": token}, "Registration successful")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	utils.Success(c, gin.H{"token": token}, "Login successful")
}
