<p align="center">
  <img width="150" height="150" alt="logo" src="https://github.com/user-attachments/assets/3e9eec3d-8312-4f5c-b8cb-14d309a17fda" />
</p>

<h1 align="center">Maintainerd Auth</h1>

The `auth` service is a modular authentication platform built for the **Maintainerd** ecosystem. It aims to be a complete, production-grade identity layer that supports both **built-in authentication** and **external identity providers** such as **Auth0**, **Cognito**, and **Google**.

> Designed from the ground up with **security**, **extensibility**, and **service-to-service communication** in mind using **gRPC**, **REST**, and **Go**.

---

## ✨ Features

- 🧾 **gRPC-first API** with optional REST gateway (via `grpc-gateway`)
- 🔐 **JWT-based authentication middleware** (in-progress)
- 🧱 Modular architecture with clean separation of concerns
- 🐘 PostgreSQL + GORM + Goose for schema management
- 🧪 Auto-seeding for essential service records
- ⚙️ Integration-ready for external providers like:
  - AWS Cognito
  - Auth0
  - Google Identity
- 🛡️ Future support for XSS/CSRF protection, OAuth2, OIDC, and SAML 2.0
- 📦 Shared protobuf contract via Git submodule

---

## 🚀 Getting Started

### ✅ Prerequisites

* Go 1.21+
* PostgreSQL
* [Goose](https://github.com/pressly/goose)
* `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`
* Make (optional but recommended)

---

## 📥 Clone the Repository

```bash
git clone --recurse-submodules https://github.com/maintainerd/auth.git
cd auth
```

If you forgot `--recurse-submodules`, run:

```bash
git submodule update --init --recursive
```

---

## ⚙️ Environment Configuration

Create a `.env` or set these env vars directly:

```env
APP_MODE=development
APP_VERSION=1.0.0
DB_URL=postgres://user:password@localhost:5432/auth_db?sslmode=disable
```

---

## 🧱 Building & Running

```bash
make run
```

Or manually:

```bash
go run cmd/server/main.go
```

---

## 🧭 Roadmap/Goal

Open Source Authentication Platform Security Checklist (SOC 2-Oriented)

### 1. 🔐 **Authentication Security**

* [ ] Enforce configurable password policy (length, complexity, etc.)
* [ ] Support for Multi-Factor Authentication (MFA)
* [ ] Passwords hashed with bcrypt/Argon2 (never SHA256/SHA512 alone)
* [ ] Secure login endpoints with HTTPS (documented requirement)
* [ ] Brute-force protection (rate limiting, CAPTCHA)
* [ ] Secure password reset (tokenized, time-limited flow)
* [ ] Session timeout & invalidation support

### 2. 🛡️ **Authorization**

* [ ] Support RBAC or ABAC for permission enforcement
* [ ] No role escalation or bypass through APIs
* [ ] Role/permission definitions externally configurable
* [ ] Audit trail capabilities for authz changes (optional but recommended)

### 3. 🔑 **Token & Session Management**

* [ ] JWT signed using RS256 or HS256 with strong keys
* [ ] Token expiration (short TTL) configurable
* [ ] Token revocation support (via DB or blacklist)
* [ ] Secure cookie support (`HttpOnly`, `Secure`, `SameSite`)
* [ ] Do not store sensitive data in tokens (only identifiers)

### 4. 🌐 **Identity Providers / Federation**

* [ ] Support for third-party OAuth2/OIDC login
* [ ] Validate all OIDC fields (issuer, audience, expiry)
* [ ] Secure storage of OAuth credentials (with recommendation to use vaults)
* [ ] Secure redirect URI validation (no wildcards)

### 5. 🔧 **Security by Design**

* [ ] Defense-in-depth: input validation, CSRF/XSS protections
* [ ] CSRF protection for all web-based flows
* [ ] Secure default configuration (secure on first run)
* [ ] Secrets and sensitive config values read from environment variables (not hardcoded)
* [ ] Full HTTPS requirement documented for deployment

### 6. 📦 **Dependency & Build Security**

* [ ] Minimal external dependencies; vetted packages only
* [ ] Regular dependency updates (use `go.mod` tidy/version pinning)
* [ ] No hardcoded secrets or API keys in the code
* [ ] Signed releases and checksums (optional, recommended)
* [ ] Build and release automation ensures reproducibility

### 7. 🔍 **Logging & Auditing Hooks**

* [ ] Emit structured logs for key auth events (login, logout, failed attempts)
* [ ] Do not log passwords, tokens, secrets
* [ ] Provide optional audit logging interface/hooks for consumers

### 8. 🛠️ **Configurability**

* [ ] Allow disabling of registration/login endpoints
* [ ] Provide toggles for MFA, email verification, password policy
* [ ] Allow integration with custom user storage (via interface or adapter)
* [ ] Support custom branding and UI theming (minimize user manipulation risk)

### 9. 🔒 **Secure API Practices**

* [ ] All sensitive APIs require authentication
* [ ] Support for API key-based access for service-to-service auth
* [ ] Input validation and sanitization for all APIs
* [ ] Rate limiting middleware / hooks (optional, documented)

### 10. 📄 **Documentation & Security Guidance**

* [ ] Clear deployment guide with security best practices (TLS, vaults, etc.)
* [ ] Sample .env / configuration files do not contain real credentials
* [ ] Highlight security requirements (HTTPS, secrets, logging)
* [ ] Explain how to integrate with secure email providers (SMTP, etc.)
* [ ] Document recommended secret rotation practices


For more detailed info [`doc/SECURITY.md`](doc/SECURITY.md).

---

## 🧑‍💻 Contributing

We welcome contributions!

1. Fork this repo
2. Create your feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes
4. Push to your branch (`git push origin feat/my-feature`)
5. Create a Pull Request

See [`docs/contributing.md`](docs/contributing.md) for guidelines.

---

## 📜 License

[MIT](LICENSE)

---

## 🔗 Related Projects

* [`grpc-contract`](https://github.com/xreyc/grpc-contract) – Shared proto definitions
* [`core`](https://github.com/maintainerd/core) – REST-to-gRPC API gateway

---

> Built with ❤️ by [@xreyc](https://github.com/xreyc) and the Maintainerd community.
