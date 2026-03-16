#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"
SESSION_ID="${SESSION_ID:-}"
MESSAGE_CONTENT="${MESSAGE_CONTENT:-What does my calendar look like tomorrow?}"

if [ -z "$API_KEY" ]; then
  echo "Error: API_KEY environment variable is not set" >&2
  exit 1
fi

curl -X POST "${BASE_URL}/api/agent/sessions/${SESSION_ID}/message" \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: ${API_KEY}" \
  -d "{\"content\":\"${MESSAGE_CONTENT}\"}"
