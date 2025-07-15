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

type ServiceHandler struct {
	service service.ServiceService
}

func NewServiceHandler(service service.ServiceService) *ServiceHandler {
	return &ServiceHandler{service}
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var service model.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.Create(&service); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to create service", err.Error())
		return
	}

	util.Created(w, dto.ToServiceDTO(&service), "Service created successfully")
}

func (h *ServiceHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	services, err := h.service.GetAll()
	if err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to fetch services", err.Error())
		return
	}

	var serviceDTOs []dto.ServiceDTO
	for _, s := range services {
		serviceDTOs = append(serviceDTOs, dto.ToServiceDTO(&s))
	}

	util.Success(w, serviceDTOs, "Services fetched successfully")
}

func (h *ServiceHandler) GetByUUID(w http.ResponseWriter, r *http.Request) {
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	srv, err := h.service.GetByUUID(serviceUUID)
	if err != nil {
		util.Error(w, http.StatusNotFound, "Service not found")
		return
	}

	util.Success(w, dto.ToServiceDTO(srv), "Service fetched successfully")
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	var service model.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.UpdateByUUID(serviceUUID, &service); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	service.ServiceUUID = serviceUUID
	util.Success(w, dto.ToServiceDTO(&service), "Service updated successfully")
}

func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	serviceUUID, err := uuid.Parse(chi.URLParam(r, "service_uuid"))
	if err != nil {
		util.Error(w, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	if err := h.service.DeleteByUUID(serviceUUID); err != nil {
		util.Error(w, http.StatusInternalServerError, "Failed to delete service", err.Error())
		return
	}

	util.Success(w, nil, "Service deleted successfully")
}
