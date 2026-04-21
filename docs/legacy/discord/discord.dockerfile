# -----------------------------
# Build stage
# -----------------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod, go.sum, and vendor first
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy the rest of the source
COPY . .

# Build with vendored deps
RUN go build -mod=vendor -o main .

# -----------------------------
# Run stage
# -----------------------------
FROM alpine:latest

ENV TZ="America/New_York"

WORKDIR /app

COPY --from=builder /app/main .
COPY .env.docker .env

EXPOSE 9000

CMD ["./main"]
