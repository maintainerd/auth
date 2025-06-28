package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dtos"
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/services"
)

type RoleHandler struct {
	service services.RoleService
}

func NewRoleHandler(service services.RoleService) *RoleHandler {
	return &RoleHandler{service}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Create(&role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dtos.ToRoleDTO(&role))
}

func (h *RoleHandler) GetAll(c *gin.Context) {
	roles, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var roleDTOs []dtos.RoleDTO
	for _, role := range roles {
		roleDTOs = append(roleDTOs, dtos.ToRoleDTO(&role))
	}

	c.JSON(http.StatusOK, roleDTOs)
}

func (h *RoleHandler) GetByUUID(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role UUID"})
		return
	}

	role, err := h.service.GetByUUID(roleUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, dtos.ToRoleDTO(role))
}

func (h *RoleHandler) Update(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role UUID"})
		return
	}

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateByUUID(roleUUID, &role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated role with correct UUID set
	role.RoleUUID = roleUUID
	c.JSON(http.StatusOK, dtos.ToRoleDTO(&role))
}

func (h *RoleHandler) Delete(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role UUID"})
		return
	}

	if err := h.service.DeleteByUUID(roleUUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}
