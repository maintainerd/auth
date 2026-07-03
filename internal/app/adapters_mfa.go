package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type mfaUserRepo struct {
	*database.BaseRepository[mfa.User]
}

func newMFAUserRepo(db *gorm.DB) mfa.UserRepository {
	return &mfaUserRepo{database.NewBaseRepository[mfa.User](db, "user_uuid", "user_id")}
}

func (r *mfaUserRepo) WithTx(tx *gorm.DB) mfa.UserRepository {
	return &mfaUserRepo{r.BaseRepository.WithTx(tx)}
}
