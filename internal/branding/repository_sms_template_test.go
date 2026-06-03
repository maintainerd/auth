package branding

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSTemplateRepository_FindByUUIDAndTenantID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uuidStr := id.String()

	t.Run("success", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "sms_templates" WHERE .*sms_template_uuid = \$1.*tenant_id = \$2.*AND "sms_templates"\."deleted_at" IS NULL`).
			WithArgs(uuidStr, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"sms_template_id", "sms_template_uuid", "tenant_id", "name", "status", "is_default", "created_at", "updated_at"}).
				AddRow(1, id, int64(1), "OTP", "active", true, now, now))
		repo := NewSMSTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(uuidStr, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, id, result.SMSTemplateUUID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "sms_templates" WHERE .*sms_template_uuid = \$1.*tenant_id = \$2.*AND "sms_templates"\."deleted_at" IS NULL`).
			WithArgs(uuidStr, int64(1), 1).
			WillReturnRows(sqlmock.NewRows([]string{"sms_template_id", "sms_template_uuid", "tenant_id", "name"}))
		repo := NewSMSTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(uuidStr, 1)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		gdb, mock := newMockGormDBRegex(t)
		mock.ExpectQuery(`SELECT \* FROM "sms_templates" WHERE .*sms_template_uuid = \$1.*tenant_id = \$2`).
			WithArgs(uuidStr, int64(1), 1).
			WillReturnError(assert.AnError)
		repo := NewSMSTemplateRepository(gdb)
		result, err := repo.FindByUUIDAndTenantID(uuidStr, 1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewSMSTemplateRepository(t *testing.T) {
	gdb, _ := newMockGormDBRegex(t)
	repo := NewSMSTemplateRepository(gdb)
	assert.NotNil(t, repo)
}

func TestSMSTemplateRepository_FindPaginated(t *testing.T) {
	gdb, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "sms_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT \* FROM "sms_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"sms_template_id", "sms_template_uuid", "name", "status", "created_at", "updated_at"}))

	repo := NewSMSTemplateRepository(gdb)
	result, err := repo.FindPaginated(SMSTemplateRepositoryGetFilter{Page: 1, Limit: 10})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
