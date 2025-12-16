# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Required for go-sqlite3 (CGO)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build server
RUN CGO_ENABLED=1 GOOS=linux \
    go build -o server ./cmd/server

# Build worker
RUN CGO_ENABLED=1 GOOS=linux \
    go build -o worker ./cmd/worker


# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

# Copy binaries
COPY --from=builder /app/server .
COPY --from=builder /app/worker .

# Copy static assets
COPY static ./static

EXPOSE 8080

# Default command → server
CMD ["./server"]
