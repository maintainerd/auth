# Release Pre-Flight & Sign-Off (v0.1.0)

Run this checklist immediately before the manual `git tag` + Docker image build/publish. It covers checklist items **K3** (clean-DB migration proof), **K4** (end-to-end smoke), and **K7** (security & scale sign-off). Automated gates (K1 build/test/lint, K2 frontend build, K5 image build, K6 secret scan) run in CI; this document is the human gate.

## K3 — Migrations apply cleanly from an empty database

```bash
# Throwaway Postgres, fresh volume:
docker run --rm -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=auth -p 5433:5432 postgres:16-alpine &
# Point the app at it and start once; migrations run at boot (advisory-locked, ordered).
DB_HOST=localhost DB_PORT=5433 DB_NAME=auth DB_USER=postgres DB_PASSWORD=pw \
  <other required env> go run ./cmd/server &
# Confirm readiness (readiness gates on DB/Redis/JWKS), then stop.
curl -fsS http://localhost:8080/readyz && echo "MIGRATIONS + READY OK"
```

Pass criteria: `/readyz` returns 200 from a fresh, empty database — proving every migration (including the partitioned `auth_events` in 048 and the in-place-edited create migrations) applies from zero with no manual surgery.

## K4 — End-to-end smoke (through nginx, not raw ports)

Exercise each path via the local hosts (`console.auth.maintainerd.local`, `identity.auth.maintainerd.local`):

- [ ] Identity: password login, registration (normal + `screen_hint=signup` + invite), MFA enroll then login, OAuth `authorize → token` (with PKCE), logout, lockout/429 screens.
- [ ] Console: login (cookie-based), and one create/read/update/delete on each admin domain (tenants, clients, APIs, roles, permissions, users, identity providers, registration flows, invites, webhooks, audit events).
- [ ] Admin operability: IdP test-connection, webhook delivery list + replay, audit export, client↔IdP connection edit.
- [ ] Branding renders per-client; `/openapi.json` served.

## K7 — Security & scale sign-off

Attach evidence for each before tagging:

- [ ] **CI green:** build, `go test -race`, golangci-lint/vet/staticcheck/gosec, `govulncheck`, coverage floor, license scan, Gitleaks (full history), CodeQL/Semgrep/Snyk.
- [ ] **Frontend green:** both apps `tsc -b` + `vite build` + `npm audit --audit-level=high`.
- [ ] **J4 load run:** k6 summary + `EXPLAIN (ANALYZE)` plans from `docs/documentations/observability/load-testing.md` — p95 targets met, no seq scans on hot paths at 1M+ rows.
- [ ] **J5 manual abuse pass:** attempted enumeration (register generic message confirmed), brute-force → lockout/429, IDOR (cross-tenant UUID → 404), open-redirect (return_to/callback rejected), CSRF on cookie endpoints, refresh-token reuse → family revoked.
- [ ] **J6 OIDC conformance:** `/.well-known/openid-configuration` matches actual capabilities; authorize/token/PKCE/refresh/revoke/introspect/userinfo/JWKS return spec-compliant responses (see `docs/documentations/oauth2/`).

## K8 — Edge & runtime hardening

The production edge and container runtime must be configured for an auth service.
See [Production Edge & Security Hardening](edge-and-security-hardening.md).

- [ ] **Proxy header buffers ≥ 16k** at the edge (ingress `proxy-buffer-size: "16k"` or nginx `proxy_buffer_size 16k`). Verify a real `POST /api/v1/login` **through the production-shaped edge** returns `200` with `Set-Cookie` — not `502 upstream sent too big header`.
- [ ] **Non-root secrets readable:** container runs as uid 65532 and boots clean; mounted Secret/cert volumes use `defaultMode: 0440` (or secrets come from env / a secret manager). No `permission denied` on gRPC/TLS material.
- [ ] **Forwarded headers trusted from the edge only:** `X-Forwarded-Proto`/`-For` forwarded by the edge; `TRUSTED_PROXY_CIDRS` set to the edge range; `TRUST_ALL_PROXIES` is **not** set in prod.
- [ ] **Port exposure split:** only `:8081` (data API) and `:3001` (identity) are internet-facing; `:8080` (control), `:3000` (console), `:8082` (metrics), `:50051` (gRPC) are private/VPN-only.
- [ ] **Hardened securityContext** applied (`runAsNonRoot`, `readOnlyRootFilesystem` + `/tmp` emptyDir, drop `ALL` caps, `seccompProfile: RuntimeDefault`), `DB_SSLMODE=require`, `REDIS_TLS=true`.

Sign-off: _name / date / commit SHA_ recorded in the GitHub release notes. Only tag `v0.1.0` once every box above is checked.
