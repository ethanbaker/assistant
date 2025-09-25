#!/bin/bash

set -e

# Script to create a new outreach implementation with a generated secret
# Usage: ./new-outreach-implementation.sh <client_id> [options]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
ENV_FILE=""
CLIENT_ID=""
CALLBACK_URL=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print usage
usage() {
    echo "Usage: $0 <client_id> [options]"
    echo ""
    echo "Arguments:"
    echo "  client_id           Unique identifier for the implementation"
    echo ""
    echo "Options:"
    echo "  --env=FILE         Load environment from FILE (default: .env)"
    echo "  --callback=URL     Callback URL for the implementation"
    echo "  --help             Show this help message"
    echo ""
    echo "Example:"
    echo "  $0 discord_bot --callback=http://localhost:3000/webhook"
    echo "  $0 slack_notifier --env=production.env --callback=https://api.example.com/slack/hook"
}

# Function to generate a secure random secret
generate_secret() {
    # Generate a 32-character random string using OpenSSL
    openssl rand -hex 16 2>/dev/null || {
        # Fallback using /dev/urandom if OpenSSL is not available
        cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
    }
}

# Function to load environment file
load_env() {
    local env_file="$1"
    if [[ -f "$env_file" ]]; then
        echo -e "${GREEN}Loading environment from: $env_file${NC}"
        echo ""

        # Export variables from env file
        export $(grep -v '^#' "$env_file" | xargs)
    else
        echo -e "${RED}Error: Environment file not found: $env_file${NC}"
        exit 1
    fi
}

# Function to validate required environment variables
validate_env() {
    local missing_vars=()
    
    if [[ -z "$MYSQL_HOST" ]]; then missing_vars+=("MYSQL_HOST"); fi
    if [[ -z "$MYSQL_PORT" ]]; then missing_vars+=("MYSQL_PORT"); fi
    if [[ -z "$MYSQL_USERNAME" ]]; then missing_vars+=("MYSQL_USERNAME"); fi
    if [[ -z "$MYSQL_ROOT_PASSWORD" ]]; then missing_vars+=("MYSQL_ROOT_PASSWORD"); fi
    if [[ -z "$MYSQL_DATABASE" ]]; then missing_vars+=("MYSQL_DATABASE"); fi
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        echo -e "${RED}Error: Missing required environment variables:${NC}"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        echo ""
        echo "Please set these variables in your environment or use --env to specify an environment file."
        exit 1
    fi
}

# Function to check if MySQL is accessible
check_mysql() {
    echo "Checking MySQL connection..."
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USERNAME" -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -e "SELECT 1;" >/dev/null 2>&1 || {
        echo -e "${RED}Error: Cannot connect to MySQL database${NC}"
        echo "Host: $MYSQL_HOST:$MYSQL_PORT"
        echo "Database: $MYSQL_DATABASE"
        echo "User: $MYSQL_USERNAME"
        exit 1
    }
    echo -e "${GREEN}MySQL connection successful${NC}"
}

# Function to create the implementation in the database
create_implementation() {
    local client_id="$1"
    local client_secret="$2"
    local callback_url="$3"
    
    echo "Creating implementation in database..."
    
    # SQL query to insert the implementation
    local sql="INSERT INTO outreach_implementations (client_id, callback_url, client_secret, created_at, updated_at) 
               VALUES ('$client_id', '$callback_url', '$client_secret', NOW(), NOW())
               ON DUPLICATE KEY UPDATE 
                   callback_url = VALUES(callback_url),
                   client_secret = VALUES(client_secret),
                   updated_at = NOW();"
    
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USERNAME" -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -e "$sql" || {
        echo -e "${RED}Error: Failed to create implementation in database${NC}"
        exit 1
    }
    
    echo -e "${GREEN}Implementation created successfully${NC}"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --env=*)
            ENV_FILE="${1#*=}"
            shift
            ;;
        --callback=*)
            CALLBACK_URL="${1#*=}"
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            echo -e "${RED}Error: Unknown option $1${NC}"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$CLIENT_ID" ]]; then
                CLIENT_ID="$1"
            else
                echo -e "${RED}Error: Too many arguments${NC}"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate required arguments
if [[ -z "$CLIENT_ID" ]]; then
    echo -e "${RED}Error: client_id is required${NC}"
    usage
    exit 1
fi

# Set default environment file if not specified
if [[ -z "$ENV_FILE" ]]; then
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        ENV_FILE="$PROJECT_ROOT/.env"
    fi
fi

# Load environment if env file is specified
if [[ -n "$ENV_FILE" ]]; then
    load_env "$ENV_FILE"
fi

# Set default callback URL if not provided
if [[ -z "$CALLBACK_URL" ]]; then
    CALLBACK_URL="http://localhost:8080/callback"
    echo -e "${YELLOW}Warning: No callback URL specified, using default: $CALLBACK_URL${NC}"
    echo ""
fi

# Validate environment variables
validate_env

# Check MySQL connection
check_mysql

# Generate client secret
CLIENT_SECRET=$(generate_secret)

echo ""
echo "Creating outreach implementation:"
echo "  Client ID: $CLIENT_ID"
echo "  Callback URL: $CALLBACK_URL"
echo "  Client Secret: $CLIENT_SECRET"
echo ""

# Create the implementation
create_implementation "$CLIENT_ID" "$CLIENT_SECRET" "$CALLBACK_URL"

echo ""
echo "You can now use this implementation with the following credentials:"
echo "  Client ID: $CLIENT_ID"
echo "  Client Secret: $CLIENT_SECRET"
echo ""
echo "Example curl command to test registration:"
echo "curl -X POST http://localhost:8080/api/outreach/implementations \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{"
echo "    \"client_id\": \"$CLIENT_ID\","
echo "    \"client_secret\": \"$CLIENT_SECRET\","
echo "    \"callback_url\": \"$CALLBACK_URL\""
echo "  }'"
echo ""
echo "Example authentication header for protected endpoints:"
echo "Authorization: Bearer $CLIENT_ID:$CLIENT_SECRET"
