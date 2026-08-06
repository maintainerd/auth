package invite

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stubInviteRepo implements only the methods the lifecycle guards touch. The
// embedded interface is nil, so any other call panics rather than silently
// succeeding — these tests must not reach further into the resend pipeline.
type stubInviteRepo struct {
	InviteRepository
	invite *Invite

	revokeCalled bool
	revokeErr    error
	resetCalled  bool
	resetErr     error
	resetToken   string
}

func (s *stubInviteRepo) FindByUUIDAndTenantID(_ uuid.UUID, _ int64, _ ...string) (*Invite, error) {
	return s.invite, nil
}

func (s *stubInviteRepo) RevokeByUUID(uuid.UUID) error {
	s.revokeCalled = true
	return s.revokeErr
}

func (s *stubInviteRepo) ResetForResend(_ uuid.UUID, newToken string, _ time.Time) error {
	s.resetCalled = true
	s.resetToken = newToken
	return s.resetErr
}

// Regression: RevokeInvite never checked the current status and RevokeByUUID
// filtered only on invite_uuid, so an accepted invite could be flipped to
// revoked while keeping its used_at — an audit trail contradicting itself.
func TestInviteService_RevokeInvite_RefusesNonPending(t *testing.T) {
	for _, status := range []string{shared.StatusAccepted, shared.StatusRevoked, statusExpired} {
		t.Run(status, func(t *testing.T) {
			repo := &stubInviteRepo{invite: &Invite{InviteUUID: uuid.New(), Status: status}}
			svc := &inviteService{inviteRepo: repo}

			err := svc.RevokeInvite(context.Background(), repo.invite.InviteUUID, 1)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot be revoked")
			assert.False(t, repo.revokeCalled, "the write must not be attempted")
		})
	}
}

func TestInviteService_RevokeInvite_AllowsPending(t *testing.T) {
	repo := &stubInviteRepo{invite: &Invite{InviteUUID: uuid.New(), Status: shared.StatusPending}}
	svc := &inviteService{inviteRepo: repo}

	require.NoError(t, svc.RevokeInvite(context.Background(), repo.invite.InviteUUID, 1))
	assert.True(t, repo.revokeCalled)
}

// A concurrent acceptance between the read and the write makes the repository's
// status predicate reject the update; the caller sees a conflict, not a 500.
func TestInviteService_RevokeInvite_LosesRaceToAcceptance(t *testing.T) {
	repo := &stubInviteRepo{
		invite:    &Invite{InviteUUID: uuid.New(), Status: shared.StatusPending},
		revokeErr: ErrInviteNotPending,
	}
	svc := &inviteService{inviteRepo: repo}

	err := svc.RevokeInvite(context.Background(), repo.invite.InviteUUID, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "settled before it could be revoked")
}

// Regression: ResendInvite never checked the status, so a revoked or accepted
// invite could be resurrected with a fresh emailed token and its acceptance
// record (used_at) destroyed.
func TestInviteService_ResendInvite_RefusesSettledInvites(t *testing.T) {
	for _, status := range []string{shared.StatusAccepted, shared.StatusRevoked} {
		t.Run(status, func(t *testing.T) {
			repo := &stubInviteRepo{invite: &Invite{InviteUUID: uuid.New(), Status: status}}
			svc := &inviteService{inviteRepo: repo}

			_, err := svc.ResendInvite(context.Background(), repo.invite.InviteUUID, 1)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot be resent")
			assert.False(t, repo.resetCalled, "no new token may be minted")
		})
	}
}

// A pending invite that is merely past its expiry is the reason resend exists,
// and a row already swept to 'expired' is likewise not settled by anyone.
func TestInviteService_ResendInvite_AllowsUnsettledStatuses(t *testing.T) {
	for _, status := range []string{shared.StatusPending, statusExpired} {
		t.Run(status, func(t *testing.T) {
			assert.True(t, isResendable(status))
		})
	}
	assert.False(t, isResendable(shared.StatusAccepted))
	assert.False(t, isResendable(shared.StatusRevoked))
}

func TestInviteService_ResendInvite_NotFound(t *testing.T) {
	repo := &stubInviteRepo{invite: nil}
	svc := &inviteService{inviteRepo: repo}

	_, err := svc.ResendInvite(context.Background(), uuid.New(), 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invite not found")
}

// stubInviteClientRepo answers the one lookup ResendInvite makes after it has
// issued the new token. Returning nil ends the call with a validation error,
// which is exactly far enough to observe what was handed to ResetForResend.
type stubInviteClientRepo struct {
	ClientRepository
}

func (stubInviteClientRepo) FindSystemIdentityByTenantID(int64) (*Client, error) {
	return nil, nil
}

// The service must hand ResetForResend the RAW token and nothing else: the
// repository is the single place a token is converted to its stored digest, so a
// service that pre-hashed would write a digest-of-a-digest and silently make
// every resent invite unredeemable — while a service that kept storing the raw
// value is the bug this whole change removes. Paired with the repository test
// asserting ResetForResend writes hashInviteToken(newToken), this pins the raw
// token to the email and the digest to the database.
func TestInviteService_ResendInvite_HandsRepositoryTheRawToken(t *testing.T) {
	const issued = "issued-raw-token"
	original := crypto.GenerateIdentifier
	crypto.GenerateIdentifier = func(int) (string, error) { return issued, nil }
	t.Cleanup(func() { crypto.GenerateIdentifier = original })

	repo := &stubInviteRepo{invite: &Invite{
		InviteUUID:      uuid.New(),
		TenantID:        7,
		Status:          shared.StatusPending,
		InvitedEmail:    "ada@example.com",
		InviteTokenHash: hashInviteToken("previous-token"),
	}}
	svc := &inviteService{inviteRepo: repo, clientRepo: stubInviteClientRepo{}}

	_, err := svc.ResendInvite(context.Background(), repo.invite.InviteUUID, 7)

	require.Error(t, err, "the stub client repo ends the call after the token is issued")
	require.True(t, repo.resetCalled)
	assert.Equal(t, issued, repo.resetToken)
	assert.NotEqual(t, hashInviteToken(issued), repo.resetToken)
}

// --- SendInvite: what lands in the row vs what lands in the email ------------

type capturingInviteRepo struct {
	InviteRepository
	created *Invite
}

func (c *capturingInviteRepo) WithTx(*gorm.DB) InviteRepository { return c }

func (c *capturingInviteRepo) Create(i *Invite) (*Invite, error) {
	c.created = i
	return i, nil
}

type activeIdentityClientRepo struct {
	ClientRepository
	tenantID int64
}

func (a activeIdentityClientRepo) WithTx(*gorm.DB) ClientRepository { return a }

func (a activeIdentityClientRepo) FindSystemIdentityByTenantID(tenantID int64) (*Client, error) {
	identifier := "identity-client"
	return &Client{ClientID: 5, TenantID: tenantID, Status: shared.StatusActive, Identifier: &identifier}, nil
}

type noopRegistrationFlowRepo struct{ RegistrationFlowRepository }

func (n noopRegistrationFlowRepo) WithTx(*gorm.DB) RegistrationFlowRepository { return n }

// The row must never hold the raw invite token. Any read of the invites table —
// a backup, a replica, a support query, a SQL injection — used to yield live
// account-creation credentials carrying the registration flow's role grants.
// The emailed link is the only place the raw token may appear.
func TestInviteService_SendInvite_PersistsOnlyTheDigest(t *testing.T) {
	const issued = "issued-raw-token"
	original := crypto.GenerateIdentifier
	crypto.GenerateIdentifier = func(int) (string, error) { return issued, nil }
	t.Cleanup(func() { crypto.GenerateIdentifier = original })

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	// The tenant lookup that follows the commit is where this test stops; the
	// invite row has already been built and handed to Create by then.
	mock.ExpectQuery(`SELECT "name","is_system" FROM "invites"`).WillReturnError(assert.AnError)

	repo := &capturingInviteRepo{}
	svc := &inviteService{
		db:                   db,
		inviteRepo:           repo,
		clientRepo:           activeIdentityClientRepo{tenantID: 7},
		registrationFlowRepo: noopRegistrationFlowRepo{},
	}

	_, err := svc.SendInvite(context.Background(), 7, "ada@example.com", 2, nil, nil)

	require.Error(t, err, "the post-commit tenant lookup is stubbed to fail")
	require.NotNil(t, repo.created, "the invite must have been created before that")
	assert.Equal(t, hashInviteToken(issued), repo.created.InviteTokenHash)
	assert.NotEqual(t, issued, repo.created.InviteTokenHash)
}
