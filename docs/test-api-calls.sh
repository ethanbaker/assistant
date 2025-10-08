#!/bin/bash

# Check if API_KEY is set
if [ -z "$API_KEY" ]; then
  echo "Error: API_KEY environment variable is not set"
  echo "Usage: API_KEY=your-api-key ./test-api-calls.sh"
  exit 1
fi

# Create new session
: "
curl -X POST http://localhost:8080/api/agent/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: $API_KEY" \
  -d '{
    "user_id": "user-1"
  }'
"

# Get session details
: "
curl -X GET http://localhost:8080/api/agent/sessions/419d669b-5324-451c-b8a4-5a208b3de469 \
  -H "X-API-KEY: $API_KEY" \
"

# Add message to session
: "
curl -X POST http://localhost:8080/api/agent/sessions/d522067d-746d-4835-9a43-2d013116aa85/message \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: $API_KEY" \
  -d '{
    "content": "Hello! How are you?"
  }'
"

# Delete session
: "
curl -X DELETE http://localhost:8080/api/agent/sessions/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-KEY: $API_KEY"
  "