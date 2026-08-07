package user

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProfilePictureRepository is the only place avatar bytes are read or written.
//
// Narrow on purpose: every method here names the columns it needs, so no caller
// can accidentally pull a 2 MiB image into a query that only wanted to know
// whether one exists.
type ProfilePictureRepository interface {
	// Upsert replaces the profile's avatar, or stores its first.
	Upsert(record *ProfilePictureRecord) error
	// FindByProfileID returns the full picture, bytes included. Only the serve
	// endpoint should call this.
	FindByProfileID(profileID int64) (*ProfilePictureRecord, error)
	// FindMetaByProfileID returns everything EXCEPT the bytes.
	//
	// This is what a conditional request needs: an ETag comparison that matched
	// would otherwise have read the whole image only to discard it and return
	// 304.
	FindMetaByProfileID(profileID int64) (*ProfilePictureRecord, error)
	// ExistsForProfileIDs reports which of the given profiles have an avatar.
	//
	// Batched because the alternative — asking per profile while rendering a
	// list — is a query per row. Returns a set rather than a slice so callers
	// can look up by id without scanning.
	ExistsForProfileIDs(profileIDs []int64) (map[int64]bool, error)
	Delete(profileID int64) error
	WithTx(tx *gorm.DB) ProfilePictureRepository
}

type profilePictureRepository struct {
	db *gorm.DB
}

func NewProfilePictureRepository(db *gorm.DB) ProfilePictureRepository {
	return &profilePictureRepository{db: db}
}

func (r *profilePictureRepository) WithTx(tx *gorm.DB) ProfilePictureRepository {
	return &profilePictureRepository{db: tx}
}

func (r *profilePictureRepository) Upsert(record *ProfilePictureRecord) error {
	// One row per profile: replacing an avatar overwrites rather than appending,
	// so an account cannot accumulate images by re-uploading.
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "profile_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "content_type", "etag", "updated_at"}),
	}).Create(record).Error
}

func (r *profilePictureRepository) FindByProfileID(profileID int64) (*ProfilePictureRecord, error) {
	var record ProfilePictureRecord
	err := r.db.Where("profile_id = ?", profileID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *profilePictureRepository) FindMetaByProfileID(profileID int64) (*ProfilePictureRecord, error) {
	var record ProfilePictureRecord
	err := r.db.
		Select("profile_picture_id", "profile_id", "content_type", "etag", "created_at", "updated_at").
		Where("profile_id = ?", profileID).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *profilePictureRepository) ExistsForProfileIDs(profileIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(profileIDs))
	if len(profileIDs) == 0 {
		return out, nil
	}
	var ids []int64
	// Selecting only the id keeps this off the bytes entirely.
	if err := r.db.Model(&ProfilePictureRecord{}).
		Where("profile_id IN ?", profileIDs).
		Pluck("profile_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (r *profilePictureRepository) Delete(profileID int64) error {
	return r.db.Where("profile_id = ?", profileID).Delete(&ProfilePictureRecord{}).Error
}
