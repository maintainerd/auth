# Pushed Authorization Requests (PAR)

Pushed Authorization Requests allow clients to push authorization parameters to the server before redirecting the user. The server returns a `request_uri` that the client uses in place of the full parameter set when initiating the authorization code flow. This prevents authorization parameters from being exposed in the browser URL bar or referrer headers.

**RFC reference:** [RFC 9126 — OAuth 2.0 Pushed Authorization Requests](https://www.rfc-editor.org/rfc/rfc9126)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| POST | `/api/v1/oauth/par` | Client credentials | 8081 |

---

## POST /api/v1/oauth/par

Accepts authorization request parameters, validates the client and redirect URI, stores the request server-side, and returns a `request_uri` with a 90-second TTL.

The `request_uri` is then passed to `GET /api/v1/oauth/authorize` as the `request_uri` parameter instead of the full set of authorization parameters.

### Authentication

The client must authenticate using one of the supported methods:

- **HTTP Basic auth** — `Authorization: Basic base64(client_id:client_secret)`
- **Body parameters** — `client_id` and `client_secret` in the POST body

Public clients (no secret) must still include `client_id`.

### Request

#### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Basic <base64(client_id:client_secret)>` (confidential clients) |

#### Body

`application/x-www-form-urlencoded`

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `response_type` | string | Yes | Must be `code` | The authorization response type. |
| `client_id` | string | Yes | Max 255 chars | The client identifier. Required when not using Basic auth. |
| `redirect_uri` | string | Yes | Max 2048 chars | Must exactly match a redirect URI registered for this client. |
| `code_challenge` | string | Yes | 43–128 chars | PKCE code challenge (base64url-encoded SHA-256 of the code verifier). |
| `code_challenge_method` | string | Yes | Must be `S256` | PKCE method. Only `S256` is supported. |
| `scope` | string | No | Max 1024 chars | Space-separated list of requested scopes (e.g. `openid profile email`). |
| `state` | string | No | Max 512 chars | Opaque value to maintain state between request and callback. |
| `nonce` | string | No | Max 512 chars | Value to associate a client session with the ID token. |
| `client_secret` | string | No | — | Client secret when using `client_secret_post` auth method. |

### Response

#### Success — 201 Created

```json
{
  "request_uri": "urn:ietf:params:oauth:request-uri:abc123randomtoken...",
  "expires_in": 90
}
```

| Field | Type | Description |
|-------|------|-------------|
| `request_uri` | string | Opaque URI prefixed with `urn:ietf:params:oauth:request-uri:`. Pass this as the `request_uri` parameter to the authorization endpoint. |
| `expires_in` | integer | Seconds until the `request_uri` expires. Fixed at `90`. |

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | A required parameter is missing, a value is out of range, or `redirect_uri` does not match any registered URI. |
| 400 | `invalid_request` | `response_type` is not `code` or `code_challenge_method` is not `S256`. |
| 401 | `invalid_client` | Client ID not found, is inactive, or client secret does not match. |
| 401 | `unauthorized_client` | Client is not authorized to use the `authorization_code` grant. |
| 500 | `server_error` | Unexpected internal error. |

### Usage with the Authorization Endpoint

After receiving a `request_uri`, redirect the user to the authorization endpoint using only the `client_id` and `request_uri`:

```
GET /api/v1/oauth/authorize
  ?client_id=<client_id>
  &request_uri=urn:ietf:params:oauth:request-uri:abc123randomtoken...
```

The server resolves the stored parameters and proceeds with the normal authorization code flow.

### Example

```bash
curl -X POST https://auth.example.com/api/v1/oauth/par \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "my_client_id:my_client_secret" \
  -d "response_type=code" \
  -d "client_id=my_client_id" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" \
  -d "code_challenge_method=S256" \
  -d "scope=openid+profile+email" \
  -d "state=xyzABC123" \
  -d "nonce=n-0S6_WzA2Mj"
```

**Response:**

```json
{
  "request_uri": "urn:ietf:params:oauth:request-uri:4ba59c4dc7f3b827d31c38e97b33db45",
  "expires_in": 90
}
```

Then redirect the user:

```
https://auth.example.com/api/v1/oauth/authorize?client_id=my_client_id&request_uri=urn%3Aietf%3Aparams%3Aoauth%3Arequest-uri%3A4ba59c4dc7f3b827d31c38e97b33db45
```
