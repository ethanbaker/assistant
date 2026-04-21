#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"
USER_ID="${USER_ID:-user-1}"

if [ -z "$API_KEY" ]; then
  echo "Error: API_KEY environment variable is not set" >&2
  exit 1
fi

curl -X POST "${BASE_URL}/api/agent/sessions" \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: ${API_KEY}" \
  -d "{\"user_id\":\"${USER_ID}\"}"
