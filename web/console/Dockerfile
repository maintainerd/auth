# Production Dockerfile for React + Vite app
# Multi-stage build for optimized production image

# Stage 1: Build the application
FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Stage 2: Production server
FROM nginx:alpine

# Committed, hardened nginx config (SPA fallback, gzip, immutable asset caching,
# CSP + security headers) replaces the previous inline echo config.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Static build output.
COPY --from=builder /app/dist /usr/share/nginx/html

# Runtime config injector: writes /usr/share/nginx/html/config.js from env on
# container start (see docker-entrypoint.sh).
COPY docker-entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh \
    && chown -R nginx:nginx /usr/share/nginx/html /var/cache/nginx /var/run \
    && touch /var/run/nginx.pid && chown nginx:nginx /var/run/nginx.pid

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/ || exit 1

USER nginx

ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
