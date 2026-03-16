#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
OUTREACH_BASE="${OUTREACH_BASE:-${BASE_URL}/api/internal/outreach}"
ADMIN_API_KEY="${ADMIN_API_KEY:-}"
JOB_NAME="${JOB_NAME:-test-job-$(date +%s)}"
HANDLER="${HANDLER:-example-job}"
SCHEDULE_JSON="${SCHEDULE_JSON:-{\"type\":\"custom\",\"interval_ms\":3600000,\"offset_ms\":0}}"
PARAMETERS_JSON="${PARAMETERS_JSON:-{}}"

if [ -z "$ADMIN_API_KEY" ]; then
  echo "Error: ADMIN_API_KEY environment variable is not set" >&2
  exit 1
fi

curl -sS -X POST "${OUTREACH_BASE}/jobs" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: ${ADMIN_API_KEY}" \
  -d "{\"name\":\"${JOB_NAME}\",\"schedule\":${SCHEDULE_JSON},\"handler\":\"${HANDLER}\",\"parameters\":${PARAMETERS_JSON}}"
