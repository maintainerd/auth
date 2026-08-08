# Production Edge & Security Hardening

How to put the all-in-one `maintainerd-auth` image behind a production edge
(reverse proxy / ingress / load balancer) without the two classes of failure
that a default edge configuration causes. Both were caught by running the
release image locally (the `auth-release` parity profile) and both are
well-known, industry-standard concerns — this doc captures the accepted fixes so
they never reach production.

> TL;DR — three things your platform must get right:
> 1. **Proxy header buffers ≥ 16k** at the edge (auth sets JWTs as `Set-Cookie`; the default 4k/8k buffer overflows → 502 on login).
> 2. **Forwarded headers** (`X-Forwarded-Proto`/`-For`) trusted from your edge only (`TRUSTED_PROXY_CIDRS`, never `TRUST_ALL_PROXIES` in prod).
> 3. **Non-root readable secrets** — the image runs as uid 65532; mounted secrets/certs must be readable by it (`defaultMode: 0440`, `fsGroup`), or use a secret manager.

---

## 1. Reverse proxy / edge requirements

The image serves **plain HTTP** on its ports and expects TLS to be terminated at
the edge (the standard pattern). The edge must be configured for an auth service:

### 1a. Proxy header buffers (REQUIRED — prevents the login 502)

On a successful `POST /api/v1/login` (and `/refresh-token`), the app returns the
access + refresh JWTs as `httpOnly` `Set-Cookie` **response headers**. RS256
signatures make these multi-kilobyte, which overflows the reverse proxy's default
response-header buffer (nginx: 4k, or one 8k buffer). The proxy then can't read
the upstream headers and returns **502 Bad Gateway** with
`upstream sent too big header while reading response header from upstream` —
even though the app answered `200`.

This is a common, documented issue for any JWT-in-cookie auth service. Fix by
raising the proxy header buffer to **16k** (bump to 32k/64k if you carry very
large tokens or many claims):

**ingress-nginx** (per-Ingress annotation — the official recommendation):
```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-buffer-size: "16k"
    nginx.ingress.kubernetes.io/proxy-buffers-number: "4"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"   # match the app's 10MB request cap
```

**Plain nginx edge** (http/server context):
```nginx
proxy_buffer_size       16k;
proxy_buffers         8 16k;
proxy_busy_buffers_size 16k;
```

**Envoy / HAProxy / cloud L7 LBs** have their own header-size limits
(`max_request_headers_kb`, `tune.http.maxhdr`, etc.) — raise the **response**
header limit the same way.

> Root-cause option (see §3): shrinking the tokens (ES256 signing, minimal
> claims) reduces cookie size and makes the default buffers sufficient. Sizing
> the buffer is the operational fix; shrinking the token is the design fix. Do
> both if you can.

### 1b. Forwarded headers (REQUIRED for correct cookies, issuer, and rate limits)

The app derives the request scheme from `X-Forwarded-Proto`. Because it serves
plain HTTP behind a TLS edge, the edge **must** send:

```
X-Forwarded-Proto  $scheme      # https — or Secure/__Host- cookies won't be set and the OIDC issuer scheme is wrong
X-Forwarded-For    <client ip>  # real client IP for per-IP rate limits, lockout, IP-restriction rules
Host               <original>   # tenant resolution keys on the request host
```

ingress-nginx sets these when `use-forwarded-headers: "true"` is enabled on the
controller. For a raw nginx edge, forward them explicitly (see the sample below).

**Trust scope:** set `TRUSTED_PROXY_CIDRS` to your edge's address range so the app
only believes forwarded headers from the edge. **Do not** set
`TRUST_ALL_PROXIES=true` in production — it lets any caller spoof
`X-Forwarded-For` and bypass every per-IP control. (`TRUST_ALL_PROXIES` is a
local-dev-only convenience.)

### 1c. Which ports the edge exposes (public vs private split)

The image binds four HTTP ports; expose them on the matching edge, never all on
the public one (see the port/security model in the README):

| Port | Surface | Edge |
|------|---------|------|
| `:8081` | public data API (OAuth2/OIDC, `/.well-known`) | **public** ingress/LB |
| `:3001` | identity SPA (login) | **public** ingress/LB |
| `:8080` | control/management API | **private** (VPN / internal ingress + NetworkPolicy) |
| `:3000` | console SPA (admin) | **private** (VPN / internal ingress + NetworkPolicy) |
| `:8082` | metrics/management | **private** (scrape internally) |
| `:50051` | gRPC (mTLS) | private / service mesh only |

### 1d. Timeouts & keep-alive

Login does bcrypt + DB + rate-limit work; keep the edge's `proxy_read_timeout`
at the default 60s (the app's own request timeout is 60s). No WebSocket upgrade
is needed by the release image (the SPAs are static; HMR is dev-only).

---

## 2. Container security context (non-root secret/cert readability)

The image runs as a **non-root user, uid/gid 65532** (`m9d`) — correct hardening.
The consequence: anything mounted into the container that the app must read
(TLS/gRPC certs, key files, a `.env`) **must be readable by uid 65532**. If you
bind-mount host files owned by another user with `0600`/`0700`, the container
can't read them and crash-loops (`permission denied`). This is the single most
common cause of a non-root container failing to start.

### 2a. Prefer a secret manager or env over mounted files

The app has a first-class secret abstraction — set `SECRET_PROVIDER` to `aws`,
`gcp`, `azure`, or `vault` and let the platform inject `JWT_PRIVATE_KEY`,
`JWT_PUBLIC_KEY`, `APP_ENCRYPTION_KEY`, `HMAC_SECRET_KEY` at runtime. Env vars and
managed secrets sidestep file-permission problems entirely and are the 12-factor
norm. Use file mounts only for material that must be a file (e.g. gRPC mTLS
certs).

### 2b. Kubernetes — hardened securityContext + readable Secret volumes

`fsGroup` fixes ownership on writable/PVC volumes, but **Secret and ConfigMap
volumes are read-only projected volumes** — you make their files readable with
`defaultMode`, not `fsGroup`. Use both:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    runAsGroup: 65532
    fsGroup: 65532                     # group-owns writable volumes for uid 65532
    seccompProfile: { type: RuntimeDefault }
  containers:
    - name: auth
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities: { drop: ["ALL"] }
      volumeMounts:
        - { name: tmp,   mountPath: /tmp }               # readOnlyRootFilesystem needs a writable /tmp
        - { name: certs, mountPath: /etc/maintainerd/grpc-certs, readOnly: true }
  volumes:
    - name: tmp
      emptyDir: {}
    - name: certs
      secret:
        secretName: maintainerd-grpc-certs
        defaultMode: 0440              # owner+group read; required for the non-root uid to read
```

### 2c. Docker / Compose

Keep the image's non-root default (don't add `user: root`). Any bind-mounted
secret/cert must be readable by uid 65532 — e.g. `chmod 0644` the cert files and
`0755` their directory, or (better) pass secrets via `--env-file` / Docker
secrets rather than bind mounts.

---

## 3. Token & cookie sizing (OWASP-aligned, and why the buffers matter)

maintainerd already follows the browser-security consensus (OWASP Session
Management, IETF *OAuth 2.0 for Browser-Based Apps*): tokens are delivered as
**`httpOnly; Secure; SameSite=Strict` cookies**, never exposed to JavaScript /
`localStorage`, with CSRF protection on cookie-authenticated state-changing
endpoints. That is the recommended model — keep it.

The tradeoff of that model is **cookie/header size**: browsers cap a cookie at
~4KB and total request headers around 8KB, and reverse proxies buffer response
headers (§1a). A large JWT eats into both. To stay comfortably within limits and
reduce edge-buffer pressure:

- **Prefer ES256 (or EdDSA) signing over RS256** for new deployments — an ES256
  signature is ~64 bytes vs ~256–512 bytes for RS256/RSA-4096, cutting token
  size substantially. (The app validates `RS256` today; ES-family is the smaller
  option when you choose keys.)
- **Keep claims minimal** — don't pack large role/permission lists into the
  access token; resolve them server-side or via the userinfo endpoint.
- **If tokens ever approach the 4KB cookie limit**, move to opaque/reference
  session cookies (server-side session, tokens stay in Redis) or a
  Backend-for-Frontend — the OWASP-preferred pattern for large-token cases.

Even with small tokens, keep the §1a buffer bump: it's cheap insurance and the
industry default recommendation for any cookie-based auth service.

---

## 4. Reference: hardened standalone nginx edge

For a non-Kubernetes deployment, a minimal hardened edge in front of the image:

```nginx
# Public edge — identity (:3001) + data API (:8081). Terminate TLS here.
map $http_upgrade $connection_upgrade { default upgrade; "" close; }

# REQUIRED: auth sets JWTs as Set-Cookie; default buffers overflow -> 502 on login.
proxy_buffer_size       16k;
proxy_buffers         8 16k;
proxy_busy_buffers_size 16k;

server {
  listen 443 ssl;
  server_name identity.example.com;         # the OIDC issuer + login UI
  ssl_certificate     /etc/tls/tls.crt;
  ssl_certificate_key /etc/tls/tls.key;

  location / {                              # identity SPA + its /api + /.well-known
    proxy_pass http://auth-upstream:3001;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;   # REQUIRED for Secure cookies + issuer scheme
  }
}
# The console (:3000) + control API (:8080) go behind a SEPARATE, private edge
# with the same proxy_* settings — never on the public listener.
```

Set `TRUSTED_PROXY_CIDRS` to this edge's network so forwarded headers are trusted
only from it.

---

## Sources

- [ingress-nginx — Accommodation for JWT (proxy-buffer-size)](https://kubernetes.github.io/ingress-nginx/examples/customization/jwt/)
- [Fixing nginx "upstream sent too big header" in an ingress controller](https://andrewlock.net/fixing-nginx-upstream-sent-too-big-header-error-when-running-an-ingress-controller-in-kubernetes/)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [IETF — OAuth 2.0 for Browser-Based Applications (BFF / token handling)](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps)
- [Curity — Using OAuth and Cookies in Browser-Based Apps](https://curity.io/resources/learn/oauth-cookie-best-practices/)
- [Kubernetes — Configure a Security Context (runAsNonRoot, fsGroup)](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
- [Wiz — Kubernetes Security Context best practices](https://www.wiz.io/academy/container-security/kubernetes-security-context-best-practices)
