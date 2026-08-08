#!/bin/sh
set -eu

# All-in-one entrypoint.
#
# 1) Runtime config injection for each SPA. The image is built once; the API
#    base URLs are written into config.js at start from env. When left empty the
#    apps fall back to their same-origin defaults (/api/v1, /public-api/api/v1),
#    which is what the bundled nginx proxies — so the default "just works".
# 2) Substitute the nginx upstream origins to the backend in THIS container.
# 3) Hand off to the process supervisor (nginx + the Go backend).

# ── 1. per-SPA runtime config (derived from the backend's env) ────────────────
# There is ONE source of truth for URLs: the backend's env. The SPAs call their
# APIs SAME-ORIGIN (left empty -> the app's built-in /api/v1 + /public-api/api/v1
# defaults, which the bundled nginx proxies), so no API URL is configured here.
# The only cross-app link — the console linking out to the hosted identity UI —
# is taken from the backend's own APP_FRONTEND_IDENTITY_HOSTNAME.
cat > /srv/console/config.js <<EOF
window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: "",
  VITE_AUTH_PUBLIC_API_BASE_URL: "",
  VITE_AUTH_IDENTITY_BASE_URL: "${APP_FRONTEND_IDENTITY_HOSTNAME:-}"
};
EOF

cat > /srv/identity/config.js <<EOF
window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: ""
};
EOF

# ── 2. nginx upstreams -> the backend in this container ───────────────────────
NGINX_CONF="${NGINX_CONF:-/etc/nginx/nginx.conf}"
INTERNAL_API_ORIGIN="${AUTH_CONTROL_PLANE_ORIGIN:-http://127.0.0.1:8080}"
PUBLIC_API_ORIGIN="${AUTH_DATA_PLANE_ORIGIN:-http://127.0.0.1:8081}"
sed -i \
  -e "s#__INTERNAL_API_ORIGIN__#${INTERNAL_API_ORIGIN}#g" \
  -e "s#__PUBLIC_API_ORIGIN__#${PUBLIC_API_ORIGIN}#g" \
  "$NGINX_CONF"

# ── 3. supervisor ─────────────────────────────────────────────────────────────
exec "$@"
