package resthandler

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Basic validation (Chi doesn't have built-in validation like Gin binding tags)
	if req.Username == "" || req.Email == "" || req.Password == "" || len(req.Password) < 6 {
		util.Error(w, http.StatusBadRequest, "Invalid request", "Missing or invalid fields")
		return
	}

	token, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Registration failed", err.Error())
		return
	}

	util.Created(w, map[string]string{"token": token}, "Registration successful")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if req.Email == "" || req.Password == "" {
		util.Error(w, http.StatusBadRequest, "Invalid request", "Missing email or password")
		return
	}

	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		util.Error(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	util.Success(w, map[string]string{"token": token}, "Login successful")
}
