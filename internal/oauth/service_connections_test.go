package oauth

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthConnectionsService_ListConnections(t *testing.T) {
	ctx := context.Background()

	t.Run("client lookup error", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) { return nil, errors.New("db error") }})
		_, err := svc.ListConnections(ctx, "my-client")
		require.Error(t, err)
	})

	t.Run("unknown client", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) { return nil, nil }})
		_, err := svc.ListConnections(ctx, "my-client")
		require.Error(t, err)
	})

	t.Run("non-console system client is rejected", func(t *testing.T) {
		db, _ := newMockDB(t)
		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return &Client{ClientID: 1, Name: shared.SystemClientNameAuthIdentity, Status: shared.StatusActive, IsSystem: true}, nil
		}})
		_, err := svc.ListConnections(ctx, "my-client")
		require.Error(t, err)
	})

	t.Run("auth-console system client enables password", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*enabled = \$2`).
			WithArgs(int64(10), true).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id"}))

		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return &Client{ClientID: 10, Name: shared.SystemClientNameAuthConsole, Status: shared.StatusActive, IsSystem: true}, nil
		}})

		result, err := svc.ListConnections(ctx, "my-client")
		require.NoError(t, err)
		assert.True(t, result.PasswordEnabled)
		assert.Empty(t, result.Connections)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("connection query error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers"`).WillReturnError(errors.New("db error"))
		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return &Client{ClientID: 10, Status: shared.StatusActive}, nil
		}})
		_, err := svc.ListConnections(ctx, "my-client")
		require.Error(t, err)
	})

	t.Run("system enables password, social becomes button, inactive idp skipped", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE.*client_id = \$1.*enabled = \$2`).
			WithArgs(int64(10), true).
			WillReturnRows(sqlmock.NewRows([]string{
				"client_identity_provider_id", "client_identity_provider_uuid", "client_id",
				"identity_provider_id", "is_default", "enabled", "display_order",
			}).
				AddRow(1, uuid.New(), 10, 100, true, true, 0).
				AddRow(2, uuid.New(), 10, 101, false, true, 1).
				AddRow(3, uuid.New(), 10, 102, false, true, 2))
		mock.ExpectQuery(`SELECT \* FROM "identity_providers"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"identity_provider_id", "identity_provider_uuid", "tenant_id", "name", "display_name",
				"provider", "provider_type", "identifier", "status", "is_default", "is_system",
			}).
				AddRow(100, uuid.New(), 1, "maintainerd", "Built-in", "maintainerd", shared.IDPTypeSystem, "sys-idp", shared.StatusActive, true, true).
				AddRow(101, uuid.New(), 1, "google", "Google", "google", "social", "google-idp", shared.StatusActive, false, false).
				AddRow(102, uuid.New(), 1, "facebook", "Facebook", "facebook", "social", "fb-idp", shared.StatusInactive, false, false))

		svc := NewOAuthConnectionsService(db, &mockClientRepo{findByIdentifierFn: func(string) (*Client, error) {
			return &Client{ClientID: 10, Status: shared.StatusActive}, nil
		}})

		result, err := svc.ListConnections(ctx, "my-client")
		require.NoError(t, err)
		assert.True(t, result.PasswordEnabled)
		require.Len(t, result.Connections, 1)
		assert.Equal(t, "google-idp", result.Connections[0].Identifier)
		assert.Equal(t, "Google", result.Connections[0].DisplayName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
