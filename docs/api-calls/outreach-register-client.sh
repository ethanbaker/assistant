#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
OUTREACH_BASE="${OUTREACH_BASE:-${BASE_URL}/api/internal/outreach}"
ADMIN_API_KEY="${ADMIN_API_KEY:-}"
CLIENT_NAME="${CLIENT_NAME:-test-client-$(date +%s)}"
WEBHOOK_URL="${WEBHOOK_URL:-https://example.com/webhooks/outreach}"

if [ -z "$ADMIN_API_KEY" ]; then
  echo "Error: ADMIN_API_KEY environment variable is not set" >&2
  exit 1
fi

curl -sS -X POST "${OUTREACH_BASE}/clients" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: ${ADMIN_API_KEY}" \
  -d "{\"name\":\"${CLIENT_NAME}\",\"webhook_url\":\"${WEBHOOK_URL}\"}"
