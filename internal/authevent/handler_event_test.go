package authevent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestAuthEventHandler_GetAll_NoTenant(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthEventHandler_GetAll_ValidationError(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	// invalid sort_order fails PaginationRequestDTO.Validate
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?sort_order=invalid", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthEventHandler_GetAll_InvalidCategory(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10&category=BAD", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthEventHandler_GetAll_InvalidSortOrder(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10&sort_order=bad", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthEventHandler_GetAll_ServiceError(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			return nil, assert.AnError
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthEventHandler_GetAll_Success(t *testing.T) {
	now := time.Now()
	eventUUID := uuid.New()
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			return &PaginationResult[AuthEventServiceDataResult]{
				Data: []AuthEventServiceDataResult{
					{
						AuthEventUUID: eventUUID,
						TenantID:      testTenantID,
						IPAddress:     "127.0.0.1",
						Category:      "AUTHN",
						EventType:     "authn_login_success",
						Severity:      "INFO",
						Result:        "success",
						Metadata:      datatypes.JSON(`{"key":"val"}`),
						CreatedAt:     now,
					},
				},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthEventHandler_GetAll_WithDateFilters(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			assert.NotNil(t, filter.DateFrom)
			assert.NotNil(t, filter.DateTo)
			return &PaginationResult[AuthEventServiceDataResult]{
				Data:       nil,
				Total:      0,
				Page:       1,
				Limit:      10,
				TotalPages: 0,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/auth-events?page=1&limit=10&date_from=2024-01-01T00:00:00Z&date_to=2024-12-31T23:59:59Z", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthEventHandler_GetAll_WithAllFilters(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, filter AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			assert.NotNil(t, filter.Category)
			assert.Equal(t, "AUTHN", *filter.Category)
			assert.NotNil(t, filter.Severity)
			assert.Equal(t, "INFO", *filter.Severity)
			assert.NotNil(t, filter.Result)
			assert.Equal(t, "success", *filter.Result)
			assert.NotNil(t, filter.EventType)
			assert.Equal(t, "authn_login_success", *filter.EventType)
			return &PaginationResult[AuthEventServiceDataResult]{
				Data:       nil,
				Total:      0,
				Page:       1,
				Limit:      10,
				TotalPages: 0,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/auth-events?page=1&limit=10&category=AUTHN&severity=INFO&result=success&event_type=authn_login_success", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthEventHandler_GetAll_NilMetadata(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			return &PaginationResult[AuthEventServiceDataResult]{
				Data: []AuthEventServiceDataResult{
					{
						AuthEventUUID: uuid.New(),
						TenantID:      testTenantID,
						IPAddress:     "10.0.0.1",
						Category:      "AUTHN",
						EventType:     "authn_login_fail",
						Severity:      "WARN",
						Result:        "failure",
						Metadata:      nil,
						CreatedAt:     time.Now(),
					},
				},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthEventHandler_GetAll_EmptyMetadata(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			return &PaginationResult[AuthEventServiceDataResult]{
				Data: []AuthEventServiceDataResult{
					{
						AuthEventUUID: uuid.New(),
						TenantID:      testTenantID,
						IPAddress:     "10.0.0.1",
						Category:      "AUTHN",
						EventType:     "authn_login_fail",
						Severity:      "WARN",
						Result:        "failure",
						Metadata:      datatypes.JSON(`{}`),
						CreatedAt:     time.Now(),
					},
				},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthEventHandler_GetAll_InvalidMetadataJSON(t *testing.T) {
	svc := &mockAuthEventService{
		findPaginatedFn: func(_ context.Context, _ AuthEventRepositoryGetFilter) (*PaginationResult[AuthEventServiceDataResult], error) {
			return &PaginationResult[AuthEventServiceDataResult]{
				Data: []AuthEventServiceDataResult{
					{
						AuthEventUUID: uuid.New(),
						TenantID:      testTenantID,
						IPAddress:     "10.0.0.1",
						Category:      "AUTHN",
						EventType:     "authn_login_fail",
						Severity:      "WARN",
						Result:        "failure",
						Metadata:      datatypes.JSON(`not-json`),
						CreatedAt:     time.Now(),
					},
				},
				Total:      1,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetAll(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestAuthEventHandler_Get_NoTenant(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := httptest.NewRequest(http.MethodGet, "/auth-events/"+testResourceUUID.String(), nil)
	r = withChiParam(r, "auth_event_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthEventHandler_Get_InvalidUUID(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/bad-uuid", nil))
	r = withChiParam(r, "auth_event_uuid", "bad-uuid")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthEventHandler_Get_NotFound(t *testing.T) {
	svc := &mockAuthEventService{
		findByUUIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*AuthEventServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "auth_event_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuthEventHandler_Get_ServiceError(t *testing.T) {
	svc := &mockAuthEventService{
		findByUUIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*AuthEventServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "auth_event_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthEventHandler_Get_Success(t *testing.T) {
	now := time.Now()
	eventUUID := uuid.New()
	desc := "user logged in"
	traceID := "abc123"
	svc := &mockAuthEventService{
		findByUUIDFn: func(_ context.Context, _ int64, _ uuid.UUID) (*AuthEventServiceDataResult, error) {
			return &AuthEventServiceDataResult{
				AuthEventUUID: eventUUID,
				TenantID:      testTenantID,
				IPAddress:     "127.0.0.1",
				Category:      "AUTHN",
				EventType:     "authn_login_success",
				Severity:      "INFO",
				Result:        "success",
				Description:   &desc,
				TraceID:       &traceID,
				Metadata:      datatypes.JSON(`{"key":"val"}`),
				CreatedAt:     now,
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "auth_event_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// CountByType
// ---------------------------------------------------------------------------

func TestAuthEventHandler_CountByType_NoTenant(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := httptest.NewRequest(http.MethodGet, "/auth-events/count?event_type=authn_login_success", nil)
	w := httptest.NewRecorder()
	h.CountByType(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthEventHandler_CountByType_MissingEventType(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/count", nil))
	w := httptest.NewRecorder()
	h.CountByType(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthEventHandler_CountByType_ServiceError(t *testing.T) {
	svc := &mockAuthEventService{
		countByEventTypeFn: func(_ context.Context, _ string, _ int64) (int64, error) {
			return 0, assert.AnError
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/count?event_type=authn_login_success", nil))
	w := httptest.NewRecorder()
	h.CountByType(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthEventHandler_CountByType_Success(t *testing.T) {
	svc := &mockAuthEventService{
		countByEventTypeFn: func(_ context.Context, _ string, _ int64) (int64, error) {
			return 42, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/count?event_type=authn_login_success", nil))
	w := httptest.NewRecorder()
	h.CountByType(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

func TestAuthEventHandler_Export_NoTenant(t *testing.T) {
	h := NewAuthEventHandler(&mockAuthEventService{})
	r := httptest.NewRequest(http.MethodGet, "/auth-events/export", nil)
	w := httptest.NewRecorder()
	h.Export(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthEventHandler_Export_Success(t *testing.T) {
	svc := &mockAuthEventService{
		exportFn: func(_ context.Context, filter AuthEventRepositoryGetFilter, format string) (*AuthEventExport, error) {
			assert.NotNil(t, filter.TenantID)
			assert.Equal(t, "csv", format)
			assert.Equal(t, "AUTHN", *filter.Category)
			return &AuthEventExport{
				Format:      "csv",
				ContentType: "text/csv",
				Filename:    "auth-events.csv",
				Data:        []byte("auth_event_id\n"),
			}, nil
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/export?format=csv&category=AUTHN", nil))
	w := httptest.NewRecorder()
	h.Export(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "auth_event_id")
}

func TestAuthEventHandler_Export_ServiceError(t *testing.T) {
	svc := &mockAuthEventService{
		exportFn: func(_ context.Context, _ AuthEventRepositoryGetFilter, _ string) (*AuthEventExport, error) {
			return nil, assert.AnError
		},
	}
	h := NewAuthEventHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/auth-events/export", nil))
	w := httptest.NewRecorder()
	h.Export(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
