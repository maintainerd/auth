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

## 🧭 Roadmap

| Feature                               | Status          |
| ------------------------------------- | --------------- |
| Basic service registration            | ✅ Complete      |
| gRPC and REST API via grpc-gateway    | ✅ Complete      |
| JWT-based authentication & middleware | 🏗️ In Progress |
| OAuth2 grant flows support            | 🔜 Planned      |
| OIDC implementation                   | 🔜 Planned      |
| Provider support (Cognito, Auth0)     | 🔜 Planned      |
| SAML 2.0 federation support           | 🔜 Planned      |
| CSRF and XSS protection middleware    | 🔜 Planned      |
| Multi-tenant and RBAC capabilities    | 🔜 Planned      |

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
