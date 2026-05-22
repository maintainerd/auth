# End Session (RP-Initiated Logout)

RP-Initiated Logout allows a Relying Party (RP) to request that the authorization server log out the current end user. The server revokes active refresh tokens for the user identified by the `id_token_hint` and optionally redirects the browser to a registered post-logout URI.

**Spec reference:** [OpenID Connect Session Management 1.0 §5 — RP-Initiated Logout](https://openid.net/specs/openid-connect-session-management-1_0.html#RPLogout)

---

## Endpoints

| Method | Path | Auth | Port |
|--------|------|------|------|
| GET | `/api/v1/oauth/end_session` | Bearer JWT (end user) | 8081 |
| POST | `/api/v1/oauth/end_session` | Bearer JWT (end user) | 8081 |

---

## GET /api/v1/oauth/end_session
## POST /api/v1/oauth/end_session

Both methods are equivalent. Use GET for browser-initiated logout redirects. Use POST when submitting from an application form.

If an `id_token_hint` is provided and resolves to a valid user, all of that user's refresh tokens are revoked. If a valid `post_logout_redirect_uri` is provided, the server redirects the browser there after logout. Otherwise, a JSON success response is returned.

The `id_token_hint` validation failure is silently ignored per the OIDC Session Management specification — logout still succeeds (tokens are revoked if any are found) and the redirect proceeds if a `post_logout_redirect_uri` was provided.

### Authentication

Requires an authenticated user session:

```
Authorization: Bearer <user_access_token>
```

### Request

For GET requests, parameters are passed as query string parameters. For POST requests, parameters are form-encoded in the request body.

#### Parameters

| Parameter | Type | Required | Constraints | Description |
|-----------|------|----------|-------------|-------------|
| `id_token_hint` | string | No | — | Previously issued ID token. Used to identify the user to log out and verify the client. |
| `client_id` | string | No | — | Client identifier. Used to scope the user lookup when `id_token_hint` is provided. |
| `post_logout_redirect_uri` | string | No | Max 2048 chars | URI to redirect the browser to after logout. Must be a valid absolute URI. |
| `state` | string | No | Max 512 chars | Opaque value appended to `post_logout_redirect_uri` as a `state` query parameter. |

#### GET Request Example

```
GET /api/v1/oauth/end_session
  ?id_token_hint=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
  &client_id=my_client_id
  &post_logout_redirect_uri=https://app.example.com/logged-out
  &state=xyz123
```

#### POST Request Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/x-www-form-urlencoded` |
| `Authorization` | `Bearer <user_access_token>` |

### Response

#### With a valid `post_logout_redirect_uri` — 302 Found

The browser is redirected to the `post_logout_redirect_uri`. If `state` was provided, it is appended:

```
Location: https://app.example.com/logged-out?state=xyz123
```

#### Without a `post_logout_redirect_uri` — 200 OK

```json
{
  "success": true,
  "message": "session ended"
}
```

#### Error Responses

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | `post_logout_redirect_uri` exceeds 2048 characters or `state` exceeds 512 characters. |
| 400 | — | POST request has malformed form data. Returns `{ "success": false, "message": "invalid form data" }`. |

### Examples

**GET — browser-initiated logout with redirect:**

```bash
curl -G https://auth.example.com/api/v1/oauth/end_session \
  -H "Authorization: Bearer <user_access_token>" \
  --data-urlencode "id_token_hint=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  --data-urlencode "client_id=my_client_id" \
  --data-urlencode "post_logout_redirect_uri=https://app.example.com/logged-out" \
  --data-urlencode "state=randomstate123"
```

Response: `302 Found` → `Location: https://app.example.com/logged-out?state=randomstate123`

**POST — application-initiated logout without redirect:**

```bash
curl -X POST https://auth.example.com/api/v1/oauth/end_session \
  -H "Authorization: Bearer <user_access_token>" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "id_token_hint=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d "client_id=my_client_id"
```

```json
{
  "success": true,
  "message": "session ended"
}
```
