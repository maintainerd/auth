package secpolicy

import (
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRestrictionRuleRepository_FindByTenantID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE tenant_id = \$1 AND "ip_restriction_rules"\."deleted_at" IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"ip_restriction_rule_id", "ip_restriction_rule_uuid", "tenant_id", "type", "ip_address", "status"}).
				AddRow(1, uuid.New(), 1, "allow", "1.2.3.4", "active"))

		result, err := NewIPRestrictionRuleRepository(db).FindByTenantID(1)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "allow", result[0].Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE tenant_id = \$1`).
			WithArgs(int64(1)).
			WillReturnError(assert.AnError)

		result, err := NewIPRestrictionRuleRepository(db).FindByTenantID(1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIPRestrictionRuleRepository_FindByTenantIDAndStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE .*tenant_id = \$1 AND status = \$2.*AND "ip_restriction_rules"\."deleted_at" IS NULL`).
			WithArgs(int64(1), "active").
			WillReturnRows(sqlmock.NewRows([]string{"ip_restriction_rule_id", "ip_restriction_rule_uuid", "tenant_id", "type", "ip_address", "status"}).
				AddRow(1, uuid.New(), 1, "allow", "1.2.3.4", "active"))

		result, err := NewIPRestrictionRuleRepository(db).FindByTenantIDAndStatus(1, "active")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "active", result[0].Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE .*tenant_id = \$1 AND status = \$2`).
			WithArgs(int64(1), "active").
			WillReturnError(assert.AnError)

		_, err := NewIPRestrictionRuleRepository(db).FindByTenantIDAndStatus(1, "active")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIPRestrictionRuleRepository_FindByTenantIDAndType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE .*tenant_id = \$1 AND type = \$2.*AND "ip_restriction_rules"\."deleted_at" IS NULL`).
			WithArgs(int64(1), "allow").
			WillReturnRows(sqlmock.NewRows([]string{"ip_restriction_rule_id", "ip_restriction_rule_uuid", "tenant_id", "type", "ip_address", "status"}).
				AddRow(1, uuid.New(), 1, "allow", "1.2.3.4", "active"))

		result, err := NewIPRestrictionRuleRepository(db).FindByTenantIDAndType(1, "allow")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "allow", result[0].Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE .*tenant_id = \$1 AND type = \$2`).
			WithArgs(int64(1), "allow").
			WillReturnError(assert.AnError)

		_, err := NewIPRestrictionRuleRepository(db).FindByTenantIDAndType(1, "allow")
		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIPRestrictionRuleRepository_FindPaginated(t *testing.T) {
	t.Run("with TenantID filter", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		tenantID := int64(1)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "ip_restriction_rules" WHERE tenant_id = \$1 AND "ip_restriction_rules"\."deleted_at" IS NULL`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE tenant_id = \$1 AND "ip_restriction_rules"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT \$2`).
			WithArgs(tenantID, 10).
			WillReturnRows(sqlmock.NewRows([]string{"ip_restriction_rule_id", "ip_restriction_rule_uuid", "tenant_id", "type", "ip_address", "status"}).
				AddRow(1, uuid.New(), tenantID, "allow", "1.2.3.4", "active"))

		result, err := NewIPRestrictionRuleRepository(db).FindPaginated(IPRestrictionRuleRepositoryGetFilter{
			TenantID: &tenantID,
			Page:     1,
			Limit:    10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, tenantID, result.Data[0].TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with all filters", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		tenantID := int64(1)
		ruleType := "allow"
		ipAddr := "1.2.3.4"
		desc := "test rule"
		createdBy := int64(10)
		updatedBy := int64(20)

		mock.ExpectQuery(`SELECT count\(\*\) FROM "ip_restriction_rules" WHERE tenant_id = \$1 AND type = \$2 AND status IN \(\$3\) AND LOWER\(ip_address\) LIKE \$4 AND LOWER\(description\) LIKE \$5 AND created_by = \$6 AND updated_by = \$7 AND "ip_restriction_rules"\."deleted_at" IS NULL`).
			WithArgs(tenantID, ruleType, "active", `%`+strings.ToLower(ipAddr)+`%`, `%`+strings.ToLower(desc)+`%`, createdBy, updatedBy).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "ip_restriction_rules" WHERE tenant_id = \$1 AND type = \$2 AND status IN \(\$3\) AND LOWER\(ip_address\) LIKE \$4 AND LOWER\(description\) LIKE \$5 AND created_by = \$6 AND updated_by = \$7 AND "ip_restriction_rules"\."deleted_at" IS NULL ORDER BY created_at DESC LIMIT \$8`).
			WithArgs(tenantID, ruleType, "active", `%`+strings.ToLower(ipAddr)+`%`, `%`+strings.ToLower(desc)+`%`, createdBy, updatedBy, 10).
			WillReturnRows(sqlmock.NewRows([]string{"ip_restriction_rule_id", "ip_restriction_rule_uuid", "tenant_id", "type", "ip_address", "status"}).
				AddRow(1, uuid.New(), tenantID, ruleType, ipAddr, "active"))

		result, err := NewIPRestrictionRuleRepository(db).FindPaginated(IPRestrictionRuleRepositoryGetFilter{
			TenantID:    &tenantID,
			Type:        &ruleType,
			Status:      []string{"active"},
			IPAddress:   &ipAddr,
			Description: &desc,
			CreatedBy:   &createdBy,
			UpdatedBy:   &updatedBy,
			Page:        1,
			Limit:       10,
		})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, ruleType, result.Data[0].Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIPRestrictionRuleRepository_Mutations(t *testing.T) {
	t.Run("WithTx returns tx-bound repository", func(t *testing.T) {
		db, mock := newSecpolicyMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		tx := db.Begin()
		require.NoError(t, tx.Error)

		repo := NewIPRestrictionRuleRepository(db).WithTx(tx)
		assert.NotNil(t, repo)
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NewIPRestrictionRuleRepository returns non-nil", func(t *testing.T) {
		db, _ := newSecpolicyMockGormDB(t)
		repo := NewIPRestrictionRuleRepository(db)
		assert.NotNil(t, repo)
	})
}
