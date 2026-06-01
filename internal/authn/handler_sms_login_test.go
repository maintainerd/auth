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
	sendOTPFn   func(ctx context.Context, req SMSLoginSendDTO) error
	verifyOTPFn func(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error)
}

func (m *mockSMSLoginService) SendOTP(ctx context.Context, req SMSLoginSendDTO) error {
	if m.sendOTPFn != nil {
		return m.sendOTPFn(ctx, req)
	}
	return nil
}

func (m *mockSMSLoginService) VerifyOTP(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error) {
	if m.verifyOTPFn != nil {
		return m.verifyOTPFn(ctx, req)
	}
	return nil, nil
}

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

func TestSMSLoginHandler_SendOTP(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/send",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send",
			map[string]string{"phone": ""})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSMSLoginService{
			sendOTPFn: func(ctx context.Context, req SMSLoginSendDTO) error {
				return errors.New("db error")
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send",
			map[string]string{"phone": "+1234567890", "client_id": "app", "provider_id": "idp"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).SendOTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/send",
			map[string]string{"phone": "+1234567890", "client_id": "app", "provider_id": "idp"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).SendOTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSMSLoginHandler_VerifyOTP(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/sms-login/verify",
			bytes.NewBufferString(`{bad`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify",
			map[string]string{"phone": "", "otp": ""})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(&mockSMSLoginService{}).VerifyOTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSMSLoginService{
			verifyOTPFn: func(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error) {
				return nil, errors.New("invalid otp")
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify",
			map[string]string{"phone": "+1234567890", "otp": "123456", "client_id": "app", "provider_id": "idp"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).VerifyOTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockSMSLoginService{
			verifyOTPFn: func(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error) {
				return &LoginResponseDTO{AccessToken: "at"}, nil
			},
		}
		r := smsJSONReq(t, http.MethodPost, "/sms-login/verify",
			map[string]string{"phone": "+1234567890", "otp": "123456", "client_id": "app", "provider_id": "idp"})
		w := httptest.NewRecorder()
		NewSMSLoginHandler(svc).VerifyOTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
