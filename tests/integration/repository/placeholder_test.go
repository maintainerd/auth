//go:build integration

// Package repository contains integration tests for the repository layer.
//
// These tests require a live Postgres and Redis instance.
// Start the stack with:
//
//	docker-compose up -d postgres-db redis-db
//
// Run with:
//
//	go test ./tests/integration/... -tags integration
//
// Integration tests are intentionally excluded from the default `go test ./...`
// run so that unit tests never require infrastructure.
//
// # Layout
//
// Each file in this package mirrors a file in internal/repository/:
//
//	user_test.go       → internal/repository/user.go
//	role_test.go       → internal/repository/role.go
//	client_test.go     → internal/repository/client.go
//	base_test.go       → internal/repository/base.go
//	...
package repository_test

import "testing"

// TestPlaceholder is a sentinel test that confirms the integration build tag
// and package declaration are wired correctly.
// Replace or delete this once real repository integration tests are added.
func TestPlaceholder(t *testing.T) {
	t.Skip("placeholder — add real integration tests in this package")
}
