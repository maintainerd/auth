package authn

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The reset-email send is dispatched off the request path in production. Run it
// inline for the rest of the suite so the existing assertions on email.SendEmail
// stay deterministic (a goroutine that outlives the test would race the
// assertion, or worse, call t.Errorf after the test finished). The timing tests
// below restore the real dispatcher for the cases that are actually about it.
func init() {
	forgotPasswordDispatch = func(fn func()) { fn() }
}

func withRealForgotPasswordDispatch(t *testing.T) {
	t.Helper()
	orig := forgotPasswordDispatch
	t.Cleanup(func() { forgotPasswordDispatch = orig })
	forgotPasswordDispatch = func(fn func()) { go fn() }
}

func withForgotPasswordFloor(t *testing.T, d time.Duration) {
	t.Helper()
	orig := forgotPasswordMinDuration
	t.Cleanup(func() { forgotPasswordMinDuration = orig })
	forgotPasswordMinDuration = d
}

// A generic response body is worthless if the response TIME still says whether
// the address exists. An unknown address used to return the moment the lookup
// missed; a known one first revoked tokens, inserted a row, rendered a template
// and blocked on SMTP.
func TestForgotPasswordService_SendPasswordResetEmail_TimingDoesNotRevealExistence(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")

	origAppPublicHostname := config.AppPublicHostname
	origConsoleHostname := config.AppFrontendConsoleHostname
	t.Cleanup(func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AppFrontendConsoleHostname = origConsoleHostname
	})
	config.AppPublicHostname = "https://api.example.com"
	config.AppFrontendConsoleHostname = "https://auth.example.com"

	withRealForgotPasswordDispatch(t)
	withForgotPasswordFloor(t, 200*time.Millisecond)

	// A deliberately slow SMTP round trip: the exact term that used to dominate
	// the known-address path.
	const smtpDelay = 2 * time.Second
	var wg sync.WaitGroup
	origSendEmail := email.SendEmail
	t.Cleanup(func() { email.SendEmail = origSendEmail })
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error {
		defer wg.Done()
		time.Sleep(smtpDelay)
		return nil
	}

	newService := func(t *testing.T, user *User) ForgotPasswordService {
		t.Helper()
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		return NewForgotPasswordService(
			gormDB,
			&mockUserRepo{findByEmailFn: func(string) (*User, error) { return user, nil }},
			&mockUserTokenRepo{findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) { return nil, nil }},
			&mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }},
			&mockEmailTemplateRepo{findByNameFn: func(string) (*branding.EmailTemplate, error) {
				return &branding.EmailTemplate{Subject: "Reset", BodyHTML: `<a href="{{.ResetURL}}">R</a>`}, nil
			}},
		)
	}

	missStart := time.Now()
	_, err := newService(t, nil).SendPasswordResetEmail(context.Background(), "nobody@example.com", nil, nil, true)
	require.NoError(t, err)
	missElapsed := time.Since(missStart)

	wg.Add(1)
	hit := &User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: shared.StatusActive}
	hitStart := time.Now()
	_, err = newService(t, hit).SendPasswordResetEmail(context.Background(), "user@example.com", nil, nil, true)
	require.NoError(t, err)
	hitElapsed := time.Since(hitStart)
	wg.Wait() // let the detached send finish before the SendEmail stub is restored

	assert.Less(t, hitElapsed, smtpDelay,
		"the known-address path must not block on SMTP — that delay is the enumeration oracle")
	assert.GreaterOrEqual(t, missElapsed, forgotPasswordMinDuration,
		"the unknown-address path must be padded to the same floor, not returned early")

	skew := hitElapsed - missElapsed
	if skew < 0 {
		skew = -skew
	}
	assert.Less(t, skew, 150*time.Millisecond,
		"the two paths must not be separable by response time (miss=%v hit=%v)", missElapsed, hitElapsed)
}

// Delivery failures are logged, never surfaced — moving the send off the request
// path must not change that.
func TestForgotPasswordService_SendPasswordResetEmail_AsyncSendFailureStillSucceeds(t *testing.T) {
	_ = os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-hmac")

	origAppPublicHostname := config.AppPublicHostname
	origConsoleHostname := config.AppFrontendConsoleHostname
	t.Cleanup(func() {
		config.AppPublicHostname = origAppPublicHostname
		config.AppFrontendConsoleHostname = origConsoleHostname
	})
	config.AppPublicHostname = "https://api.example.com"
	config.AppFrontendConsoleHostname = "https://auth.example.com"

	withForgotPasswordFloor(t, 0)

	origSendEmail := email.SendEmail
	t.Cleanup(func() { email.SendEmail = origSendEmail })
	email.SendEmail = func(context.Context, *gorm.DB, email.SendEmailParams) error {
		return errors.New("smtp failure")
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewForgotPasswordService(
		gormDB,
		&mockUserRepo{findByEmailFn: func(string) (*User, error) {
			return &User{UserID: 1, UserUUID: uuid.New(), Email: "user@example.com", Status: shared.StatusActive}, nil
		}},
		&mockUserTokenRepo{findByUserIDAndTokenTypeFn: func(int64, string) ([]UserToken, error) { return nil, nil }},
		&mockClientRepo{findSystemFn: func() (*Client, error) { return buildActiveClient(), nil }},
		&mockEmailTemplateRepo{findByNameFn: func(string) (*branding.EmailTemplate, error) {
			return &branding.EmailTemplate{Subject: "Reset", BodyHTML: `<a href="{{.ResetURL}}">R</a>`}, nil
		}},
	)

	resp, err := svc.SendPasswordResetEmail(context.Background(), "user@example.com", nil, nil, true)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}
