package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dtos"
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/services"
	"github.com/maintainerd/auth/internal/utils"
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
		utils.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.Create(&role); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to create role", err.Error())
		return
	}

	utils.Created(c, dtos.ToRoleDTO(&role), "Role created successfully")
}

func (h *RoleHandler) GetAll(c *gin.Context) {
	roles, err := h.service.GetAll()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to fetch roles", err.Error())
		return
	}

	var roleDTOs []dtos.RoleDTO
	for _, role := range roles {
		roleDTOs = append(roleDTOs, dtos.ToRoleDTO(&role))
	}

	utils.Success(c, roleDTOs, "Roles fetched successfully")
}

func (h *RoleHandler) GetByUUID(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	role, err := h.service.GetByUUID(roleUUID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "Role not found")
		return
	}

	utils.Success(c, dtos.ToRoleDTO(role), "Role fetched successfully")
}

func (h *RoleHandler) Update(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.UpdateByUUID(roleUUID, &role); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	role.RoleUUID = roleUUID
	utils.Success(c, dtos.ToRoleDTO(&role), "Role updated successfully")
}

func (h *RoleHandler) Delete(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	if err := h.service.DeleteByUUID(roleUUID); err != nil {
		utils.Error(c, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	utils.Success(c, nil, "Role deleted successfully")
}
