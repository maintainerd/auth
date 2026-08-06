package apperror

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCError(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		code    codes.Code
		message string
	}{
		{"nil", nil, codes.OK, ""},
		{"existing grpc status", status.Error(codes.ResourceExhausted, "slow down"), codes.ResourceExhausted, "slow down"},
		{"not found", NewNotFound("tenant"), codes.NotFound, "tenant not found"},
		{"conflict", NewConflict("already exists"), codes.AlreadyExists, "already exists"},
		{"forbidden", NewForbidden("denied"), codes.PermissionDenied, "denied"},
		{"unauthorized", NewUnauthorized("bad token"), codes.Unauthenticated, "bad token"},
		{"validation", NewValidation("bad request"), codes.InvalidArgument, "bad request"},
		{"too many requests", NewTooManyRequests("slow down"), codes.ResourceExhausted, "slow down"},
		{"internal", NewInternal("database failed", errors.New("boom")), codes.Internal, "database failed"},
		{"unknown", errors.New("boom"), codes.Internal, "internal server error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToGRPCError(tc.err)
			if tc.err == nil {
				assert.NoError(t, got)
				return
			}
			st, ok := status.FromError(got)
			assert.True(t, ok)
			assert.Equal(t, tc.code, st.Code())
			assert.Equal(t, tc.message, st.Message())
		})
	}
}

func TestToGRPCError_RetryInfo(t *testing.T) {
	t.Run("retry hint becomes RetryInfo", func(t *testing.T) {
		st, ok := status.FromError(ToGRPCError(NewTooManyRequestsAfter("slow down", 30*time.Second)))
		assert.True(t, ok)

		var sawRetryInfo bool
		for _, detail := range st.Details() {
			switch d := detail.(type) {
			case *errdetails.ErrorInfo:
				assert.Equal(t, "TOO_MANY_REQUESTS", d.Reason)
			case *errdetails.RetryInfo:
				sawRetryInfo = true
				assert.Equal(t, 30*time.Second, d.RetryDelay.AsDuration())
			}
		}
		assert.True(t, sawRetryInfo)
	})

	t.Run("no hint means no RetryInfo", func(t *testing.T) {
		st, ok := status.FromError(ToGRPCError(NewTooManyRequests("slow down")))
		assert.True(t, ok)
		for _, detail := range st.Details() {
			_, isRetryInfo := detail.(*errdetails.RetryInfo)
			assert.False(t, isRetryInfo)
		}
	})
}

func TestToGRPCError_ValidationDetails(t *testing.T) {
	st, ok := status.FromError(ToGRPCError(NewValidation("name is required")))
	assert.True(t, ok)

	var sawErrorInfo bool
	var sawBadRequest bool
	for _, detail := range st.Details() {
		switch d := detail.(type) {
		case *errdetails.ErrorInfo:
			sawErrorInfo = true
			assert.Equal(t, "VALIDATION", d.Reason)
			assert.Equal(t, "maintainerd.auth", d.Domain)
		case *errdetails.BadRequest:
			sawBadRequest = true
			assert.Len(t, d.FieldViolations, 1)
			assert.Equal(t, "request", d.FieldViolations[0].Field)
		}
	}
	assert.True(t, sawErrorInfo)
	assert.True(t, sawBadRequest)
}
