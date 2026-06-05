package setup

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupState_TableName(t *testing.T) {
	assert.Equal(t, "setup_states", (SetupState{}).TableName())
}

func TestSetupStateRepository(t *testing.T) {
	t.Run("find missing returns nil", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "setup_states" WHERE key = $1 ORDER BY "setup_states"."setup_state_id" LIMIT $2`)).
			WithArgs(SetupStateBootstrap, 1).
			WillReturnRows(sqlmock.NewRows([]string{"setup_state_id", "key", "is_complete", "completed_at", "created_at", "updated_at"}))

		got, err := NewSetupStateRepository(db).FindByKey(SetupStateBootstrap)
		require.NoError(t, err)
		assert.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("is complete reads row", func(t *testing.T) {
		now := time.Now()
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "setup_states" WHERE key = $1 ORDER BY "setup_states"."setup_state_id" LIMIT $2`)).
			WithArgs(SetupStateBootstrap, 1).
			WillReturnRows(sqlmock.NewRows([]string{"setup_state_id", "key", "is_complete", "completed_at", "created_at", "updated_at"}).
				AddRow(int64(1), SetupStateBootstrap, true, now, now, now))

		complete, err := NewSetupStateRepository(db).IsComplete(SetupStateBootstrap)
		require.NoError(t, err)
		assert.True(t, complete)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("is complete propagates find error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "setup_states" WHERE key = $1 ORDER BY "setup_states"."setup_state_id" LIMIT $2`)).
			WithArgs(SetupStateBootstrap, 1).
			WillReturnError(assert.AnError)

		complete, err := NewSetupStateRepository(db).IsComplete(SetupStateBootstrap)
		require.Error(t, err)
		assert.False(t, complete)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("mark complete creates missing row", func(t *testing.T) {
		now := time.Now().UTC()
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "setup_states" WHERE key = $1 ORDER BY "setup_states"."setup_state_id" LIMIT $2`)).
			WithArgs(SetupStateBootstrap, 1).
			WillReturnRows(sqlmock.NewRows([]string{"setup_state_id", "key", "is_complete", "completed_at", "created_at", "updated_at"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "setup_states"`)).
			WillReturnRows(sqlmock.NewRows([]string{"setup_state_id"}).AddRow(int64(1)))
		mock.ExpectCommit()

		state, err := NewSetupStateRepository(db).MarkComplete(SetupStateBootstrap, now)
		require.NoError(t, err)
		assert.True(t, state.IsComplete)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with tx preserves repository behavior", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		repo := NewSetupStateRepository(db).WithTx(db)
		assert.NotNil(t, repo)
	})
}

func TestOpenSetupStateRepository(t *testing.T) {
	now := time.Now()
	repo := NewOpenSetupStateRepository()
	assert.NotNil(t, repo.WithTx(nil))

	state, err := repo.FindByKey(SetupStateBootstrap)
	require.NoError(t, err)
	assert.Nil(t, state)

	complete, err := repo.IsComplete(SetupStateBootstrap)
	require.NoError(t, err)
	assert.False(t, complete)

	state, err = repo.MarkComplete(SetupStateBootstrap, now)
	require.NoError(t, err)
	assert.True(t, state.IsComplete)
	assert.Equal(t, SetupStateBootstrap, state.Key)
	assert.Equal(t, now, *state.CompletedAt)
}
