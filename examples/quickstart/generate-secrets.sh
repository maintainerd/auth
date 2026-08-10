#!/usr/bin/env bash
# =============================================================================
#  Generate this deployment's own keys/secrets and append them to ./.env.
#
#  Everything is generated LOCALLY with openssl — nothing leaves your machine,
#  and no secret is ever committed. Run once after `cp .env.example .env`.
#
#  Produces: an RSA JWT signing keypair, a 32-byte AES-256 encryption key, and
#  an HMAC key (the latter two as `base64:`-prefixed values).
# =============================================================================
set -euo pipefail

ENV_FILE="${1:-.env}"

if [ ! -f "$ENV_FILE" ]; then
  echo "✗ $ENV_FILE not found. Run:  cp .env.example .env" >&2
  exit 1
fi

if grep -q '^JWT_PRIVATE_KEY=' "$ENV_FILE"; then
  echo "✓ Secrets already present in $ENV_FILE — nothing to do."
  exit 0
fi

command -v openssl >/dev/null || { echo "✗ openssl is required" >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$tmp/jwt_private.pem" 2>/dev/null
openssl pkey -in "$tmp/jwt_private.pem" -pubout -out "$tmp/jwt_public.pem" 2>/dev/null

{
  echo ""
  echo "# --- generated locally by generate-secrets.sh — KEEP SECRET, do not commit ---"
  echo "APP_ENCRYPTION_KEY=base64:$(openssl rand -base64 32)"
  echo "HMAC_SECRET_KEY=base64:$(openssl rand -base64 32)"
  echo "JWT_PRIVATE_KEY=\"$(awk '{printf "%s\\n", $0}' "$tmp/jwt_private.pem")\""
  echo "JWT_PUBLIC_KEY=\"$(awk '{printf "%s\\n", $0}' "$tmp/jwt_public.pem")\""
} >> "$ENV_FILE"

echo "✓ Appended a fresh JWT keypair + encryption/HMAC secrets to $ENV_FILE"
echo "  Next:  docker compose up -d"
