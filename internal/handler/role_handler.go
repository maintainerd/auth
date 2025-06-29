package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"github.com/maintainerd/auth/internal/util"
)

type RoleHandler struct {
	service service.RoleService
}

func NewRoleHandler(service service.RoleService) *RoleHandler {
	return &RoleHandler{service}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var role model.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.Create(&role); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to create role", err.Error())
		return
	}

	util.Created(c, dto.ToRoleDTO(&role), "Role created successfully")
}

func (h *RoleHandler) GetAll(c *gin.Context) {
	roles, err := h.service.GetAll()
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to fetch roles", err.Error())
		return
	}

	var roledto []dto.RoleDTO
	for _, role := range roles {
		roledto = append(roledto, dto.ToRoleDTO(&role))
	}

	util.Success(c, roledto, "Roles fetched successfully")
}

func (h *RoleHandler) GetByUUID(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	role, err := h.service.GetByUUID(roleUUID)
	if err != nil {
		util.Error(c, http.StatusNotFound, "Role not found")
		return
	}

	util.Success(c, dto.ToRoleDTO(role), "Role fetched successfully")
}

func (h *RoleHandler) Update(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	var role model.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.UpdateByUUID(roleUUID, &role); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	role.RoleUUID = roleUUID
	util.Success(c, dto.ToRoleDTO(&role), "Role updated successfully")
}

func (h *RoleHandler) Delete(c *gin.Context) {
	roleUUID, err := uuid.Parse(c.Param("role_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	if err := h.service.DeleteByUUID(roleUUID); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	util.Success(c, nil, "Role deleted successfully")
}
