package repository

import (
	"gorm.io/gorm"
)

type BaseRepository[T any] struct {
	db            *gorm.DB
	UUIDFieldName string // e.g., "role_uuid", "user_uuid"
	IDFieldName   string // e.g., "role_id", "user_id"
}

// Default db connection
func NewBaseRepository[T any](db *gorm.DB, uuidFieldName, idFieldName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:            db,
		UUIDFieldName: uuidFieldName,
		IDFieldName:   idFieldName,
	}
}

// WithTx creates a new repo bound to the given transaction
func (r *BaseRepository[T]) WithTx(tx *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db:            tx,
		UUIDFieldName: r.UUIDFieldName,
		IDFieldName:   r.IDFieldName,
	}
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

// FindByUUID with optional preloads
func (r *BaseRepository[T]) FindByUUID(uuid any, preloads ...string) (*T, error) {
	var entity T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.UUIDFieldName+" = ?", uuid).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// FindByUUIDs with optional preloads
func (r *BaseRepository[T]) FindByUUIDs(uuids []string, preloads ...string) ([]T, error) {
	var entities []T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.UUIDFieldName+" IN ?", uuids).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

// FindByID with optional preloads
func (r *BaseRepository[T]) FindByID(id any, preloads ...string) (*T, error) {
	var entity T
	query := r.db.Model(new(T))
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	if err := query.Where(r.IDFieldName+" = ?", id).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// UpdateByUUID and return the updated entity
func (r *BaseRepository[T]) UpdateByUUID(uuid any, updatedData any) (*T, error) {
	if err := r.db.Model(new(T)).Where(r.UUIDFieldName+" = ?", uuid).Updates(updatedData).Error; err != nil {
		return nil, err
	}
	return r.FindByUUID(uuid)
}

// UpdateByID and return the updated entity
func (r *BaseRepository[T]) UpdateByID(id any, updatedData any) (*T, error) {
	if err := r.db.Model(new(T)).Where(r.IDFieldName+" = ?", id).Updates(updatedData).Error; err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

// DeleteByUUID and return the deleted entity
func (r *BaseRepository[T]) DeleteByUUID(uuid any) error {
	return r.db.Where(r.UUIDFieldName+" = ?", uuid).Delete(new(T)).Error
}

// DeleteByID and return the deleted entity
func (r *BaseRepository[T]) DeleteByID(id any) error {
	return r.db.Where(r.IDFieldName+" = ?", id).Delete(new(T)).Error
}

// PaginationResult holds paginated data and meta
type PaginationResult[T any] struct {
	Data       []T
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// Paginate with optional preloads
func (r *BaseRepository[T]) Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[T], error) {
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
