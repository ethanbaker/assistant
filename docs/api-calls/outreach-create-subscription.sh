#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
OUTREACH_BASE="${OUTREACH_BASE:-${BASE_URL}/api/internal/outreach}"
CLIENT_API_KEY="${CLIENT_API_KEY:-}"
JOB_NAME="${JOB_NAME:-test-job-1773624839}"
PRIORITY="${PRIORITY:-1}"

if [ -z "$CLIENT_API_KEY" ]; then
  echo "Error: CLIENT_API_KEY environment variable is not set" >&2
  exit 1
fi

curl -sS -X POST "${OUTREACH_BASE}/subscriptions" \
  -H "Content-Type: application/json" \
  -H "X-Client-Key: ${CLIENT_API_KEY}" \
  -d "{\"job_name\":\"${JOB_NAME}\",\"priority\":${PRIORITY}}"
