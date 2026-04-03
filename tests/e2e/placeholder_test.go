//go:build e2e

// Package e2e contains end-to-end tests for the auth service.
//
// These tests require the full HTTP server to be running.
// Start the stack with:
//
//	docker-compose up -d
//
// Run with:
//
//	go test ./tests/e2e/... -tags e2e
//
// E2E tests send real HTTP requests against the running server
// (e.g. localhost:8080/login) and assert on response codes and
// body shapes — exactly what a client or another service would do.
//
// # Layout
//
// Each file in this package targets a user-facing flow:
//
//	login_test.go             → POST /login happy-path + error cases
//	register_test.go          → POST /register happy-path + duplicate email
//	forgot_password_test.go   → POST /forgot-password flow
//	reset_password_test.go    → POST /reset-password flow
//	...
package e2e_test

import "testing"

// TestPlaceholder is a sentinel test that confirms the e2e build tag
// and package declaration are wired correctly.
// Replace or delete this once real e2e tests are added.
func TestPlaceholder(t *testing.T) {
	t.Skip("placeholder — add real e2e tests in this package")
}

