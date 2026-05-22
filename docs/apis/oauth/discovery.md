# Discovery & JWKS

These endpoints allow clients and authorization servers to discover metadata about this authorization server and retrieve public keys for token verification. All three are fully public, require no authentication, and are served on the public identity port (8081).

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/.well-known/openid-configuration` | None |
| GET | `/.well-known/oauth-authorization-server` | None |
| GET | `/.well-known/jwks.json` | None |

Responses are cached for 1 hour (`Cache-Control: public, max-age=3600`).

---

## GET /.well-known/openid-configuration

Returns the OpenID Connect Discovery 1.0 document. Use this to auto-configure OIDC relying parties.

### Response

#### Success — 200 OK

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/api/v1/oauth/authorize",
  "token_endpoint": "https://auth.example.com/api/v1/oauth/token",
  "userinfo_endpoint": "https://auth.example.com/api/v1/oauth/userinfo",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "revocation_endpoint": "https://auth.example.com/api/v1/oauth/revoke",
  "introspection_endpoint": "https://auth.example.com/api/v1/oauth/introspect",
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "response_types_supported": ["code"],
  "grant_types_supported": [
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code",
    "urn:ietf:params:oauth:grant-type:token-exchange",
    "urn:openid:params:grant-type:ciba"
  ],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": ["S256"]
}
```

### Example

```bash
curl "https://auth.example.com/.well-known/openid-configuration"
```

---

## GET /.well-known/oauth-authorization-server

Returns the RFC 8414 Authorization Server Metadata document. Equivalent to the OIDC discovery document but omits OIDC-specific fields (`userinfo_endpoint`, `id_token_signing_alg_values_supported`). Includes additional endpoint URIs for advanced grant flows.

### Response

#### Success — 200 OK

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/api/v1/oauth/authorize",
  "token_endpoint": "https://auth.example.com/api/v1/oauth/token",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "revocation_endpoint": "https://auth.example.com/api/v1/oauth/revoke",
  "introspection_endpoint": "https://auth.example.com/api/v1/oauth/introspect",
  "pushed_authorization_request_endpoint": "https://auth.example.com/api/v1/oauth/par",
  "device_authorization_endpoint": "https://auth.example.com/api/v1/oauth/device_authorization",
  "registration_endpoint": "https://auth.example.com/api/v1/oauth/register",
  "end_session_endpoint": "https://auth.example.com/api/v1/oauth/end_session",
  "backchannel_authentication_endpoint": "https://auth.example.com/api/v1/oauth/ciba",
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "response_types_supported": ["code"],
  "grant_types_supported": [
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code",
    "urn:ietf:params:oauth:grant-type:token-exchange",
    "urn:openid:params:grant-type:ciba"
  ],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": ["S256"],
  "backchannel_token_delivery_modes_supported": ["poll"]
}
```

### Example

```bash
curl "https://auth.example.com/.well-known/oauth-authorization-server"
```

---

## GET /.well-known/jwks.json

Returns the JSON Web Key Set (RFC 7517) containing the RSA public key used to sign all JWTs issued by this server. Use this to verify the signature of access tokens and ID tokens.

The key ID (`kid`) is controlled by the `JWT_KEY_ID` environment variable and defaults to `maintainerd-auth-key-1`.

### Response

#### Success — 200 OK

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "maintainerd-auth-key-1",
      "alg": "RS256",
      "n": "<base64url-encoded modulus>",
      "e": "<base64url-encoded exponent>"
    }
  ]
}
```

#### Error Responses

| Status | Description |
|--------|-------------|
| 500 | The RSA signing key has not been initialized on the server. |

```json
{ "error": "keys not initialised" }
```

### Example

```bash
curl "https://auth.example.com/.well-known/jwks.json"
```
