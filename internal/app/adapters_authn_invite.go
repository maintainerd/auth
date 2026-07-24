package app

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// authn.RegistrationFlowRoleRepository adapter — wraps idp.RegistrationFlowRoleRepository
// ---------------------------------------------------------------------------

type authnRegistrationFlowRoleRepoAdapter struct {
	repo     idp.RegistrationFlowRoleRepository
	flowRepo idp.RegistrationFlowRepository
	db       *gorm.DB
}

func newAuthnRegistrationFlowRoleRepoAdapter(db *gorm.DB, repo idp.RegistrationFlowRoleRepository, flowRepo idp.RegistrationFlowRepository) authn.RegistrationFlowRoleRepository {
	return &authnRegistrationFlowRoleRepoAdapter{repo: repo, flowRepo: flowRepo, db: db}
}

func (a *authnRegistrationFlowRoleRepoAdapter) WithTx(tx *gorm.DB) authn.RegistrationFlowRoleRepository {
	return &authnRegistrationFlowRoleRepoAdapter{repo: a.repo.WithTx(tx), flowRepo: a.flowRepo.WithTx(tx), db: tx}
}

func (a *authnRegistrationFlowRoleRepoAdapter) FindByID(registrationFlowID int64) (*authn.RegistrationFlow, error) {
	flow, err := a.flowRepo.FindByID(registrationFlowID)
	if err != nil {
		return nil, err
	}
	return toAuthnRegistrationFlow(flow), nil
}

func (a *authnRegistrationFlowRoleRepoAdapter) FindByNameAndClientTenant(name string, clientID, tenantID int64) (*authn.RegistrationFlow, error) {
	flow, err := a.flowRepo.FindByNameAndClientTenant(name, clientID, tenantID)
	if err != nil {
		return nil, err
	}
	return toAuthnRegistrationFlow(flow), nil
}

func toAuthnRegistrationFlow(flow *idp.RegistrationFlow) *authn.RegistrationFlow {
	if flow == nil {
		return nil
	}
	return &authn.RegistrationFlow{
		RegistrationFlowID:   flow.RegistrationFlowID,
		TenantID:             flow.TenantID,
		ClientID:             flow.ClientID,
		Status:               flow.Status,
		VerificationRequired: flow.VerificationRequired,
		RequiredFields:       flow.RequiredFields,
		IsSystem:             flow.IsSystem,
	}
}

// FindGrantableRoleIDsByRegistrationFlowID applies the redeem-time grant cap.
// Tenant, status, system and soft-delete are filtered in SQL; the
// management-plane test reuses shared.IsElevatedPermission so it cannot drift
// from the attach-time check in idp.
func (a *authnRegistrationFlowRoleRepoAdapter) FindGrantableRoleIDsByRegistrationFlowID(registrationFlowID, tenantID int64) ([]int64, error) {
	type roleRow struct {
		RoleID int64
		Name   string
	}
	var rows []roleRow
	if err := a.db.
		Table("registration_flow_roles AS rfr").
		Select("r.role_id AS role_id, r.name AS name").
		Joins("JOIN roles r ON r.role_id = rfr.role_id").
		Where(`rfr.registration_flow_id = ?
		       AND r.tenant_id = ?
		       AND r.is_system = FALSE
		       AND r.status = ?
		       AND r.deleted_at IS NULL`,
			registrationFlowID, tenantID, shared.StatusActive).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		var permNames []string
		if err := a.db.
			Table("role_permissions AS rp").
			Joins("JOIN permissions p ON p.permission_id = rp.permission_id").
			Where("rp.role_id = ? AND p.deleted_at IS NULL", row.RoleID).
			Pluck("p.name", &permNames).Error; err != nil {
			return nil, err
		}
		if elevated := shared.FirstElevatedPermission(permNames); elevated != "" {
			// Refuse silently rather than failing the registration: the user did
			// nothing wrong, and a misconfigured flow must not become a signup
			// outage. The omission is recorded so it is diagnosable.
			slog.Warn("registration flow role skipped: administrative permission",
				"registration_flow_id", registrationFlowID,
				"role_id", row.RoleID,
				"role", row.Name,
				"permission", elevated)
			continue
		}
		ids = append(ids, row.RoleID)
	}
	return ids, nil
}

// ---------------------------------------------------------------------------
// authn.InviteRepository adapter
// ---------------------------------------------------------------------------

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

func (a *authnInviteRepoAdapter) FindByTokenForUpdate(token string) (*authn.Invite, error) {
	r, err := a.repo.FindByTokenForUpdate(token)
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
