package resthandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *ServiceHandler) Create(c *gin.Context) {
	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.Create(&service); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to create service", err.Error())
		return
	}

	util.Created(c, dto.ToServiceDTO(&service), "Service created successfully")
}

func (h *ServiceHandler) GetAll(c *gin.Context) {
	services, err := h.service.GetAll()
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to fetch services", err.Error())
		return
	}

	var serviceDTOs []dto.ServiceDTO
	for _, service := range services {
		serviceDTOs = append(serviceDTOs, dto.ToServiceDTO(&service))
	}

	util.Success(c, serviceDTOs, "Services fetched successfully")
}

func (h *ServiceHandler) GetByUUID(c *gin.Context) {
	serviceUUID, err := uuid.Parse(c.Param("service_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	service, err := h.service.GetByUUID(serviceUUID)
	if err != nil {
		util.Error(c, http.StatusNotFound, "Service not found")
		return
	}

	util.Success(c, dto.ToServiceDTO(service), "Service fetched successfully")
}

func (h *ServiceHandler) Update(c *gin.Context) {
	serviceUUID, err := uuid.Parse(c.Param("service_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := h.service.UpdateByUUID(serviceUUID, &service); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to update service", err.Error())
		return
	}

	service.ServiceUUID = serviceUUID
	util.Success(c, dto.ToServiceDTO(&service), "Service updated successfully")
}

func (h *ServiceHandler) Delete(c *gin.Context) {
	serviceUUID, err := uuid.Parse(c.Param("service_uuid"))
	if err != nil {
		util.Error(c, http.StatusBadRequest, "Invalid service UUID")
		return
	}

	if err := h.service.DeleteByUUID(serviceUUID); err != nil {
		util.Error(c, http.StatusInternalServerError, "Failed to delete service", err.Error())
		return
	}

	util.Success(c, nil, "Service deleted successfully")
}
