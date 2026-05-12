#!/usr/bin/env sh
set -eu

DEPLOY_SCOPE="${1:-auto}"
APP_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
FRONTEND_ROOT="${FRONTEND_ROOT:-/var/www/portfolio-v4}"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-portfolio_v4}"
COMPOSE_FILE="${COMPOSE_FILE:-compose.prod.yaml}"
PRODUCTION_REF="${PRODUCTION_REF:-origin/main}"

case "$DEPLOY_SCOPE" in
auto | app | server | all)
	;;
*)
	echo "invalid deploy scope: $DEPLOY_SCOPE" >&2
	exit 1
	;;
esac

cd "$APP_ROOT"
git fetch origin main
git reset --hard "$PRODUCTION_REF"

deploy_app=false
deploy_server=false

case "$DEPLOY_SCOPE" in
auto | all)
	deploy_app=true
	deploy_server=true
	;;
app)
	deploy_app=true
	;;
server)
	deploy_server=true
	;;
esac

if [ "$deploy_app" = "true" ]; then
	cd "$APP_ROOT/app"
	if [ -f "$HOME/.config/portfolio-v4/frontend.env" ]; then
		set -a
		# shellcheck disable=SC1091
		. "$HOME/.config/portfolio-v4/frontend.env"
		set +a
	fi

	bun install --frozen-lockfile
	bun run build

	mkdir -p "$FRONTEND_ROOT"
	rsync -a --delete dist/app/browser/ "$FRONTEND_ROOT/"
fi

cd "$APP_ROOT/server"

if [ "$deploy_server" = "true" ]; then
	docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" up -d --build
fi

if [ -f .env ]; then
	set -a
	# shellcheck disable=SC1091
	. ./.env
	set +a
fi

if [ -z "${POSTGRES_PASSWORD:-}" ]; then
	echo "POSTGRES_PASSWORD is required in server/.env to seed production data" >&2
	exit 1
fi

docker build --target build -t portfolio-v4-seed-builder .
echo "refreshing production data from the current committed seeder..."
docker run --rm \
	--network "${COMPOSE_PROJECT}_default" \
	-e DATABASE_URL="postgres://portfolio:${POSTGRES_PASSWORD}@postgres:5432/portfolio?sslmode=disable" \
	portfolio-v4-seed-builder \
	go run ./cmd/seed

if command -v curl >/dev/null 2>&1; then
	curl -fsS http://127.0.0.1:18080/healthz >/dev/null || true
fi
