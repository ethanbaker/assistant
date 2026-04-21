#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

CONTAINER_NAME="assistant-discord"
PORT=9000
NETWORK_NAME="assistant-network"

# Stop any existing container
if [[ $(docker ps -qa -f name="$CONTAINER_NAME") ]]; then
    echo -e "${YELLOW}Stopping existing $CONTAINER_NAME container...${NC}"
    docker stop "$CONTAINER_NAME"
    docker rm "$CONTAINER_NAME"
    echo -e ""
fi

echo -e "${YELLOW}Starting deployment of $CONTAINER_NAME...${NC}"
echo -e ""

# Vendor Go modules (includes the root module via replace directive)
echo -e "${YELLOW}Vendoring Go modules...${NC}"
if go mod vendor; then
    echo -e "${GREEN}Vendoring complete${NC}"
else
    echo -e "${RED}Vendoring failed${NC}"
    exit 1
fi
echo -e ""

# Build Docker image
echo -e "${YELLOW}Building Docker image...${NC}"
if docker build -t "$CONTAINER_NAME" -f Dockerfile .; then
    echo -e "${GREEN}Image built successfully${NC}"
else
    echo -e "${RED}Image build failed${NC}"
    exit 1
fi
echo -e ""

# Run the container
echo -e "${YELLOW}Starting container...${NC}"
if docker run -d \
    --name "$CONTAINER_NAME" \
    --network "$NETWORK_NAME" \
    -p "$PORT:$PORT" \
    "$CONTAINER_NAME"; then
    echo -e "${GREEN}Container started${NC}"
else
    echo -e "${RED}Failed to start container${NC}"
    exit 1
fi
echo -e ""

# Verify network connection
echo -e "${YELLOW}Verifying network connection to $NETWORK_NAME...${NC}"
if docker network inspect "$NETWORK_NAME" | grep "$CONTAINER_NAME" > /dev/null; then
    echo -e "${GREEN}$CONTAINER_NAME is connected to $NETWORK_NAME${NC}"
else
    echo -e "${YELLOW}Connecting $CONTAINER_NAME to $NETWORK_NAME...${NC}"
    if docker network connect "$NETWORK_NAME" "$CONTAINER_NAME"; then
        echo -e "${GREEN}Connected successfully${NC}"
    else
        echo -e "${RED}Failed to connect to $NETWORK_NAME${NC}"
        exit 1
    fi
fi
echo -e ""

echo -e "${GREEN}$CONTAINER_NAME is running on port $PORT${NC}"
