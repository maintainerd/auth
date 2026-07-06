package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

type UserTrustedDeviceHandler struct {
	deviceService UserTrustedDeviceService
	userService   UserService
	userRepo      UserRepository
	auditLogger   auditlog.ManagementAuditLogger
}

func NewUserTrustedDeviceHandler(deviceService UserTrustedDeviceService, userService UserService, userRepo UserRepository) *UserTrustedDeviceHandler {
	return &UserTrustedDeviceHandler{
		deviceService: deviceService,
		userService:   userService,
		userRepo:      userRepo,
	}
}

// SetAuditLogger injects the audit logger (called by the wiring layer).
func (h *UserTrustedDeviceHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) {
	h.auditLogger = l
}

func (h *UserTrustedDeviceHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
}

type UserTrustedDeviceResponseDTO struct {
	UUID              string `json:"uuid"`
	DeviceFingerprint string `json:"device_fingerprint"`
	DeviceName        string `json:"device_name,omitempty"`
	IPAddress         string `json:"ip_address,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
	TrustedUntil      string `json:"trusted_until"`
	LastSeenAt        string `json:"last_seen_at,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// ListMyDevices returns all trusted devices for the authenticated user.
//
// GET /me/devices
func (h *UserTrustedDeviceHandler) ListMyDevices(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	devices, err := h.deviceService.ListDevices(r.Context(), auth.User.UserID)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve devices")
		return
	}

	dtos := make([]UserTrustedDeviceResponseDTO, 0, len(devices))
	for _, d := range devices {
		dtos = append(dtos, toDeviceDTO(d))
	}

	resp.Success(w, dtos, "Devices retrieved successfully")
}

// DeleteMyDevice removes a trusted device for the authenticated user.
//
// DELETE /me/devices/{device_uuid}
func (h *UserTrustedDeviceHandler) DeleteMyDevice(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromRequest(r)
	if auth.User == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	deviceUUID := chi.URLParam(r, "device_uuid")
	if deviceUUID == "" {
		resp.Error(w, http.StatusBadRequest, "Missing device_uuid")
		return
	}

	if err := h.deviceService.DeleteDevice(r.Context(), deviceUUID); err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	tenantIDDD := int64(0)
	if auth.Tenant != nil {
		tenantIDDD = auth.Tenant.TenantID
	}
	actorUserIDDD := &auth.User.UserID
	changesJSONDD, _ := json.Marshal(map[string]any{"before": map[string]any{"device_uuid": deviceUUID}})
	h.logAudit(r, tenantIDDD, actorUserIDDD, "device.delete", "trusted_device", deviceUUID, nil, string(changesJSONDD), "success")

	resp.Success(w, nil, "Device removed successfully")
}

// GetUserDevices returns all trusted devices for a specific user (admin).
//
// GET /users/{user_uuid}/devices
func (h *UserTrustedDeviceHandler) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	userUUID := chi.URLParam(r, "user_uuid")
	if userUUID == "" {
		resp.Error(w, http.StatusBadRequest, "Missing user_uuid")
		return
	}

	if _, err := uuid.Parse(userUUID); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid user_uuid format")
		return
	}

	u, err := h.userRepo.FindByUUID(userUUID)
	if err != nil || u == nil {
		resp.Error(w, http.StatusNotFound, "User not found")
		return
	}

	devices, err := h.deviceService.ListDevices(r.Context(), u.UserID)
	if err != nil {
		resp.Error(w, http.StatusInternalServerError, "Failed to retrieve devices")
		return
	}

	dtos := make([]UserTrustedDeviceResponseDTO, 0, len(devices))
	for _, d := range devices {
		dtos = append(dtos, toDeviceDTO(d))
	}

	resp.Success(w, dtos, "User devices retrieved successfully")
}

func toDeviceDTO(d UserTrustedDevice) UserTrustedDeviceResponseDTO {
	dto := UserTrustedDeviceResponseDTO{
		UUID:              d.UserTrustedDeviceUUID.String(),
		DeviceFingerprint: d.DeviceFingerprint,
		TrustedUntil:      d.TrustedUntil.Format("2006-01-02T15:04:05Z"),
		CreatedAt:         d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if d.DeviceName != nil {
		dto.DeviceName = *d.DeviceName
	}
	if d.IPAddress != nil {
		dto.IPAddress = *d.IPAddress
	}
	if d.UserAgent != nil {
		dto.UserAgent = *d.UserAgent
	}
	if d.LastSeenAt != nil {
		dto.LastSeenAt = d.LastSeenAt.Format("2006-01-02T15:04:05Z")
	}
	return dto
}
