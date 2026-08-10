#!/usr/bin/env bash
# =============================================================================
#  One-time local setup: generate this deployment's keys/secrets and a
#  self-signed TLS certificate for the .local hostnames.
#
#  Everything is generated LOCALLY with openssl — nothing leaves your machine
#  and no secret is committed. Run once after `cp .env.example .env`.
# =============================================================================
set -euo pipefail

ENV_FILE=.env
CERT_DIR=certs
command -v openssl >/dev/null || { echo "✗ openssl is required" >&2; exit 1; }
[ -f "$ENV_FILE" ] || { echo "✗ $ENV_FILE not found. Run:  cp .env.example .env" >&2; exit 1; }

# 1) App secrets: RSA JWT keypair + encryption + HMAC keys, appended to .env.
if grep -q '^JWT_PRIVATE_KEY=' "$ENV_FILE"; then
  echo "• Secrets already in $ENV_FILE — skipping."
else
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$tmp/jwt_private.pem" 2>/dev/null
  openssl pkey -in "$tmp/jwt_private.pem" -pubout -out "$tmp/jwt_public.pem" 2>/dev/null
  {
    echo ""
    echo "# --- generated locally by setup.sh — KEEP SECRET, do not commit ---"
    echo "APP_ENCRYPTION_KEY=base64:$(openssl rand -base64 32)"
    echo "HMAC_SECRET_KEY=base64:$(openssl rand -base64 32)"
    echo "JWT_PRIVATE_KEY=\"$(awk '{printf "%s\\n", $0}' "$tmp/jwt_private.pem")\""
    echo "JWT_PUBLIC_KEY=\"$(awk '{printf "%s\\n", $0}' "$tmp/jwt_public.pem")\""
  } >> "$ENV_FILE"
  echo "✓ Appended JWT keypair + encryption/HMAC secrets to $ENV_FILE"
fi

# 2) Self-signed TLS cert covering the four .local hostnames (browser will warn
#    the first time — it's your own local cert; accept it to continue).
mkdir -p "$CERT_DIR"
if [ -f "$CERT_DIR/auth.maintainerd.local.crt" ]; then
  echo "• TLS cert already in $CERT_DIR — skipping."
else
  openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
    -keyout "$CERT_DIR/auth.maintainerd.local.key" \
    -out    "$CERT_DIR/auth.maintainerd.local.crt" \
    -subj "/CN=auth.maintainerd.local" \
    -addext "subjectAltName=DNS:console.auth.maintainerd.local,DNS:identity.auth.maintainerd.local,DNS:console-api.auth.maintainerd.local,DNS:identity-api.auth.maintainerd.local" \
    2>/dev/null
  echo "✓ Generated self-signed TLS cert in $CERT_DIR/"
fi

echo ""
echo "Next:  add the hostnames to /etc/hosts, then  docker compose up -d"
