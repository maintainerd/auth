# 🛠️ Setup Guide – `md-auth` Authentication Service

Welcome to the official setup guide for `md-auth`, an open-source authentication service by [Maintainerd](https://github.com/maintainerd). This guide helps you get the project running locally using Docker with zero manual setup for migrations or seeders — everything runs automatically on startup.

---

## 📋 Prerequisites

Make sure the following tools are installed on your machine:

* [Docker](https://www.docker.com/get-started)
* [Docker Compose](https://docs.docker.com/compose/)
* [Git](https://git-scm.com/)
* [Make](https://www.gnu.org/software/make/) (optional)
* [Go 1.21+](https://golang.org/dl/) (optional for local development outside Docker)

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/maintainerd/auth.git md-auth
cd md-auth
```

---

### 2. Copy the Example Environment File

```bash
cp .env.example .env
```

Edit `.env` as needed (e.g., `ADMIN_USERNAME`, `DB_PASSWORD`, etc.).

---

### 3. (Optional) Initialize Git Submodules

If the project uses Git submodules (like `contract/`):

```bash
git submodule update --init --recursive
```

---

### 4. Set up a Local Domain (Optional but Recommended)

To access the service via `http://auth.maintainerd.local`, add this to your **hosts file**:

#### On Linux/macOS:

```bash
sudo nano /etc/hosts
```

#### On Windows:

```
C:\Windows\System32\drivers\etc\hosts
```

Add the following line:

```
127.0.0.1 auth.maintainerd.local
```

You can now access your service via:
👉 `http://auth.maintainerd.local`

Ensure your `.env` also reflects this:

```env
APP_HOSTNAME=http://auth.maintainerd.local
```

---

### 5. Start the Project

Run the services using Docker Compose:

```bash
docker compose up --build
```

This will:

* Build and start the `md-auth` container with live reload via `air`
* Spin up `postgres-db` and `redis-db`
* Automatically apply migrations and seed the database

---

## ⚙️ Services Overview

| Service       | Description              | URL / Hostname                                             |
| ------------- | ------------------------ | ---------------------------------------------------------- |
| `md-auth`     | Go-based auth API        | `http://localhost:8080` or `http://auth.maintainerd.local` |
| `postgres-db` | PostgreSQL database      | `localhost:5433`                                           |
| `redis-db`    | Redis with password auth | `localhost:6379`                                           |

---

## 🔐 Configuration Summary

The app reads values from `.env`. Key sections:

### Database

```env
DB_HOST=postgres-db
DB_PORT=5432
DB_USER=devuser
DB_PASSWORD=Pass123
DB_NAME=maintainerd
```

### Redis

```env
REDIS_CONNECTION_STRING=redis://:Pass123@redis-db:6379
REDIS_PASSWORD=Pass123
```

### App

```env
APP_VERSION=v1
APP_MODE=micro
APP_HOSTNAME=http://auth.maintainerd.local
```

### JWT

```env
JWT_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----...
JWT_PUBLIC_KEY=-----BEGIN PUBLIC KEY-----...
JWT_ISSUER=https://account.medlexer.com
```

---

## 🔁 Live Reload

The app uses [`air`](https://github.com/air-verse/air) inside the container. Any code changes trigger a rebuild and restart automatically.

---

## 🧪 Run Tests

Inside the container:

```bash
docker exec -it md-auth go test ./...
```

Or locally (if you have Go installed):

```bash
go test ./...
```

---

## 🧼 Cleanup

To stop the containers:

```bash
docker compose down
```

To rebuild everything:

```bash
docker compose build --no-cache
```

---

## 🤝 Contributing

We welcome contributions!

1. Fork the repo
2. Create a feature branch
3. Submit a pull request
4. Follow any guidelines in `CONTRIBUTING.md`

---

## 📬 Need Help?

* [Open an issue](https://github.com/maintainerd/auth/issues)
* [Start a discussion](https://github.com/maintainerd/auth/discussions)
* Contact the Maintainerd team via GitHub