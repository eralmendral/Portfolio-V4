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
export DATABASE_URL="${DATABASE_URL:-postgres://portfolio:portfolio@localhost:$POSTGRES_HOST_PORT/portfolio?sslmode=disable}"

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

go run ./cmd/seed
