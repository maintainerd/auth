package resthandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var role model.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.Create(&role); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create role", err.Error())
		return
	}

	util.Created(w, dto.ToRoleDTO(&role), "Role created successfully")
}

func (h *RoleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.GetAll()
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch roles", err.Error())
		return
	}

	var roledto []dto.RoleDTO
	for _, role := range roles {
		roledto = append(roledto, dto.ToRoleDTO(&role))
	}

	util.Success(w, roledto, "Roles fetched successfully")
}

func (h *RoleHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	role, err := h.service.GetByUUID(roleUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Role not found")
		return
	}

	util.Success(w, dto.ToRoleDTO(role), "Role fetched successfully")
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	var role model.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.UpdateByUUID(roleUUID, &role); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update role", err.Error())
		return
	}

	role.RoleUUID = roleUUID
	util.Success(w, dto.ToRoleDTO(&role), "Role updated successfully")
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	roleUUID, err := uuid.Parse(chi.URLParam(r, "role_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid role UUID")
		return
	}

	if err := h.service.DeleteByUUID(roleUUID); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete role", err.Error())
		return
	}

	util.Success(w, nil, "Role deleted successfully")
}
