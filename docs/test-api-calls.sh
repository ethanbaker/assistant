#!/bin/bash

# Create new session
: "
curl -X POST http://localhost:8080/api/agent/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: 2d78d012-29a7-4210-b427-3037e79dc33b" \
  -d '{
    "user_id": "user-1"
  }'
"

# Get session details
: "
curl -X GET http://localhost:8080/api/agent/sessions/419d669b-5324-451c-b8a4-5a208b3de469 \
  -H "X-API-KEY: 2d78d012-29a7-4210-b427-3037e79dc33b" \
"

# Add message to session
curl -X POST http://localhost:8080/api/agent/sessions/fd70dbd5-7d46-41cd-9fb9-2deb2b7c79ba/message \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: 2d78d012-29a7-4210-b427-3037e79dc33b" \
  -d '{
    "content": "Hello! How are you?"
  }'

# Delete session
: "
curl -X DELETE http://localhost:8080/api/agent/sessions/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-KEY: 2d78d012-29a7-4210-b427-3037e79dc33b"
  "