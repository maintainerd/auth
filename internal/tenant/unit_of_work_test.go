package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectUnitOfWork_Do(t *testing.T) {
	tenantRepo := &mockTenantRepo{}
	memberRepo := &mockTenantMemberRepo{}
	uow := newDirectUnitOfWork(tenantRepo, memberRepo)

	t.Run("success exposes direct repositories", func(t *testing.T) {
		called := false

		err := uow.Do(context.Background(), func(tx Transaction) error {
			called = true
			assert.Same(t, tenantRepo, tx.TenantRepository())
			assert.Same(t, memberRepo, tx.TenantMemberRepository())
			return tx.DeleteTenantCascade(context.Background(), 1)
		})

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates callback error", func(t *testing.T) {
		expected := errors.New("callback failed")

		err := uow.Do(context.Background(), func(Transaction) error {
			return expected
		})

		require.ErrorIs(t, err, expected)
	})
}

func TestGormUnitOfWork_DoWithoutDB(t *testing.T) {
	tenantRepo := &mockTenantRepo{}
	memberRepo := &mockTenantMemberRepo{}

	t.Run("nil receiver still invokes callback", func(t *testing.T) {
		var uow *gormUnitOfWork
		called := false

		err := uow.Do(context.Background(), func(tx Transaction) error {
			called = true
			assert.Nil(t, tx.TenantRepository())
			assert.Nil(t, tx.TenantMemberRepository())
			return nil
		})

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("nil db uses direct transaction with repositories", func(t *testing.T) {
		uow := NewGormUnitOfWork(nil, tenantRepo, memberRepo, nil)

		err := uow.Do(context.Background(), func(tx Transaction) error {
			assert.Same(t, tenantRepo, tx.TenantRepository())
			assert.Same(t, memberRepo, tx.TenantMemberRepository())
			return nil
		})

		require.NoError(t, err)
	})
}
