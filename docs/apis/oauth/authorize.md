# Authorization Endpoint

Initiates the OAuth 2.0 Authorization Code flow (RFC 6749 §4.1.1). The client sends the user's browser to this endpoint after the user has already authenticated. The server validates the client, redirect URI, and PKCE parameters, then either issues an authorization code immediately or returns a consent challenge that the frontend must resolve before the code can be issued.

**Authentication required.** This endpoint requires a valid JWT access token in the `Authorization` header. The user must be logged in before initiating authorization.

---

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/oauth/authorize` | Bearer JWT |

---

## GET /api/v1/oauth/authorize

### Authentication

Bearer JWT in the `Authorization` header. The token identifies the user on whose behalf authorization is being requested.

### Request

#### Headers

| Name | Required | Description |
|------|----------|-------------|
| `Authorization` | Yes | `Bearer <access_token>` |

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `response_type` | string | Yes | Must be `code`. |
| `client_id` | string | Yes | The client's OAuth identifier. Maximum 255 characters. |
| `redirect_uri` | string | Yes | Where to redirect after authorization. Must exactly match a URI registered for the client. Maximum 2048 characters. |
| `code_challenge` | string | Yes | PKCE code challenge derived from the `code_verifier`. Must be 43–128 characters. |
| `code_challenge_method` | string | Yes | Must be `S256`. |
| `scope` | string | No | Space-delimited list of requested scopes. Maximum 1024 characters. Supported values: `openid`, `profile`, `email`, `offline_access`. |
| `state` | string | No | Opaque value used to maintain state between the request and callback. Returned unchanged in the redirect. Maximum 512 characters. |
| `nonce` | string | No | Value to associate the client session with the ID token. Maximum 512 characters. |

### Response

There are two possible success responses depending on whether the client requires user consent.

#### Consent not required (or already granted) — 200 OK

The authorization code has been issued. The `redirect_uri` field contains the full callback URL including `code` and `state` query parameters. The client should redirect the user to this URI.

```json
{
  "success": true,
  "message": "Authorization successful",
  "data": {
    "redirect_uri": "https://app.example.com/callback?code=<authorization_code>&state=<state>"
  }
}
```

Authorization codes expire after **10 minutes**.

#### Consent required — 200 OK

The user has not yet consented to the requested scopes for this client. The `consent_challenge` is a UUID that must be passed to the [consent endpoint](./consent.md) to retrieve the consent screen details and submit the user's decision.

```json
{
  "success": true,
  "message": "Consent required",
  "data": {
    "consent_challenge": "a3f1b2c4-1234-5678-abcd-ef0123456789"
  }
}
```

Consent challenges expire after **10 minutes**.

#### Error Responses

OAuth errors are returned as JSON with `error` and `error_description` fields (RFC 6749 §5.2).

| Status | `error` | Description |
|--------|---------|-------------|
| 400 | `invalid_request` | A required parameter is missing, invalid, or `redirect_uri` does not match any registered URI for the client. |
| 400 | `invalid_request` | `client_id` is unknown or the client is inactive. |
| 401 | `unauthorized_client` | The client is not authorized to use the `authorization_code` grant type. |
| 400 | `unsupported_response_type` | `response_type` `code` is not enabled for this client. |
| 500 | `server_error` | An unexpected internal error occurred. |

Non-OAuth errors (e.g., missing authentication):

| Status | Description |
|--------|-------------|
| 401 | `{ "success": false, "message": "Authentication required" }` — no valid JWT was provided. |
| 422 | Validation error — one or more query parameters failed validation. |

```json
{ "error": "invalid_request", "error_description": "redirect_uri does not match any registered redirect URIs" }
```

### Example

```bash
curl "https://auth.example.com/api/v1/oauth/authorize\
?response_type=code\
&client_id=my-app\
&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback\
&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM\
&code_challenge_method=S256\
&scope=openid%20email\
&state=xyzABC123" \
  -H "Authorization: Bearer <access_token>"
```
