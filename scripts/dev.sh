#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
PUBLIC_API_BASE_URL="${PUBLIC_API_BASE_URL:-$SERVER_URL}"

server_pid=""
client_pid=""

cleanup() {
	if [ -n "$client_pid" ] && kill -0 "$client_pid" 2>/dev/null; then
		kill "$client_pid" 2>/dev/null || true
	fi
	if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
		kill "$server_pid" 2>/dev/null || true
	fi
}

trap cleanup INT TERM EXIT

echo "starting backend on $SERVER_URL"
"$ROOT_DIR/server/scripts/run-server.sh" &
server_pid=$!

echo "starting frontend with PUBLIC_API_BASE_URL=$PUBLIC_API_BASE_URL"
(
	cd "$ROOT_DIR/client"
	PUBLIC_API_BASE_URL="$PUBLIC_API_BASE_URL" npm run dev
) &
client_pid=$!

while :; do
	if ! kill -0 "$server_pid" 2>/dev/null; then
		wait "$server_pid"
		exit $?
	fi
	if ! kill -0 "$client_pid" 2>/dev/null; then
		wait "$client_pid"
		exit $?
	fi
	sleep 1
done
