# Back-Channel Logout

Back-Channel Logout allows the authorization server (or another RP in a federation) to notify this server to terminate a user's session by sending a signed Logout Token JWT directly to the logout endpoint — without any browser involvement.

When a valid logout token is received, the server revokes all refresh tokens associated with the identified user.

**Spec reference:** [OpenID Connect Back-Channel Logout 1.0 §2.5](https://openid.net/specs/openid-connect-backchannel-1_0.html#BCRequest)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/logout/backchannel` | Logout Token (self-contained) | 8081 |

---

## POST /api/v1/oauth/logout/backchannel

Receives a Logout Token JWT, validates it, identifies the user from the `sub` claim, and revokes all active refresh tokens for that user.

The Logout Token must be a signed JWT verifiable against the server's public key. The token must contain a `sub` claim identifying the user. The `client_id` is resolved from the token's `client_id` claim.

There is no response body on success. Per the OIDC Back-Channel Logout specification, the server responds with `200 OK` and an empty body.

### Authentication

This endpoint does not require a session. Authentication is provided by the Logout Token JWT itself. The token's signature is verified against the server's configured signing key.

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `logout_token` | string | Yes | A signed JWT Logout Token. Must contain a `sub` claim. |

#### Logout Token Claims

The logout token is a JWT with the following relevant claims:

| Claim | Description |
|-------|-------------|
| `sub` | Subject — the user identifier. Required. |
| `client_id` | The client the session belongs to. Used to scope the user lookup. |
| `iat` | Issued-at time. |
| `jti` | JWT ID. Should be unique to prevent replay. |

### Response

#### Success — 200 OK

Empty body. No `Content-Type` header is set on success.

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `logout_token` field is missing. |
| 400 | `invalid_request` | Logout token JWT is invalid, expired, or its signature cannot be verified. |
| 400 | `invalid_request` | Logout token is missing the `sub` claim. |
| 500 | `server_error` | Unexpected internal error during user lookup or token revocation. |

### Example

```bash
curl -X POST https://auth.example.com/api/v1/oauth/logout/backchannel \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "logout_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

Success response: `200 OK` with empty body.

Error response:

```json
{
  "error": "invalid_request",
  "error_description": "logout_token is invalid or expired"
}
```

### Notes

- If the `sub` claim resolves to a user that does not exist in the system, the request silently succeeds (no tokens to revoke).
- Replay protection should be implemented by the caller using the `jti` claim. The server does not currently enforce `jti` uniqueness.
- This endpoint is intended to be called server-to-server. It should not be called from a browser.
