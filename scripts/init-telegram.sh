#!/bin/bash

# Telegram MCP Session Initialization Script
# This script helps you securely initialize the Telegram session

echo "Telegram MCP Session Initialization"
echo "======================================"
echo ""

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo ".env file not found!"
    exit 1
fi

# Source environment variables
source .env

# Validate required environment variables
if [ -z "$TG_PHONE_NUMBER" ]; then
    echo "TG_PHONE_NUMBER not set in .env file"
    echo "Please edit .env and set your phone number (with country code, e.g., +1234567890)"
    exit 0
fi

if [ -z "$TG_APP_ID" ]; then
    echo "TG_APP_ID not set in .env file"
    exit 0
fi

if [ -z "$TG_API_HASH" ]; then
    echo "TG_API_HASH not set in .env file"
    exit 0
fi

if [ -z "$TG_SESSION_PATH" ]; then
    echo "TG_SESSION_PATH not set in .env file"
    exit 0
fi


echo "Phone Number: $TG_PHONE_NUMBER"
echo "App ID: $TG_APP_ID"
echo "API Hash: ${TG_API_HASH:0:8}..."
echo "Session Path: $TG_SESSION_PATH"
echo ""

echo "Starting Telegram MCP session initialization..."

npx -y @chaindead/telegram-mcp
    --phone "$TG_PHONE_NUMBER" \
    --app-id "$TG_APP_ID" \
    --api-hash "$TG_API_HASH" \
    --session "$TG_SESSION_PATH"