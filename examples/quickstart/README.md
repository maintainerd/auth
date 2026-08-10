# Quick-start (local, released image)

Run `maintainerd-auth` locally behind nginx with clean HTTPS hostnames (no ports),
using the published image + PostgreSQL + Redis. For local testing — in production
you front it with your own TLS and real hostnames.

### 1. Download these files into one empty folder

- [`docker-compose.yml`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/docker-compose.yml)
- [`.env.example`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/.env.example)
- [`nginx.conf`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/nginx.conf)
- [`setup.sh`](https://raw.githubusercontent.com/maintainerd/maintainerd-auth/main/examples/quickstart/setup.sh)

### 2. Run

```bash
cp .env.example .env
chmod +x setup.sh && ./setup.sh          # generates your keys + a local TLS cert

sudo tee -a /etc/hosts >/dev/null <<'EOF'
127.0.0.1 console.auth.maintainerd.local identity.auth.maintainerd.local console-api.auth.maintainerd.local identity-api.auth.maintainerd.local
EOF

docker compose up -d
```

### 3. Open the setup wizard

**https://console.auth.maintainerd.local/setup/tenant**

Create your first tenant and admin. (Your browser will warn once about the
self-signed local cert — accept it.)

| | URL |
|---|---|
| Setup wizard (start here) | https://console.auth.maintainerd.local/setup/tenant |
| Admin console | https://console.auth.maintainerd.local |
| Hosted login | https://identity.auth.maintainerd.local |
| OIDC discovery | https://identity-api.auth.maintainerd.local/.well-known/openid-configuration |

> Local testing only. For production, use your own TLS + real hostnames, keep
> `DB_SSLMODE=require`, and source secrets from a manager — see the
> [Environment Variables reference](../../docs/contributing/environment-variables.md).
