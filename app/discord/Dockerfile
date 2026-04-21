# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy module files and vendored dependencies first for layer caching
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy source and the root module (referenced via replace directive)
COPY . .

RUN go build -mod=vendor -o discord .

# Run stage
FROM alpine:latest

ENV TZ="America/New_York"

WORKDIR /app

COPY resources /app/resources
COPY .env.docker .env

COPY --from=builder /app/discord .

EXPOSE 9000

CMD ["./discord"]
