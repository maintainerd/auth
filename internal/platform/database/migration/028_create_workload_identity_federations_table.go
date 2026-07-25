package migration

import "gorm.io/gorm"

func CreateWorkloadIdentityFederationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS workload_identity_federations (
    workload_identity_federation_id     BIGSERIAL    PRIMARY KEY,
    workload_identity_federation_uuid   UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    tenant_id                           BIGINT       NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id                           BIGINT       NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    name                                VARCHAR(100) NOT NULL,
    description                         TEXT,
    -- 512 rather than 2048: a real OIDC issuer is a short https URL, and this column
    -- participates in uq_workload_identity_federations_trust_key below. At 2048 the
    -- index tuple (issuer + audience + subject_pattern) could exceed btree's ~2704
    -- byte maximum and fail the INSERT with a 500.
    issuer_url                          VARCHAR(512) NOT NULL,
    audience                            VARCHAR(512) NOT NULL,
    subject_claim                       VARCHAR(100) NOT NULL DEFAULT 'sub',
    subject_pattern                     VARCHAR(512) NOT NULL,
    allowed_scopes                      TEXT[]       NOT NULL DEFAULT '{}',
    attribute_mapping                   JSONB        NOT NULL DEFAULT '{}',
    is_active                           BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by                          BIGINT,
    updated_by                          BIGINT,
    created_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                          TIMESTAMPTZ,

    -- Constraint names stay under Postgres' 63-byte identifier limit; the
    -- chk_workload_identity_federations_ prefix alone is 34 characters, so a longer
    -- suffix would be silently truncated and the IF NOT EXISTS guards used elsewhere
    -- would never match it.
    CONSTRAINT chk_wif_name          CHECK (btrim(name) <> ''),
    -- The DTO layer enforces https, but seeders, migrations and any future gRPC or
    -- admin path bypass it entirely.
    CONSTRAINT chk_wif_issuer        CHECK (issuer_url ~ '^https://[^[:space:]/]+'),
    -- go-oidc requires the discovery document's issuer to match byte-for-byte, so a
    -- stored trailing slash can only ever produce a federation that never matches.
    CONSTRAINT chk_wif_issuer_slash  CHECK (issuer_url NOT LIKE '%/'),
    -- An empty audience can never match a token's aud claim, so the row would be a
    -- silently dead trust rule.
    CONSTRAINT chk_wif_audience      CHECK (btrim(audience) <> ''),
    -- Blank subject_claim silently degrades to 'sub' in several code paths; the
    -- column default should be the only source of that fallback.
    CONSTRAINT chk_wif_subj_claim    CHECK (btrim(subject_claim) <> ''),
    -- An empty subject_pattern matches nothing (matchSubjectPattern returns false),
    -- which is safe but useless; a NULL-ish rule should not be storable.
    CONSTRAINT chk_wif_subj_pattern  CHECK (btrim(subject_pattern) <> ''),
    CONSTRAINT chk_wif_attr_map      CHECK (jsonb_typeof(attribute_mapping) = 'object'),
    CONSTRAINT chk_wif_scopes_null   CHECK (array_position(allowed_scopes, NULL) IS NULL),
    CONSTRAINT chk_wif_scopes_blank  CHECK (NOT ('' = ANY (allowed_scopes)))
);

-- Partial unique index: soft-delete-aware (inline UNIQUE constraint would reject
-- re-creating a WIF config with the same name after soft-deleting the old one).
CREATE UNIQUE INDEX IF NOT EXISTS uq_workload_identity_federations_tenant_name
    ON workload_identity_federations (tenant_id, name) WHERE deleted_at IS NULL;

-- The trust key. Two live federations in one tenant with the same
-- (issuer, audience, subject_pattern) but different clients make the exchange path's
-- match ambiguous, and which client won was decided by row order. This cannot
-- prevent OVERLAPPING patterns (repo:org/* vs repo:org/repo:*) — the exchange path
-- rejects cross-tenant ambiguity at match time for that — but it does stop exact
-- duplicates.
CREATE UNIQUE INDEX IF NOT EXISTS uq_workload_identity_federations_trust_key
    ON workload_identity_federations (tenant_id, issuer_url, audience, subject_pattern)
    WHERE deleted_at IS NULL;

-- Every listing is "tenant_id = ? ORDER BY created_at DESC" (FindPaginated), so the
-- index is composite rather than tenant_id alone — which this replaces.
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_tenant_created_at
    ON workload_identity_federations (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

-- The exchange hot path is FindActiveByIssuer: issuer_url = ? AND is_active AND NOT
-- deleted. Folding is_active into the predicate keeps the index to live rules only.
CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_issuer
    ON workload_identity_federations (issuer_url) WHERE deleted_at IS NULL AND is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_workload_identity_federations_client
    ON workload_identity_federations (client_id) WHERE deleted_at IS NULL;
`).Error
}
