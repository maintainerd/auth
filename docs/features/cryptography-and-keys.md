# Cryptography & Key Management

> How maintainerd-auth signs JWTs (RS256), rotates and publishes its signing keys, binds tokens to client keys (DPoP), and encrypts secrets at rest.

| | |
|---|---|
| **Status** | Implemented |
| **Code** | `internal/platform/jwt`, `internal/platform/crypto`, `internal/platform/dpop`, `internal/oauth` (signing-key store, rotation runner, JWKS) |
| **Endpoints** | `GET /.well-known/jwks.json` (public key set); `GET /.well-known/openid-configuration` (advertises `jwks_uri`) |
| **Storage** | `signing_keys` table (migration `073_create_signing_keys_table.go`); in-memory process key store when the key comes from env |
| **Config** | `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY`, `JWT_KEY_ID`, `JWT_KEY_ROTATION_PERIOD_SECONDS`, `APP_PUBLIC_HOSTNAME`, `APP_ENCRYPTION_KEY`, `APP_ENCRYPTION_KEYS_PREVIOUS` |

## Overview

This feature covers every cryptographic primitive the auth server relies on:

- **JWT signing & verification** — RS256 over an RSA-2048+ key, with a `kid` on every token and a multi-key JWKS so tokens signed by a just-rotated key still verify.
- **Signing-key lifecycle** — two mutually exclusive ownership models (operator env key vs. DB-backed auto-generated key), automatic rotation on the DB path, and a graceful retirement window.
- **PKCE, hashing, and secure randomness** (`internal/platform/crypto`) — S256 PKCE validation, SHA-256 hashing of codes/tokens before storage, CSPRNG identifiers/OTPs, and AES-256-GCM envelope encryption for secrets at rest.
- **DPoP** (`internal/platform/dpop`) — RFC 9449 proof-of-possession that binds an access token to a client's ephemeral key via `cnf.jkt`.

The token-*content* settings (clock-skew leeway config, extra-claim lists) are documented separately — see the token-config settings doc referenced under [Related](#related). This doc is the crypto/key-material source of truth.

## How it works

### JWT signing

1. A signing key is loaded at startup (env or DB — see [Signing-key ownership](#signing-key-ownership--rotation)) into a process-global `keyStore` (`internal/platform/jwt/jwt.go:69`). The store holds one **active** private/public key + `kid`, plus a slice of **retiring** public keys.
2. Token issuance (`GenerateAccessToken*`, `GenerateIDToken*`, `GenerateRefreshToken*`, `GenerateLogoutToken`, step-up challenge tokens) builds the claim set, then calls `generateTokenWithAlgorithm` (`jwt.go:934`) which signs with the active key and stamps `token.Header["kid"]`.
3. **Algorithm**: the default and effective algorithm is **RS256** (`jwt.go:950`). `generateTokenWithAlgorithm` additionally accepts `PS256` (RSA-PSS over the same RSA key) as an opt-in per-issuance `SigningAlgorithm`; `ES256` is explicitly rejected (`jwt.go:955`) because the store holds no ECDSA key. In practice all standard paths leave `SigningAlgorithm` empty, so every minted token is RS256.

### JWT validation

`ValidateToken` (`jwt.go:1063`) parses with a **fixed 30s clock-skew leeway** (`tokenValidationLeeway`, `jwt.go:51` — deliberately NOT a per-tenant value, to preserve tenant isolation) and:

1. Enforces the signing method: RSA methods must be RS256, RSA-PSS must be PS256; anything else (including `alg:none`, HMAC, or unexpected RSA variants) is rejected — this is the algorithm-confusion guard (`jwt.go:1094`).
2. Resolves the verification key **by `kid`** via `keyStore.verificationKey` (`jwt.go:162`): the active key, or any retiring key still inside the window. A token with no `kid` falls back to the active key.
3. Runs `validateTokenClaims` (`jwt.go:1153`): requires `sub`/`aud`/`iss`/`iat`/`exp`/`jti`, and matches `iss` against the accepted-issuer allowlist (`validateIssuerClaim`, `token_type.go:192`).
4. Optionally consults a JTI denylist (`SetJTIChecker`) to reject explicitly revoked tokens.

`ValidateAccessTokenWithContext` (`token_type.go:62`) layers a **token-type** check on top so an ID token cannot be replayed as a Bearer access token.

### Issuer derivation (drift-corrected)

The `iss` value is **derived from `APP_PUBLIC_HOSTNAME`**, not any legacy `ISSUER_URL`. `TokenIssuer` (`internal/platform/jwt/issuer.go:24`) returns `config.AppPublicHostname` when set, falling back to the client domain only when it is unset. Because all tenants share one signing key, the `iss` match **is** the tenant boundary: `validateIssuerClaim` fails **closed** — an empty allowlist accepts only the server's own issuer (`APP_PUBLIC_HOSTNAME`) and rejects everything else (`token_type.go:197`).

### Signing-key ownership & rotation

Two mutually exclusive models, chosen at boot by whether `JWT_PRIVATE_KEY` is set (`cmd/server/bootstrap.go:49`, `:88`):

| | Env-owned key | DB-owned key (self-host default) |
|---|---|---|
| **Trigger** | `JWT_PRIVATE_KEY` + `JWT_PUBLIC_KEY` set | `JWT_PRIVATE_KEY` unset |
| **Load path** | `InitJWTKeys` (`jwt.go:259`) parses PEM, validates the pair, installs under `JWT_KEY_ID` (default `maintainerd-auth-key-1`) | `EnsureGlobalSigningKeyFromDB` (`startup_signing_key.go:24`) loads the active `signing_keys` row, or generates + persists an RSA-2048 key |
| **Auto-rotation** | **Disabled** — the operator rotates by changing the variable and redeploying (`bootstrap.go:65`) | `StartSigningKeyRotationRunner` (`signing_key_rotation_runner.go:36`), every `JWT_KEY_ROTATION_PERIOD_SECONDS` (default 24h) |
| **Why the split** | An in-memory rotation would sign with a key no restart/replica can reload | Rotation writes a new row so every replica converges on the published key set |

**DB rotation** (`RotateGlobalSigningKey`, `startup_signing_key.go:53`): generate a fresh RSA-2048 key → persist it as a new `active` row → install as the active signer **only after commit** → retire superseded rows older than the retention window (`RefreshTokenTTL` = 7 days). Superseded keys stay `active` (and therefore in JWKS) until they age out, so tokens they signed keep verifying.

**In-memory rotation** (`jwt.RotateKeys` / `runner.StartKeyRotationRunner`) promotes the active key to `retiringKeys`, generates a new RSA-2048 key, and prunes retiring keys older than `RefreshTokenTTL` (`jwt.go:106`, `:224`). This runner is wired as a var (`cmd/server/workers.go:21`) but is **not invoked** by the current startup path — only the DB-backed runner runs, and only when `JWT_PRIVATE_KEY` is unset. Neither runner rotates at startup (a boot key is seconds old; rotating it would burn a key per restart).

### JWKS publication

`GET /.well-known/jwks.json` (`internal/oauth/handler_discovery.go:144`) publishes the **union** of DB-backed keys (`keySvc.ListJWKS`) and the in-memory store's keys (`jwt.GetAllPublicKeys` — active + retiring), de-duplicated by `kid`. Each entry is `{kty:"RSA", use:"sig", alg:"RS256", kid, n, e}`. The union exists because a JWKS must list every key that can verify an outstanding token; anything narrower is a verification outage, and every value here is a public key so the union discloses nothing.

### PKCE, hashing & randomness (`internal/platform/crypto`)

- **PKCE**: `ValidatePKCEChallenge` (`pkce.go:27`) requires `S256` only, a 43–128 char verifier matching the RFC 7636 charset, and `BASE64URL(SHA256(verifier)) == code_challenge`. `plain` is unsupported.
- **Hashing before storage**: authorization codes, refresh tokens, and OAuth browser-binding secrets are stored as `BASE64URL(SHA256(...))` — the raw values are never persisted (`pkce.go:55`–`76`).
- **Randomness**: all identifiers/OTPs draw from `crypto/rand` — `GenerateIdentifier`, `GenerateRandomString` (`rand.go`), `GenerateOTP` (`otp.go`), and the JWT `GenerateSecureID` / `generateSecureJTI` (`jwt.go:55`, `:235`, which panic on RNG failure rather than emit weak IDs).
- **Encryption at rest**: AES-256-GCM with a random 12-byte nonce prepended to the ciphertext (`encrypt.go:22`). Stored values are tagged `k1:<key-id>:<b64>` where `key-id` is a truncated SHA-256 fingerprint of the key, so `APP_ENCRYPTION_KEY` can rotate while retired keys (`APP_ENCRYPTION_KEYS_PREVIOUS`) still decrypt old rows. `SafeDecryptAtRest` fails **closed** for tagged values (logs + returns empty) and only falls back to plaintext for legacy untagged rows.

### DPoP (RFC 9449)

`internal/platform/dpop` binds tokens to a client's ephemeral key so a stolen bearer token is useless without the private key. It is wired into the token endpoint (`internal/oauth/handler_token.go`).

1. On the token request the client sends a `DPoP` proof JWT. `ValidateProof` (`dpop.go:78`) checks `typ=dpop+jwt`, extracts the embedded JWK, verifies the proof signature (EC or RSA), and validates `jti`/`htm`/`htu`/`iat` (proof max age 5 min) and, when present, `ath`.
2. The RFC 7638 **SHA-256 JWK thumbprint** is computed and embedded as `cnf.jkt` in the issued access token; the token's `token_type` becomes `DPoP` (`jwt.go:504`).
3. On refresh, the stored `DPoPJKT` is compared to the presented thumbprint with a **constant-time** compare (`service_token.go:401`).
4. **Replay prevention**: the proof `jti` is denylisted in Redis (`JTIStore`) scoped by thumbprint (`dpop:<thumbprint>:<jti>`); cache errors fail closed. A `DPoP-Nonce` challenge mechanism (`nonce.go`, `StoreNonceManager`) is available via `SetDPoPNonceGate`.
5. `htu` matching (`dpop.go:350`) is case-insensitive on scheme/host but **case-sensitive on path**, excluding query/fragment.

## Implementation

| Concern | File / symbol |
|---|---|
| Process key store (active + retiring, `kid` lookup, rotation, pruning) | `internal/platform/jwt/jwt.go:69` (`keyStore`) |
| Env key load + RSA-2048 min-strength validation | `jwt.go:259` (`InitJWTKeys`), `jwt.go:252` (`validateKeyStrength`, `MinKeySize=2048`) |
| Token issuance (access/ID/refresh/logout/step-up) | `jwt.go:421`, `:663`, `:847`, `:1238`, `:992` |
| Sign + algorithm select (RS256 default, PS256 opt-in, ES256 rejected) | `jwt.go:934` (`generateTokenWithAlgorithm`) |
| Validation + algorithm-confusion guard + JTI denylist | `jwt.go:1063` (`ValidateToken`), `jwt.go:1153` (`validateTokenClaims`) |
| Token-type gate + issuer allowlist (fail-closed) | `internal/platform/jwt/token_type.go:62`, `:192` |
| Issuer derivation from `APP_PUBLIC_HOSTNAME` | `internal/platform/jwt/issuer.go:24` |
| DB signing-key model / migration | `internal/oauth/model_signing_key.go`, `migration/073_create_signing_keys_table.go` |
| DB startup load + auto-generate + rotate | `internal/oauth/startup_signing_key.go:24`, `:53` |
| DB rotation runner (24h default) | `internal/oauth/signing_key_rotation_runner.go:36` |
| In-memory rotation runner (declared, not wired) | `internal/platform/runner/key_rotation.go:25` |
| JWKS endpoint (DB ∪ in-memory, dedup by `kid`) | `internal/oauth/handler_discovery.go:144` |
| PKCE S256 / hashing / PKCE-gen | `internal/platform/crypto/pkce.go` |
| CSPRNG identifiers / random strings / OTP | `internal/platform/crypto/rand.go`, `otp.go` |
| AES-256-GCM envelope encryption + key-id tagging | `internal/platform/crypto/encrypt.go` |
| DPoP proof validation / thumbprint / replay | `internal/platform/dpop/dpop.go`, `keys.go`, `nonce.go` |

**Signing-key algorithm note**: the `signing_keys` CHECK constraint allows `RS256/384/512, ES256/384/512, EdDSA`, but the code only ever generates and installs **RS256** keys (`startup_signing_key.go:113`).

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | unset | PEM RSA key pair. When set, key is operator-managed & process-local; auto-rotation is disabled. When unset, the DB-backed key path is used. |
| `JWT_KEY_ID` | `maintainerd-auth-key-1` | `kid` for the env-loaded key (`jwt.go:292`). |
| `JWT_KEY_ROTATION_PERIOD_SECONDS` | `86400` (24h) | DB-backed rotation interval (`config.go:226`; runner clamps ≤0 to 24h). |
| `APP_PUBLIC_HOSTNAME` | required | Source of the token `iss` and the issuer allowlist fallback. |
| `APP_ENCRYPTION_KEY` | required, 32 bytes | AES-256 key for encrypt-at-rest and for encrypting DB-stored signing-key private keys (`config.go:236`). |
| `APP_ENCRYPTION_KEYS_PREVIOUS` | unset | Comma-separated 32-byte retired keys, decrypt-only, for rotating `APP_ENCRYPTION_KEY` (`config.go:242`). |
| `SECRET_REFRESH_PERIOD_SECONDS` | `300` | How often the secret-refresh runner reloads env-provided key material (`runner/secret_refresh.go`). |

No per-tenant crypto settings exist — one process-wide signing key serves all tenants (which is why the `iss` allowlist is the tenant boundary).

## Security considerations

- **Algorithm confusion is blocked** at validation: only RS256/PS256 RSA methods are accepted; `alg:none`, HMAC, and mismatched RSA variants are rejected (`jwt.go:1094`).
- **Minimum key strength** RSA-2048 is enforced on every loaded/installed key (`validateKeyStrength`); env key pairs are additionally checked for `N`/`E` consistency.
- **Tenant isolation** rests on the fail-closed issuer check: an unconfigured allowlist accepts only the server's own issuer, never "anything."
- **Rotation never orphans tokens**: retiring keys stay in JWKS for one `RefreshTokenTTL` (7d) window; the DB runner installs a new signer only after the row commits, and a failed rotation leaves the previous persisted key active.
- **Clock-skew leeway is fixed at 30s** and intentionally not tenant-configurable, closing a prior last-writer-wins tenant-isolation hole (`jwt.go:36`).
- **Secrets at rest** use authenticated AES-256-GCM; the key-id tag makes encryption-key rotation possible and makes a lost key diagnosable instead of silently returning ciphertext. Signing-key private keys stored in the DB are encrypted with `APP_ENCRYPTION_KEY` (`aes256gcm-v1`), falling back to plaintext storage only when no key is configured.
- **DPoP** turns a stolen bearer token into a non-credential: the `cnf.jkt` binding, constant-time thumbprint compare on refresh, per-thumbprint replay denylist, and fail-closed cache behavior all resist theft and replay.
- **CSPRNG everywhere**: identifiers, OTPs, JTIs, GCM nonces, and DPoP nonces all draw from `crypto/rand`; JTI/ID generation panics rather than degrade to weak randomness.

## Related

- [./authentication.md](./authentication.md) — login flows that mint the tokens signed here (AMR/ACR, step-up).
- [./sessions.md](./sessions.md) — refresh-token rotation, revocation, and the JTI denylist that validation consults.
- [./security-settings.md](./security-settings.md) — per-tenant token/clock-skew config surface.
- [./federation.md](./federation.md) — multi-issuer validation and `IsSelfIssued` classification.
- [./setup-and-bootstrap.md](./setup-and-bootstrap.md) — startup key selection (env vs. DB) and required env vars.
- [./multi-tenancy.md](./multi-tenancy.md) — why one shared signing key makes `iss` the tenant boundary.

</content>
</invoke>
