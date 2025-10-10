# Use the official Go image as the base image
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

# Use a minimal image for the final stage
FROM alpine:latest

ENV TZ="America/New_York"

# Install dependencies
# - nodejs, npm, and git for `npx` commands (MCP)
# - tzdata for timezone support
RUN apk --no-cache add nodejs npm git tzdata

WORKDIR /app

# Copy go.mod for absolute path references
COPY go.mod /app/go.mod

# Copy resources and env
COPY resources /app/resources
COPY .env.docker /app/.env

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./main"]
