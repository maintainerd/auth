#!/bin/sh
set -eu

# Runtime configuration injection.
#
# The image is built once and can be deployed against different API origins.
# Instead of baking the API base URL into the JS bundle, we write a small
# config.js at container start from environment variables. index.html loads it
# before the app bundle, and config.ts reads window.__ENV__ (falling back to the
# build-time import.meta.env value when unset).
CONFIG_PATH="${CONFIG_PATH:-/usr/share/nginx/html/config.js}"

cat > "$CONFIG_PATH" <<EOF
window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: "${VITE_AUTH_API_BASE_URL:-}"
};
EOF

# Same-origin API proxy upstream.
#
# The app calls /api/ on its OWN origin so the session cookie stays first-party
# (auth cookies are __Host- prefixed and cannot cross hosts). nginx forwards to
# the public data plane. This is the upstream ORIGIN, not a browser-facing URL.
NGINX_CONF="${NGINX_CONF:-/etc/nginx/conf.d/default.conf}"
PUBLIC_API_ORIGIN="${AUTH_DATA_PLANE_ORIGIN:-http://maintainerd-auth:8081}"
if [ -w "$NGINX_CONF" ]; then
  sed -i "s#__PUBLIC_API_ORIGIN__#${PUBLIC_API_ORIGIN}#g" "$NGINX_CONF"
fi

exec "$@"
