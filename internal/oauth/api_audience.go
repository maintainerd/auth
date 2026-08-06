package oauth

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// API projects the apis table so a token request can name a registered resource
// server as its audience. It carries no json tags: like the other projections in
// deps.go it never touches the wire, it exists so the oauth package does not
// import the api domain.
type API struct {
	APIID     int64     `gorm:"column:api_id;primaryKey"`
	APIUUID   uuid.UUID `gorm:"column:api_uuid"`
	TenantID  int64     `gorm:"column:tenant_id"`
	ServiceID int64     `gorm:"column:service_id"`
	Name      string    `gorm:"column:name"`
	// Identifier is the API's externally-addressable name and is what an RFC 8707
	// `resource` / RFC 8693 `audience` parameter names. It is unique per tenant
	// (migration 011, uq_apis_tenant_identifier).
	Identifier string         `gorm:"column:identifier"`
	Status     string         `gorm:"column:status"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (API) TableName() string { return "apis" }

// ClientAPI projects the client_apis join: which registered APIs a client may
// request a token for.
type ClientAPI struct {
	ClientAPIID   int64     `gorm:"column:client_api_id;primaryKey"`
	ClientAPIUUID uuid.UUID `gorm:"column:client_api_uuid"`
	ClientID      int64     `gorm:"column:client_id"`
	APIID         int64     `gorm:"column:api_id"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (ClientAPI) TableName() string { return "client_apis" }

// newOAuthInvalidTarget builds the RFC 8707 §2 / RFC 8693 §2.2.2 `invalid_target`
// error. It is not in apperror's constructor set because only the audience path
// raises it.
func newOAuthInvalidTarget(description string) *apperror.OAuthError {
	return &apperror.OAuthError{
		Code:        "invalid_target",
		Description: description,
		StatusCode:  400,
	}
}

// resolveRequestedAudience turns the caller's `audience` / `resource` parameters
// into the `aud` claim of the token about to be minted.
//
// Every access token used to be stamped with aud = the requesting client's own
// identifier, which made the apis / client_apis / api_permissions model
// unreachable: a resource server that correctly rejects a token whose `aud` is
// not its own identifier (RFC 9068 §4, RFC 7519 §4.1.3) could never be handed a
// usable token, so operators were pushed to disable audience checking entirely.
//
// The empty return value means "the caller named no resource"; the caller keeps
// its own default. Anything the caller DID name must resolve to an API that is
// (a) registered in the caller's own tenant, (b) active and not soft-deleted,
// and (c) linked to this client through client_apis. Every other outcome —
// including a nil db or a query error — is invalid_target, because an audience
// that cannot be verified must never be minted onto a token: `aud` is what a
// resource server trusts to decide the token was meant for it.
func resolveRequestedAudience(db *gorm.DB, client *Client, audience, resource string) (string, *apperror.OAuthError) {
	target, oerr := normalizeRequestedTarget(audience, resource)
	if oerr != nil {
		return "", oerr
	}
	if target == "" {
		return "", nil
	}

	if client == nil {
		return "", newOAuthInvalidTarget("the requested audience could not be verified")
	}
	if oerr := verifyClientAudience(db, client.TenantID, client.ClientID, target); oerr != nil {
		return "", oerr
	}
	return target, nil
}

// normalizeRequestedTarget collapses the caller's `audience` (RFC 8693 §2.1) and
// `resource` (RFC 8707 §2) parameters into the single target they must agree on,
// or reports invalid_target. An empty target with a nil error means the caller
// named none.
//
// This is the only parser for "what target did the caller ask for", on either
// token-exchange leg. The credentialed RFC 8693 leg reaches it through
// resolveRequestedAudience; the keyless workload-identity leg reaches it in
// OAuthTokenExchangeHandler.Exchange, which normalizes before the federation
// exchanger is handed a target at all — so the federation domain never sees a
// raw caller parameter and has no reason to parse one. Two parsers would be two
// different answers, and the looser one is the one an attacker uses.
func normalizeRequestedTarget(audience, resource string) (string, *apperror.OAuthError) {
	audience = strings.TrimSpace(audience)
	resource = strings.TrimSpace(resource)

	if resource != "" {
		// RFC 8707 §2: `resource` MUST be an absolute URI without a fragment.
		u, err := url.Parse(resource)
		if err != nil || !u.IsAbs() || u.Fragment != "" {
			return "", newOAuthInvalidTarget("resource must be an absolute URI without a fragment")
		}
	}

	target := audience
	switch {
	case target == "":
		target = resource
	case resource != "" && resource != audience:
		// Honouring one and dropping the other would silently mint a token for a
		// resource the caller did not ask for.
		return "", newOAuthInvalidTarget("audience and resource must not name different targets")
	}

	return target, nil
}

// verifyClientAudience reports whether `target` is an audience this client may
// be issued a token for: a registered API that is (a) in the client's own
// tenant, (b) active and not soft-deleted, and (c) linked to the client through
// client_apis.
//
// Deliberately NOT exported, and neither is normalizeRequestedTarget. An earlier
// pass exported both plus a ResolveWorkloadAudience wrapper so the keyless
// workload-identity path "would not grow its own copy" of the audience rule; the
// federation package grew one anyway (federation.resolveWorkloadAudience), the
// wrapper ended up with no caller, and the two rules drifted apart — the live one
// admits only the federation's registered audience while the dead one also
// admitted any client_apis-linked target. Two answers to one security question,
// with the looser one unreachable and therefore unreviewed, is worse than one
// answer. The federation rule is the stricter and is the one that runs; this
// helper now serves only the credentialed RFC 8693 / RFC 8707 path in this
// package. Export it again only by making federation delegate to it and deleting
// the local copy in the same change.
func verifyClientAudience(db *gorm.DB, tenantID, clientID int64, target string) *apperror.OAuthError {
	// A blank target or a missing handle is not "no restriction": it is an
	// audience nobody checked, and stamping it would tell a resource server the
	// token was meant for it on no evidence at all.
	if strings.TrimSpace(target) == "" || db == nil {
		return newOAuthInvalidTarget("the requested audience could not be verified")
	}

	var count int64
	err := db.
		Model(&API{}).
		Joins("JOIN client_apis ON client_apis.api_id = apis.api_id").
		Where("apis.tenant_id = ?", tenantID).
		Where("apis.identifier = ?", target).
		Where("apis.status = ?", shared.StatusActive).
		Where("apis.deleted_at IS NULL").
		Where("client_apis.client_id = ?", clientID).
		Count(&count).Error
	if err != nil {
		return newOAuthInvalidTarget("the requested audience could not be verified")
	}
	if count == 0 {
		return newOAuthInvalidTarget("the requested audience is not an API this client may request a token for")
	}
	return nil
}
