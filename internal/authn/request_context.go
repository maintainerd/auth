package authn

import (
	"context"
	"strings"
)

type requestContextKey string

const (
	registrationCaptchaTokenKey requestContextKey = "registration_captcha_token"
	loginTrustedDeviceTokenKey  requestContextKey = "login_trusted_device_token"
	loginRememberDeviceKey      requestContextKey = "login_remember_device"
)

func contextWithRegistrationCaptchaToken(ctx context.Context, token string) context.Context {
	if strings.TrimSpace(token) == "" {
		return ctx
	}
	return context.WithValue(ctx, registrationCaptchaTokenKey, strings.TrimSpace(token))
}

func registrationCaptchaTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if token, ok := ctx.Value(registrationCaptchaTokenKey).(string); ok {
		return token
	}
	return ""
}

func contextWithTrustedDeviceToken(ctx context.Context, token string) context.Context {
	if strings.TrimSpace(token) == "" {
		return ctx
	}
	return context.WithValue(ctx, loginTrustedDeviceTokenKey, strings.TrimSpace(token))
}

func trustedDeviceTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if token, ok := ctx.Value(loginTrustedDeviceTokenKey).(string); ok {
		return token
	}
	return ""
}

func contextWithRememberDevice(ctx context.Context, remember bool) context.Context {
	if !remember {
		return ctx
	}
	return context.WithValue(ctx, loginRememberDeviceKey, true)
}

func rememberDeviceFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	remember, _ := ctx.Value(loginRememberDeviceKey).(bool)
	return remember
}
