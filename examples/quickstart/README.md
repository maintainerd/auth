# Quick-start sample

Run the released `maintainerd-auth` image locally with PostgreSQL + Redis — no
repo clone, no files to write by hand.

### 1. Download the sample files

```bash
mkdir maintainerd-auth && cd maintainerd-auth
base=https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart
curl -O "$base/.env.example" -O "$base/docker-compose.yml" -O "$base/generate-secrets.sh"
cp .env.example .env
```

### 2. Generate your own keys (local, openssl)

```bash
chmod +x generate-secrets.sh && ./generate-secrets.sh
```

This writes a fresh RSA JWT keypair + encryption/HMAC secrets into `.env`.
Nothing leaves your machine.

### 3. Map the local hostnames (one time)

```bash
sudo tee -a /etc/hosts >/dev/null <<'EOF'
127.0.0.1 console.auth.maintainerd.local identity.auth.maintainerd.local console-api.auth.maintainerd.local identity-api.auth.maintainerd.local
EOF
```

### 4. Start it

```bash
docker compose up -d
```

### 5. First run 👉 open the console and set up

Go to **http://console.auth.maintainerd.local:3000** — the setup wizard creates
your first **tenant** and **admin** account.

| What | URL |
|------|-----|
| Admin console (setup starts here) | http://console.auth.maintainerd.local:3000 |
| Hosted login (end users) | http://identity.auth.maintainerd.local:3001 |
| OIDC discovery | http://identity-api.auth.maintainerd.local:8081/.well-known/openid-configuration |

> `http` on `.local` is for local trials only. For production use real HTTPS
> hostnames, set `COOKIE_SECURE=true` and `DB_SSLMODE=require`, and source
> secrets from a manager — see the
> [Environment Variables reference](../../docs/contributing/environment-variables.md).
