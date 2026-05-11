#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SERVER_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

cd "$SERVER_DIR"

if ! command -v docker >/dev/null 2>&1; then
	echo "docker is required to start the local postgres service" >&2
	exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
	echo "docker compose is required to start the local postgres service" >&2
	exit 1
fi

export POSTGRES_HOST_PORT="${POSTGRES_HOST_PORT:-5433}"

docker compose up -d postgres

echo "waiting for postgres..."
attempt=0
until docker compose exec -T postgres pg_isready -U portfolio -d portfolio >/dev/null 2>&1; do
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 30 ]; then
		echo "postgres did not become ready after 30 seconds" >&2
		exit 1
	fi
	sleep 1
done

export ADDR="${ADDR:-:8080}"
export DATABASE_URL="${DATABASE_URL:-postgres://portfolio:portfolio@localhost:$POSTGRES_HOST_PORT/portfolio?sslmode=disable}"
export ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
export ADMIN_PASSWORD="${ADMIN_PASSWORD:-change-me}"
export CLIENT_ORIGINS="${CLIENT_ORIGINS:-${CLIENT_ORIGIN:-http://localhost:4200,http://127.0.0.1:4200}}"
export JWT_SECRET="${JWT_SECRET:-development-secret-change-me}"
export JWT_ISSUER="${JWT_ISSUER:-portfolio-server}"
export TOKEN_TTL="${TOKEN_TTL:-24h}"
export UPLOAD_STORAGE="${UPLOAD_STORAGE:-local}"
export UPLOAD_DIR="${UPLOAD_DIR:-public/uploads/projects}"
export UPLOAD_BASE_URL="${UPLOAD_BASE_URL:-/uploads/projects}"
export MAX_UPLOAD_BYTES="${MAX_UPLOAD_BYTES:-314572800}"

case "$ADDR" in
	:*) SERVER_URL="http://localhost$ADDR" ;;
	*) SERVER_URL="http://$ADDR" ;;
esac

echo "server listening on $SERVER_URL"
echo "postgres available on localhost:$POSTGRES_HOST_PORT"
exec go run .
