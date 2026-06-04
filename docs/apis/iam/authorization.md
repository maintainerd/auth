# Service-to-Service Authorization

Maintainerd Auth distributes service identity policy bundles so services can
make local authorization decisions and refresh immediately when IAM assignments
change.

## Policy Bundle

`GET /api/v1/services/me/policy-bundle`

Authentication: service-account OAuth access token from `client_credentials`.
The token must carry `sub_type=service` and `svc=<service name>` or use the
service name as `sub`.

Headers:

- `Authorization: Bearer <service token>`
- `If-None-Match: "<bundle version>"` optional

Responses:

- `200 OK`: returns the current principal-scoped bundle.
- `304 Not Modified`: returned when `If-None-Match` matches the current bundle
  version.
- `401 Unauthorized`: token is missing or is not a service token.

Example:

```json
{
  "success": true,
  "data": {
    "service": "serviceA",
    "version": "v516dcc9a08d5",
    "policies": [
      {
        "version": "v1",
        "statement": [
          {
            "effect": "allow",
            "action": ["serviceB:invoke"],
            "resource": ["serviceB:grpc"]
          }
        ]
      }
    ],
    "generated_at": "2026-06-04T00:00:00Z"
  }
}
```

## Authorize

`POST /api/v1/authorize/`

Authentication: OAuth access token.

Request:

```json
{
  "principal": "serviceA",
  "action": "serviceB:invoke",
  "resource": "serviceB:grpc"
}
```

Response:

```json
{
  "success": true,
  "data": {
    "allowed": true,
    "reason": "matched allow"
  }
}
```

The evaluator uses AWS-style identity-policy semantics:

- Default deny.
- Explicit deny wins over allow.
- Exact match and wildcard match are supported for both action and resource.
- Unknown policy document versions are ignored.

## Integration Guide

1. Link an OAuth `client_credentials` client to its IAM service through
   `clients.service_id`.
2. The service obtains a token from `POST /api/v1/oauth/token`.
3. The service fetches `GET /api/v1/services/me/policy-bundle` at startup and
   stores the returned `ETag`.
4. The service evaluates outbound calls locally.
5. The service re-polls with `If-None-Match` every cache interval and after IAM
   webhook events.

IAM change events sent through the existing webhook dispatcher:

- `iam.policy.updated`
- `iam.service.policy.assigned`
- `iam.service.policy.removed`
