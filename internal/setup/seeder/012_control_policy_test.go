package seeder

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSeedControlPolicy(t *testing.T) {
	t.Run("existing policy skips create", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "policies" WHERE (name = $1 AND tenant_id = $2 AND version = $3) AND "policies"."deleted_at" IS NULL ORDER BY "policies"."policy_id" LIMIT $4`)).
			WithArgs(SystemControlPolicyName, int64(1), "v1", 1).
			WillReturnRows(policyRows())

		require.NoError(t, SeedControlPolicy(db, 1))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates system control policy", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "policies" WHERE (name = $1 AND tenant_id = $2 AND version = $3) AND "policies"."deleted_at" IS NULL ORDER BY "policies"."policy_id" LIMIT $4`)).
			WithArgs(SystemControlPolicyName, int64(1), "v1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"policy_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "policies"`)).
			WillReturnRows(sqlmock.NewRows([]string{"policy_id"}).AddRow(int64(10)))
		mock.ExpectCommit()

		require.NoError(t, SeedControlPolicy(db, 1))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("check error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "policies" WHERE (name = $1 AND tenant_id = $2 AND version = $3) AND "policies"."deleted_at" IS NULL ORDER BY "policies"."policy_id" LIMIT $4`)).
			WithArgs(SystemControlPolicyName, int64(1), "v1", 1).
			WillReturnError(assert.AnError)

		err := SeedControlPolicy(db, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check control policy")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create error is returned", func(t *testing.T) {
		db, mock := newSeederMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "policies" WHERE (name = $1 AND tenant_id = $2 AND version = $3) AND "policies"."deleted_at" IS NULL ORDER BY "policies"."policy_id" LIMIT $4`)).
			WithArgs(SystemControlPolicyName, int64(1), "v1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"policy_id"}))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "policies"`)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := SeedControlPolicy(db, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to seed control policy")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestControlPolicyDocumentShape(t *testing.T) {
	document := controlPolicyDocument()
	raw, err := json.Marshal(document)
	require.NoError(t, err)

	assert.Equal(t, "v1", document.Version)
	require.Len(t, document.Statement, 1)
	assert.Equal(t, "allow", document.Statement[0].Effect)
	assert.Equal(t, []string{"*"}, document.Statement[0].Resource)
	assert.Contains(t, string(raw), "tenant:*")

	for _, action := range []string{
		"tenant:create",
		"service:policy:assign",
		"permission:read",
		"client:secret:rotate",
		"api_key:update",
		"user:invite",
		"auth_event:read",
		"signup-flow:update",
		"security-setting:update",
		"ip-restriction-rule:create",
		"email-template:update",
		"sms-template:update",
		"login-template:update",
		"branding:update",
		"tenant-setting:update",
		"email-config:update",
		"sms-config:update",
		"webhook-endpoint:create",
	} {
		t.Run("allows "+action, func(t *testing.T) {
			decision := iam.Evaluate([]iam.PolicyDocument{document}, iam.AuthzRequest{
				Action:   action,
				Resource: "auth:management",
			})
			assert.True(t, decision.Allowed)
		})
	}

	for _, action := range []string{
		"public:register",
		"public:login",
		"public:reset-password",
	} {
		t.Run("does not grant "+action, func(t *testing.T) {
			decision := iam.Evaluate([]iam.PolicyDocument{document}, iam.AuthzRequest{
				Action:   action,
				Resource: "auth:public",
			})
			assert.False(t, decision.Allowed)
		})
	}
}

func newSeederMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func policyRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"policy_id", "policy_uuid", "tenant_id", "name", "description", "document", "version", "status", "is_system", "created_at", "updated_at",
	}).AddRow(
		int64(1), "00000000-0000-0000-0000-000000000001", int64(1), SystemControlPolicyName, "desc",
		[]byte(`{"version":"v1","statement":[]}`), "v1", "active", true, now, now,
	)
}
