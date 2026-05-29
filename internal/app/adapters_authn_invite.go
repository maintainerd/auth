package app

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

type authnInviteRepoAdapter struct {
	repo invite.InviteRepository
}

func newAuthnInviteRepoAdapter(repo invite.InviteRepository) authn.InviteRepository {
	return &authnInviteRepoAdapter{repo: repo}
}

func (a *authnInviteRepoAdapter) WithTx(tx *gorm.DB) authn.InviteRepository {
	return &authnInviteRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnInviteRepoAdapter) Create(e *authn.Invite) (*authn.Invite, error) {
	r, err := a.repo.Create(toInviteInvite(e))
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) CreateOrUpdate(e *authn.Invite) (*authn.Invite, error) {
	r, err := a.repo.CreateOrUpdate(toInviteInvite(e))
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) FindAll(p ...string) ([]authn.Invite, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnInvites(r), err
}

func (a *authnInviteRepoAdapter) FindByUUID(id any, p ...string) (*authn.Invite, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.Invite, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnInvites(r), err
}

func (a *authnInviteRepoAdapter) FindByID(id any, p ...string) (*authn.Invite, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) UpdateByUUID(id, data any) (*authn.Invite, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) UpdateByID(id, data any) (*authn.Invite, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnInviteRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnInviteRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.Invite], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Invite]{Data: mapAuthnInvites(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnInviteRepoAdapter) FindByToken(token string) (*authn.Invite, error) {
	r, err := a.repo.FindByToken(token)
	return toAuthnInvite(r), err
}

func (a *authnInviteRepoAdapter) MarkAsUsed(inviteUUID uuid.UUID) error {
	return a.repo.MarkAsUsed(inviteUUID)
}

type authnPasswordHistoryRepoAdapter struct {
	repo user.UserPasswordHistoryRepository
}

func newAuthnPasswordHistoryRepoAdapter(repo user.UserPasswordHistoryRepository) authn.UserPasswordHistoryRepository {
	return &authnPasswordHistoryRepoAdapter{repo: repo}
}

func (a *authnPasswordHistoryRepoAdapter) WithTx(tx *gorm.DB) authn.UserPasswordHistoryRepository {
	return &authnPasswordHistoryRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnPasswordHistoryRepoAdapter) AddEntry(userID int64, hash string) error {
	return a.repo.AddEntry(userID, hash)
}

func (a *authnPasswordHistoryRepoAdapter) FindRecentHashes(userID int64, count int) ([]string, error) {
	return a.repo.FindRecentHashes(userID, count)
}

func (a *authnPasswordHistoryRepoAdapter) PruneExcess(userID int64, keepCount int) error {
	return a.repo.PruneExcess(userID, keepCount)
}
