#!/bin/sh
set -eu

# Runtime configuration injection.
#
# The image is built once and can be deployed against different API origins.
# Instead of baking the API base URLs into the JS bundle, we write a small
# config.js at container start from environment variables. index.html loads it
# before the app bundle, and config.ts reads window.__ENV__ (falling back to the
# build-time import.meta.env value when unset).
CONFIG_PATH="${CONFIG_PATH:-/usr/share/nginx/html/config.js}"

cat > "$CONFIG_PATH" <<EOF
window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: "${VITE_AUTH_API_BASE_URL:-}",
  VITE_AUTH_PUBLIC_API_BASE_URL: "${VITE_AUTH_PUBLIC_API_BASE_URL:-}",
  VITE_AUTH_IDENTITY_BASE_URL: "${VITE_AUTH_IDENTITY_BASE_URL:-}"
};
EOF

# Same-origin API proxy upstreams.
#
# The app calls /api/ and /public-api/ on its OWN origin so the session cookie
# stays first-party (auth cookies are __Host- prefixed and cannot cross hosts).
# nginx then forwards to the two backend planes. These are the upstream ORIGINS
# (scheme://host[:port]) — not the browser-facing URLs.
NGINX_CONF="${NGINX_CONF:-/etc/nginx/conf.d/default.conf}"
INTERNAL_API_ORIGIN="${AUTH_CONTROL_PLANE_ORIGIN:-http://maintainerd-auth:8080}"
PUBLIC_API_ORIGIN="${AUTH_DATA_PLANE_ORIGIN:-http://maintainerd-auth:8081}"

if [ -w "$NGINX_CONF" ]; then
  sed -i \
    -e "s#__INTERNAL_API_ORIGIN__#${INTERNAL_API_ORIGIN}#g" \
    -e "s#__PUBLIC_API_ORIGIN__#${PUBLIC_API_ORIGIN}#g" \
    "$NGINX_CONF"
fi

exec "$@"
