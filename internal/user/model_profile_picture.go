package user

import "time"

// ProfilePictureRecord is an uploaded avatar.
//
// Deliberately its own table rather than columns on Profile: profiles is read
// on every profile fetch, list and Preload, and an ORM's default `SELECT *`
// would carry up to 2 MiB of image into all of them. Here the bytes are touched
// by exactly one endpoint.
type ProfilePictureRecord struct {
	ProfilePictureID int64     `gorm:"column:profile_picture_id;primaryKey"`
	ProfileID        int64     `gorm:"column:profile_id;not null;uniqueIndex"`
	Data             []byte    `gorm:"column:data;not null"`
	ContentType      string    `gorm:"column:content_type;not null"`
	ETag             string    `gorm:"column:etag;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ProfilePictureRecord) TableName() string {
	return "profile_pictures"
}

// ProfilePicture is a decoded, validated avatar ready to store or serve.
type ProfilePicture struct {
	Data        []byte
	ContentType string
	ETag        string
}
