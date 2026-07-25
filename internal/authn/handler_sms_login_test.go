package authn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSMSLoginService struct {
	sendOTPFn   func(ctx context.Context, phone string, clientID, tenantID *string) error
	verifyOTPFn func(ctx context.Context, phone, otp string, clientID, tenantID *string) (*LoginResponseDTO, error)
}

func (m *mockSMSLoginService) SendOTP(ctx context.Context, phone string, clientID, tenantID *string) error {
	if m.sendOTPFn != nil {
		return m.sendOTPFn(ctx, phone, clientID, tenantID)
	}
	return nil
}

func (m *mockSMSLoginService) VerifyOTP(ctx context.Context, phone, otp string, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if m.verifyOTPFn != nil {
		return m.verifyOTPFn(ctx, phone, otp, clientID, tenantID)
	}
	return nil, nil
}

func (m *mockSMSLoginService) SetMFACoordinator(SMSMFACoordinator) {}

func smsJSONReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestSMSLoginHandler_SendOTPPublic(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/send?client_id=app",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("tenant_id present on public returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?client_id=app&tenant_id=acme",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?client_id=app",
			map[string]string{"phone": ""})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSMSLoginService{
			sendOTPFn: func(ctx context.Context, phone string, clientID, tenantID *string) error {
				return errors.New("db error")
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?client_id=app",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?client_id=app",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSLoginHandler_SendOTPInternal(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/send?tenant_id=acme",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing tenant_id returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("client_id present on internal returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?tenant_id=acme&client_id=app",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send?tenant_id=acme",
			map[string]string{"phone": "+1234567890"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTPInternal(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSLoginHandler_VerifyOTPPublic(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/verify?client_id=app",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("tenant_id present on public returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?client_id=app&tenant_id=acme",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?client_id=app",
			map[string]string{"phone": "", "otp": ""})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSMSLoginService{
			verifyOTPFn: func(ctx context.Context, phone, otp string, clientID, tenantID *string) (*LoginResponseDTO, error) {
				return nil, errors.New("invalid otp")
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?client_id=app",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockSMSLoginService{
			verifyOTPFn: func(ctx context.Context, phone, otp string, clientID, tenantID *string) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?client_id=app",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).VerifyOTPPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSLoginHandler_VerifyOTPInternal(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/verify?tenant_id=acme",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing tenant_id returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("client_id present on internal returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?tenant_id=acme&client_id=app",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTPInternal(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockSMSLoginService{
			verifyOTPFn: func(ctx context.Context, phone, otp string, clientID, tenantID *string) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify?tenant_id=acme",
			map[string]string{"phone": "+1234567890", "otp": "123456"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).VerifyOTPInternal(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
