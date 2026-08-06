package client

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// Rotated-secret grace period
// ===========================================================================

// The 168-hour cap used to live only in the HTTP DTO. The gRPC handler passes
// req.GetGracePeriodHours() straight through and validates nothing, so a rotation
// with 876000 hours kept the compromised previous secret accepted for a century
// while the tenant saw a success response and a client.secret_rotated event. The
// service is the only layer every transport passes through, so the cap belongs
// there.
func TestClientService_RotateSecret_EnforcesGracePeriodCap(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	rotate := func(t *testing.T, hours int) error {
		t.Helper()
		gormDB, _ := newMockGormDB(t)
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				t.Fatal("the transaction must not open for an out-of-range grace period")
				return nil, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, hours)
		return err
	}

	t.Run("a century-long grace period is refused", func(t *testing.T) {
		err := rotate(t, 876000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "grace_period_hours")
	})

	t.Run("one hour past the cap is refused", func(t *testing.T) {
		require.Error(t, rotate(t, maxSecretGracePeriodHours+1))
	})

	t.Run("a negative grace period is refused", func(t *testing.T) {
		require.Error(t, rotate(t, -1))
	})

	// The cap itself must still be usable, or the fix would just move the bug.
	t.Run("the cap itself is accepted", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
				return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		secret, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, maxSecretGracePeriodHours)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})
}

// ===========================================================================
// Reversible secret copy (secret_encrypted)
// ===========================================================================

// secret_encrypted is AES under one app-wide key, so it is recoverable plaintext
// at rest. Only client_secret_jwt reads it (it HMACs the client assertion with the
// secret); client_secret_basic/_post verify the bcrypt hash and never touch it.
// Writing it for every confidential client was a second, weaker credential store
// with no consumer.
func TestClientService_RotateSecret_EncryptedCopyOnlyForClientSecretJWT(t *testing.T) {
	cUUID := uuid.New()
	actorUUID := uuid.New()
	tenantID := int64(1)

	rotate := func(t *testing.T, existing *Client) *Client {
		t.Helper()
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		var saved *Client
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) { return existing, nil },
			createOrUpdateFn: func(c *Client) (*Client, error) {
				saved = c
				return c, nil
			},
		}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)
		_, err := svc.RotateSecret(context.Background(), cUUID, tenantID, actorUUID, 0)
		require.NoError(t, err)
		require.NotNil(t, saved)
		return saved
	}

	t.Run("client_secret_jwt keeps the reversible copy", func(t *testing.T) {
		saved := rotate(t, &Client{
			ClientID: 1, ClientUUID: cUUID, TenantID: tenantID,
			TokenEndpointAuthMethod: TokenAuthMethodClientSecretJWT,
		})
		require.NotNil(t, saved.SecretEncrypted, "client_secret_jwt HMACs with the plaintext secret")
		assert.NotEmpty(t, *saved.SecretEncrypted)
	})

	for _, method := range []string{TokenAuthMethodSecretBasic, TokenAuthMethodSecretPost} {
		t.Run(method+" stores only the hash", func(t *testing.T) {
			saved := rotate(t, &Client{
				ClientID: 1, ClientUUID: cUUID, TenantID: tenantID,
				TokenEndpointAuthMethod: method,
			})
			assert.Nil(t, saved.SecretEncrypted, "%s verifies the bcrypt hash and never reads this column", method)
			require.NotNil(t, saved.SecretHash)
		})
	}

	// A client that was created before this rule, or that has since moved off
	// client_secret_jwt, must have its stale reversible copy dropped on rotation
	// rather than carried forward.
	t.Run("rotating off client_secret_jwt clears a stale copy", func(t *testing.T) {
		stale := "test-enc:previously-stored"
		saved := rotate(t, &Client{
			ClientID: 1, ClientUUID: cUUID, TenantID: tenantID,
			TokenEndpointAuthMethod: TokenAuthMethodSecretBasic,
			SecretEncrypted:         &stale,
		})
		assert.Nil(t, saved.SecretEncrypted)
	})
}

// ===========================================================================
// Client identifier allocation
// ===========================================================================

// identifier is the OAuth client_id every authorize and token request resolves a
// client by, and clients has a global UNIQUE index on it (uq_clients_identifier).
// Generation was unchecked, so a collision surfaced as a raw constraint violation
// on create instead of a retry.
func TestGenerateUniqueClientIdentifier(t *testing.T) {
	t.Run("retries past a taken identifier", func(t *testing.T) {
		var checked []string
		repo := &mockClientRepo{
			existsByIdentifierFn: func(identifier string) (bool, error) {
				checked = append(checked, identifier)
				return len(checked) == 1, nil // the first candidate is already taken
			},
		}
		got, err := generateUniqueClientIdentifier(repo)
		require.NoError(t, err)
		require.Len(t, checked, 2)
		assert.Equal(t, checked[1], got)
		assert.NotEqual(t, checked[0], got)
		assert.Len(t, got, clientIdentifierLength)
	})

	// Exhausting the retries means the generator is broken, not that we were
	// unlucky — handing out an unchecked identifier would be worse than failing.
	t.Run("gives up rather than returning an unchecked identifier", func(t *testing.T) {
		attempts := 0
		repo := &mockClientRepo{
			existsByIdentifierFn: func(string) (bool, error) {
				attempts++
				return true, nil
			},
		}
		_, err := generateUniqueClientIdentifier(repo)
		require.Error(t, err)
		assert.Equal(t, clientIdentifierAttempts, attempts)
	})

	t.Run("propagates a lookup failure instead of assuming the name is free", func(t *testing.T) {
		repo := &mockClientRepo{
			existsByIdentifierFn: func(string) (bool, error) { return false, errors.New("db down") },
		}
		_, err := generateUniqueClientIdentifier(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db down")
	})
}

// The uniqueness question is "does any row hold this value", which is not what
// FindByIdentifier answers: it filters to active clients. An inactive or
// soft-deleted client still owns its client_id, and reusing it would re-point
// anything still presenting the old value at a different client.
func TestClientRepository_ExistsByIdentifier_IgnoresStatusAndSoftDelete(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "clients" WHERE identifier = \$1`).
		WithArgs("abc123").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	repo := NewClientRepository(gormDB)
	exists, err := repo.ExistsByIdentifier("abc123")
	require.NoError(t, err)
	assert.True(t, exists)

	// No status predicate and no deleted_at predicate: an inactive or soft-deleted
	// row must still count as taken.
	require.NoError(t, mock.ExpectationsWereMet())
}

// ===========================================================================
// is_default
// ===========================================================================

// is_default is platform-owned: the seeder sets it on the bootstrap client, the
// table allows one per tenant, and Update/SetStatus/Delete all refuse a client
// that carries it. A caller-settable flag therefore minted a client nobody could
// ever edit, deactivate or delete.
func TestClientService_Create_NeverMarksTheClientDefault(t *testing.T) {
	tenantID := int64(1)
	actorUUID := uuid.New()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	expectClientIdentityProviderConnectionInsert(mock)
	mock.ExpectCommit()

	idpRepo := &mockIdentityProviderRepo{
		findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
		},
	}
	var created *Client
	clientRepo := &mockClientRepo{
		findByNameAndTenantIDFn: func(string, int64) (*Client, error) { return nil, nil },
		createOrUpdateFn: func(c *Client) (*Client, error) {
			c.ClientID = 1
			created = c
			return c, nil
		},
		findByUUIDFn: func(any, ...string) (*Client, error) { return clientWithIDP(tenantID), nil },
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, idpRepo,
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)

	_, err := svc.Create(context.Background(), tenantID, "app", "App", "public", "example.com",
		nil, "active", uuid.New().String(), nil, true, nil, nil, nil, nil, actorUUID, nil)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.False(t, created.IsDefault, "is_default is not tenant-admin input")
	assert.False(t, created.IsSystem)
}

// Update must not promote an ordinary client either — the same trap, reached from
// the other side.
func TestClientService_Update_DoesNotPromoteToDefault(t *testing.T) {
	tenantID := int64(1)
	cUUID := uuid.New()
	actorUUID := uuid.New()

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var saved *Client
	clientRepo := &mockClientRepo{
		findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Client, error) {
			return &Client{ClientID: 1, ClientUUID: cUUID, TenantID: tenantID, Name: "app", ClientType: "spa"}, nil
		},
		findByNameAndTenantIDFn: func(string, int64) (*Client, error) { return nil, nil },
		createOrUpdateFn: func(c *Client) (*Client, error) {
			saved = c
			return c, nil
		},
		findByUUIDFn: func(any, ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: tenantID, Tenant: &Tenant{TenantID: tenantID}}, nil
		},
	}
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)

	_, err := svc.Update(context.Background(), cUUID, tenantID, "app", "App", "spa", "example.com",
		nil, "active", nil, nil, nil, nil, nil, nil, nil, actorUUID, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.False(t, saved.IsDefault)
}

// ===========================================================================
// CORS allow-list
// ===========================================================================

// onTenantSurface builds the context the CORS middleware supplies once
// RequestTenantMiddleware has run: the tenant derived from the REQUEST's own
// host, never from anything the caller sends.
func onTenantSurface(slug string) context.Context {
	return middleware.WithRequestTenant(context.Background(),
		middleware.RequestTenant{Surface: shared.FrontendSurfaceIdentity, Slug: slug, OK: true})
}

func onSystemTenantSurface() context.Context {
	return middleware.WithRequestTenant(context.Background(),
		middleware.RequestTenant{Surface: shared.FrontendSurfaceIdentity, IsSystem: true, OK: true})
}

// corsOriginRows builds the tenant-scoped result set refresh() now scans.
func corsOriginRows(rows ...[3]any) *sqlmock.Rows {
	out := sqlmock.NewRows([]string{"tenant_name", "is_system", "uri"})
	for _, r := range rows {
		out = out.AddRow(r[0], r[1], r[2])
	}
	return out
}

// The clients side of the join is a raw string, so GORM's soft-delete scope only
// covers client_uris. Without an explicit predicate a soft-deleted client's origin
// stays in the allow-list; that it currently does not is incidental to
// DeleteByUUID soft-deleting the URI rows first.
func TestCORSOriginResolver_ExcludesSoftDeletedClients(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	// The expectation IS the assertion: sqlmock's regexp matcher only satisfies
	// this if the emitted SQL carries a deleted_at predicate on BOTH sides of the
	// join. client_uris already had one; clients did not.
	mock.ExpectQuery(`client_uris\.deleted_at IS NULL.*clients\.deleted_at IS NULL`).
		WillReturnRows(corsOriginRows([3]any{"acme", false, "https://app.example"}))

	resolver := NewCORSOriginResolver(gormDB)
	assert.True(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://app.example"))
	require.NoError(t, mock.ExpectationsWereMet())
}

// A soft-deleted or suspended TENANT keeps its DNS surface resolvable for a
// while, so its clients' origins must drop out of the allow-list the same way a
// deleted client's do.
func TestCORSOriginResolver_FiltersOnTenantLifecycle(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`tenants\.deleted_at IS NULL.*tenants\.status`).
		WillReturnRows(corsOriginRows([3]any{"acme", false, "https://app.example"}))

	resolver := NewCORSOriginResolver(gormDB)
	assert.True(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://app.example"))
	require.NoError(t, mock.ExpectationsWereMet())
}

// The allow-list used to be one flat set with no tenant predicate, so an origin
// any tenant registered received credentialed CORS on EVERY tenant's surface.
// Combined with Access-Control-Allow-Credentials and the CSRF exemption on GETs,
// that let a tenant read another tenant's users' cookie-authenticated responses.
func TestCORSOriginResolver_IsScopedToTheRequestTenant(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`client_uris`).WillReturnRows(corsOriginRows(
		[3]any{"acme", false, "https://acme-app.example"},
		[3]any{"evil", false, "https://attacker.example"},
		[3]any{"root", true, "https://ops.example"},
	))
	resolver := NewCORSOriginResolver(gormDB)

	// Each tenant gets exactly its own registrations.
	assert.True(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://acme-app.example"))
	assert.True(t, resolver.IsAllowedCORSOrigin(onTenantSurface("evil"), "https://attacker.example"))
	assert.True(t, resolver.IsAllowedCORSOrigin(onSystemTenantSurface(), "https://ops.example"))

	// …and nothing another tenant registered.
	assert.False(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://attacker.example"))
	assert.False(t, resolver.IsAllowedCORSOrigin(onSystemTenantSurface(), "https://attacker.example"))
	assert.False(t, resolver.IsAllowedCORSOrigin(onTenantSurface("evil"), "https://acme-app.example"))
	// The system tenant is not a wildcard scope, and a tenant literally named
	// after the system sentinel cannot reach it either.
	assert.False(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://ops.example"))
	require.NoError(t, mock.ExpectationsWereMet())
}

// The system tenant answers on TWO hosts: the bare base host (which
// shared.ResolveTenantHost reports as IsSystem with no slug) and, like every
// other tenant, {its name}.<base> (which it reports as a plain slug — that is
// how the authorize endpoint binds it, via ResolveTenantIDByName(rt.Slug)).
// Indexing its rows only under the system sentinel denied every origin it
// registered whenever the browser was on its named subdomain, so the feature
// looked dead on the one tenant that ships with the product.
func TestCORSOriginResolver_SystemTenantIsScopedToBothOfItsHosts(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	mock.ExpectQuery(`client_uris`).WillReturnRows(corsOriginRows(
		[3]any{"maintainerd", true, "https://ops.example"},
		[3]any{"acme", false, "https://acme-app.example"},
	))
	resolver := NewCORSOriginResolver(gormDB)

	// Bare base host.
	assert.True(t, resolver.IsAllowedCORSOrigin(onSystemTenantSurface(), "https://ops.example"))
	// maintainerd.<base> — same tenant, so the same allow-list.
	assert.True(t, resolver.IsAllowedCORSOrigin(onTenantSurface("maintainerd"), "https://ops.example"))

	// The extra key is the system tenant's own name, not a wildcard: it grants
	// nothing to another tenant and takes nothing from one. tenants.name is
	// unique among live rows (uq_tenants_name) and only one row may carry
	// is_system = true (uq_tenants_single_system), so no other tenant can claim
	// this scope.
	assert.False(t, resolver.IsAllowedCORSOrigin(onTenantSurface("maintainerd"), "https://acme-app.example"))
	assert.False(t, resolver.IsAllowedCORSOrigin(onTenantSurface("acme"), "https://ops.example"))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Granting a credentialed origin is a per-tenant decision, so a request whose
// own host names no tenant has no decision to make and must be denied. It must
// also not fall back to "any tenant's list".
func TestCORSOriginResolver_DeniesWhenTheRequestTenantIsUnknown(t *testing.T) {
	gormDB, mock := newMockGormDBRegex(t)
	resolver := NewCORSOriginResolver(gormDB)

	// No RequestTenant in the context at all.
	assert.False(t, resolver.IsAllowedCORSOrigin(context.Background(), "https://app.example"))
	// Present, but the host matched no configured base.
	unresolved := middleware.WithRequestTenant(context.Background(), middleware.RequestTenant{})
	assert.False(t, resolver.IsAllowedCORSOrigin(unresolved, "https://app.example"))
	// OK but slug-less and not the system tenant is incoherent — deny.
	blankSlug := middleware.WithRequestTenant(context.Background(), middleware.RequestTenant{OK: true})
	assert.False(t, resolver.IsAllowedCORSOrigin(blankSlug, "https://app.example"))

	// None of these may even reach the database: there is nothing to look up.
	require.NoError(t, mock.ExpectationsWereMet())
}
