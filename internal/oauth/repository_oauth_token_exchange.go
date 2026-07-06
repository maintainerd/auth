package oauth

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type oauthTokenExchangeRepository struct {
	*BaseRepository[OAuthTokenExchange]
}

func NewOAuthTokenExchangeRepository(db *gorm.DB) OAuthTokenExchangeRepository {
	return &oauthTokenExchangeRepository{
		BaseRepository: database.NewBaseRepository[OAuthTokenExchange](db, "oauth_token_exchange_uuid", "oauth_token_exchange_id"),
	}
}

func (r *oauthTokenExchangeRepository) Record(exchange *OAuthTokenExchange) error {
	return r.DB().Create(exchange).Error
}
