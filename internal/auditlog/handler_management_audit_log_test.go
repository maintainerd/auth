package auditlog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type mockManagementAuditLogRepo struct {
	createFn                func(record *ManagementAuditLog) error
	findPaginatedFn         func(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error)
	findByUUIDAndTenantIDFn func(auditLogUUID uuid.UUID, tenantID int64) (*ManagementAuditLog, error)
}

func (m *mockManagementAuditLogRepo) Create(record *ManagementAuditLog) error {
	if m.createFn != nil {
		return m.createFn(record)
	}
	return nil
}

func (m *mockManagementAuditLogRepo) FindPaginated(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(tenantID, filter)
	}
	return nil, 0, nil
}

func (m *mockManagementAuditLogRepo) FindByUUIDAndTenantID(auditLogUUID uuid.UUID, tenantID int64) (*ManagementAuditLog, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(auditLogUUID, tenantID)
	}
	return nil, nil
}

func managementAuditLogRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	return middleware.WithAuthContext(req, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: 42},
	})
}

func decodeResponseData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	return data
}

func TestManagementAuditLogHandler_ListIncludesActorNames(t *testing.T) {
	entryUUID := uuid.New()
	actorName := "Jane Admin"
	actorUserID := int64(7)

	repo := &mockManagementAuditLogRepo{
		findPaginatedFn: func(tenantID int64, filter ManagementAuditLogFilter) ([]ManagementAuditLog, int64, error) {
			require.Equal(t, int64(42), tenantID)
			return []ManagementAuditLog{
				{
					ManagementAuditLogUUID: entryUUID,
					TenantID:               &tenantID,
					ActorUserID:            &actorUserID,
					ActorUserName:          &actorName,
					Action:                 "user.updated",
					ResourceType:           "user",
					ResourceID:             "user-123",
					Changes:                datatypes.JSON(`{"after":{"status":"active"}}`),
					Outcome:                "success",
					CreatedAt:              time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
				},
			}, 1, nil
		},
	}

	rec := httptest.NewRecorder()
	NewManagementAuditLogHandler(repo).List(rec, managementAuditLogRequest(http.MethodGet, "/management-audit-log"))

	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeResponseData(t, rec)
	rows := data["rows"].([]any)
	first := rows[0].(map[string]any)
	assert.Equal(t, "Jane Admin", first["actor_user_name"])
	assert.Nil(t, first["actor_user_id"], "internal integer id must never be serialized outbound")
	assert.Equal(t, float64(1), data["total_pages"])
}

func TestManagementAuditLogHandler_Get(t *testing.T) {
	entryUUID := uuid.New()

	t.Run("returns enriched audit log entry", func(t *testing.T) {
		clientName := "Auth Console"
		actorClientID := int64(11)
		repo := &mockManagementAuditLogRepo{
			findByUUIDAndTenantIDFn: func(auditLogUUID uuid.UUID, tenantID int64) (*ManagementAuditLog, error) {
				require.Equal(t, entryUUID, auditLogUUID)
				require.Equal(t, int64(42), tenantID)
				return &ManagementAuditLog{
					ManagementAuditLogUUID: entryUUID,
					TenantID:               &tenantID,
					ActorClientID:          &actorClientID,
					ActorClientName:        &clientName,
					Action:                 "client.updated",
					ResourceType:           "client",
					ResourceID:             "client-123",
					Changes:                datatypes.JSON(`{}`),
					Outcome:                "success",
					CreatedAt:              time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
				}, nil
			},
		}
		router := chi.NewRouter()
		router.Get("/management-audit-log/{audit_log_uuid}", NewManagementAuditLogHandler(repo).Get)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, managementAuditLogRequest(http.MethodGet, "/management-audit-log/"+entryUUID.String()))

		require.Equal(t, http.StatusOK, rec.Code)
		data := decodeResponseData(t, rec)
		assert.Equal(t, "Auth Console", data["actor_client_name"])
		assert.Nil(t, data["actor_client_id"], "internal integer id must never be serialized outbound")
	})

	t.Run("rejects invalid UUID", func(t *testing.T) {
		router := chi.NewRouter()
		router.Get("/management-audit-log/{audit_log_uuid}", NewManagementAuditLogHandler(&mockManagementAuditLogRepo{}).Get)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, managementAuditLogRequest(http.MethodGet, "/management-audit-log/not-a-uuid"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns not found when the entry does not belong to tenant", func(t *testing.T) {
		repo := &mockManagementAuditLogRepo{
			findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*ManagementAuditLog, error) {
				return nil, nil
			},
		}
		router := chi.NewRouter()
		router.Get("/management-audit-log/{audit_log_uuid}", NewManagementAuditLogHandler(repo).Get)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, managementAuditLogRequest(http.MethodGet, "/management-audit-log/"+entryUUID.String()))

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("returns server error when repository fails", func(t *testing.T) {
		repo := &mockManagementAuditLogRepo{
			findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*ManagementAuditLog, error) {
				return nil, errors.New("db down")
			},
		}
		router := chi.NewRouter()
		router.Get("/management-audit-log/{audit_log_uuid}", NewManagementAuditLogHandler(repo).Get)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, managementAuditLogRequest(http.MethodGet, "/management-audit-log/"+entryUUID.String()))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
