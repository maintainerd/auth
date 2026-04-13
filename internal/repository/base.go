package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// allowedSortColumns is the set of column names that are safe to interpolate
// directly into an ORDER BY clause. Add new columns as needed.
var allowedSortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "status": {},
	"email": {}, "username": {}, "identifier": {}, "title": {},
	"first_name": {}, "last_name": {}, "phone": {}, "city": {},
	"country": {}, "is_default": {}, "event_type": {}, "tenant_id": {},
	"is_system": {}, "is_active": {}, "type": {}, "version": {},
	"priority": {}, "provider_name": {}, "client_id": {},
	"category": {}, "severity": {}, "result": {}, "error_reason": {},
}

// normalizePagination clamps page and limit to safe positive values.
// page defaults to 1 and limit defaults to 20 when zero or negative.
func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return page, limit
}

// sanitizeOrder validates sortBy against the allowlist and returns a safe
// ORDER BY expression. Falls back to defaultCol (e.g. "created_at DESC") if
// sortBy is empty or not in the allowlist.
func sanitizeOrder(sortBy, sortOrder, defaultCol string) string {
	col := strings.ToLower(strings.TrimSpace(sortBy))
	if col == "" {
		return defaultCol
	}
	if _, ok := allowedSortColumns[col]; !ok {
		return defaultCol
	}
	order := "ASC"
	if strings.ToLower(strings.TrimSpace(sortOrder)) == "desc" {
		order = "DESC"
	}
	return col + " " + order
}

// sanitizeOrderPrefixed is like sanitizeOrder but prepends a table prefix
// (e.g. "users.") to the validated column name. Use this when queries JOIN
// multiple tables and the ORDER BY must be unambiguous.
func sanitizeOrderPrefixed(prefix, sortBy, sortOrder, defaultCol string) string {
	col := strings.ToLower(strings.TrimSpace(sortBy))
	if col == "" {
		return defaultCol
	}
	if _, ok := allowedSortColumns[col]; !ok {
		return defaultCol
	}
	order := "ASC"
	if strings.ToLower(strings.TrimSpace(sortOrder)) == "desc" {
		order = "DESC"
	}
	return prefix + col + " " + order
}

// BaseRepository provides common CRUD operations for GORM-backed entities.
type BaseRepository[T any] struct {
	db            *gorm.DB
	uuidFieldName string // e.g., "role_uuid", "user_uuid"
	idFieldName   string // e.g., "role_id", "user_id"
}

// NewBaseRepository creates a base repository bound to the given *gorm.DB.
func NewBaseRepository[T any](db *gorm.DB, uuidFieldName, idFieldName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:            db,
		uuidFieldName: uuidFieldName,
		idFieldName:   idFieldName,
	}
}

// WithTx creates a new repo bound to the given transaction, carrying the same
// field-name configuration. Concrete repos should call r.BaseRepository.WithTx(tx)
// instead of re-constructing with hardcoded field names.
func (r *BaseRepository[T]) WithTx(tx *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:            tx,
		uuidFieldName: r.uuidFieldName,
		idFieldName:   r.idFieldName,
	}
}

// DB returns the underlying *gorm.DB so concrete repos that embed
// BaseRepository can access it without keeping their own redundant copy.
func (r *BaseRepository[T]) DB() *gorm.DB {
	return r.db
}

// Create a new record and return it with populated fields (e.g., auto-generated ID)
func (r *BaseRepository[T]) Create(entity *T) (*T, error) {
	if err := r.db.Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// Create Or Update
func (r *BaseRepository[T]) CreateOrUpdate(entity *T) (*T, error) {
	if err := r.db.Save(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// FindAll with optional preloads
func (r *BaseRepository[T]) FindAll(preloads ...string) ([]T, error) {
	var entities []T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindByUUID with optional preloads. Returns nil, nil when the record does not exist.
func (r *BaseRepository[T]) FindByUUID(uuid any, preloads ...string) (*T, error) {
	var entity T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.uuidFieldName+" = ?", uuid).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// FindByUUIDs with optional preloads.
func (r *BaseRepository[T]) FindByUUIDs(uuids []string, preloads ...string) ([]T, error) {
	var entities []T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.uuidFieldName+" IN ?", uuids).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindByID with optional preloads. Returns nil, nil when the record does not exist.
func (r *BaseRepository[T]) FindByID(id any, preloads ...string) (*T, error) {
	var entity T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.idFieldName+" = ?", id).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// UpdateByUUID and return the updated entity (atomic: update + re-fetch in one transaction).
func (r *BaseRepository[T]) UpdateByUUID(uuid any, updatedData any) (*T, error) {
	var result *T
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(new(T)).Where(r.uuidFieldName+" = ?", uuid).Updates(updatedData).Error; err != nil {
			return err
		}
		var entity T
		if err := tx.Where(r.uuidFieldName+" = ?", uuid).First(&entity).Error; err != nil {
			return err
		}
		result = &entity
		return nil
	})
	return result, err
}

// UpdateByID and return the updated entity (atomic: update + re-fetch in one transaction).
func (r *BaseRepository[T]) UpdateByID(id any, updatedData any) (*T, error) {
	var result *T
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(new(T)).Where(r.idFieldName+" = ?", id).Updates(updatedData).Error; err != nil {
			return err
		}
		var entity T
		if err := tx.Where(r.idFieldName+" = ?", id).First(&entity).Error; err != nil {
			return err
		}
		result = &entity
		return nil
	})
	return result, err
}

// DeleteByUUID deletes the record matching the given UUID.
// Returns gorm.ErrRecordNotFound when no row matches.
func (r *BaseRepository[T]) DeleteByUUID(uuid any) error {
	result := r.db.Where(r.uuidFieldName+" = ?", uuid).Delete(new(T))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteByID deletes the record matching the given ID.
// Returns gorm.ErrRecordNotFound when no row matches.
func (r *BaseRepository[T]) DeleteByID(id any) error {
	result := r.db.Where(r.idFieldName+" = ?", id).Delete(new(T))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Paginate with optional preloads
func (r *BaseRepository[T]) Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[T], error) {
	page, limit = normalizePagination(page, limit)

	var entities []T
	var total int64

	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if len(conditions) > 0 {
		query = query.Where(conditions)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	if err := query.Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[T]{
		Data:       entities,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
