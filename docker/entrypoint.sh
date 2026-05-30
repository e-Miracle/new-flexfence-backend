#!/bin/sh
set -e

export PORT="${PORT:-8080}"

su -s /bin/sh app -c "/app/api" &
API_PID=$!

trap 'kill -TERM "$API_PID" 2>/dev/null; wait "$API_PID" 2>/dev/null; exit 0' TERM INT

# Wait until the API accepts connections.
for _ in $(seq 1 30); do
  if wget -qO- "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

exec nginx -g 'daemon off;'
