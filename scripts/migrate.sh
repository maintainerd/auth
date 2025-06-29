#!/bin/bash
set -a
source .env
set +a

goose -dir db/migration postgres \
"postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE" "$@"
