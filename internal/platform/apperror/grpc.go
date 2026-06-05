package apperror

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const grpcErrorDomain = "maintainerd.auth"

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	code, message, reason := classifyGRPCError(err)
	st := status.New(code, message)
	stWithDetails, detailsErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: grpcErrorDomain,
	})
	if detailsErr == nil {
		st = stWithDetails
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		if detailed, detailsErr := st.WithDetails(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "request", Description: validationErr.Error()},
			},
		}); detailsErr == nil {
			st = detailed
		}
	}

	return st.Err()
}

func classifyGRPCError(err error) (codes.Code, string, string) {
	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return codes.NotFound, notFoundErr.Error(), "NOT_FOUND"
	}
	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) {
		return codes.AlreadyExists, conflictErr.Error(), "CONFLICT"
	}
	var forbiddenErr *ForbiddenError
	if errors.As(err, &forbiddenErr) {
		return codes.PermissionDenied, forbiddenErr.Error(), "FORBIDDEN"
	}
	var unauthorizedErr *UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return codes.Unauthenticated, unauthorizedErr.Error(), "UNAUTHORIZED"
	}
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return codes.InvalidArgument, validationErr.Error(), "VALIDATION"
	}
	var internalErr *InternalError
	if errors.As(err, &internalErr) {
		return codes.Internal, internalErr.Reason, "INTERNAL"
	}
	return codes.Internal, "internal server error", "INTERNAL"
}
