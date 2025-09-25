#!/bin/bash

set -e 

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CONTAINER_NAME="assistant-discord"
PORT=9000
NETWORK_NAME="assistant-network"


# Stop any running container
if [[ $(docker ps -qa -f name=$CONTAINER_NAME) ]]; then
    echo -e "${YELLOW}Stopping existing $CONTAINER_NAME container...${NC}"
    docker stop $CONTAINER_NAME
    docker rm $CONTAINER_NAME
    echo -e ""
fi

echo -e "${YELLOW}Starting deployment of $CONTAINER_NAME...${NC}"
echo -e ""

# Setup vendor
echo -e "${YELLOW}Setting up Go modules vendor...${NC}"
if go mod vendor; then
    echo -e "${GREEN}Successfully set up Go modules vendor${NC}"
else
    echo -e "${RED}Failed to set up Go modules vendor${NC}"
    exit 1
fi
echo -e ""

# Build the image 
echo -e "${YELLOW}Building Docker image for $CONTAINER_NAME...${NC}"
if docker build -t $CONTAINER_NAME -f Dockerfile.discord .; then
    echo -e "${GREEN}Successfully built Docker image for $CONTAINER_NAME${NC}"
else
    echo -e "${RED}Failed to build Docker image for $CONTAINER_NAME${NC}"
    exit 1
fi
echo -e ""

# Run the container and attach it directly to assistant-network
echo -e "${YELLOW}Running $CONTAINER_NAME container...${NC}"
if docker run -d --name $CONTAINER_NAME --network $NETWORK_NAME -p $PORT:$PORT $CONTAINER_NAME; then
    echo -e "${GREEN}Successfully started $CONTAINER_NAME container${NC}"
else
    echo -e "${RED}Failed to start $CONTAINER_NAME container${NC}"
    exit 1
fi
echo -e ""

# Attach compose network
echo -e "${YELLOW}Checking connection status of $CONTAINER_NAME to assistant-network...${NC}"
if docker network inspect assistant-network | grep $CONTAINER_NAME; then
    echo -e "${GREEN}$CONTAINER_NAME is already connected to assistant-network${NC}"
else
    echo -e "${YELLOW}$CONTAINER_NAME is not connected to assistant-network. Attempting to connect...${NC}"
    if docker network connect assistant-network $CONTAINER_NAME; then
        echo -e "${GREEN}Successfully reconnected $CONTAINER_NAME to assistant-network${NC}"
    else 
        echo -e "${RED}Failed to reconnect $CONTAINER_NAME to assistant-network${NC}"
        exit 1
    fi
fi
echo -e ""

echo -e "${GREEN}$CONTAINER_NAME is up and running on port $PORT${NC}"
echo -e ""
