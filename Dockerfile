# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Required for go-sqlite3 (CGO)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/jstemmer/go-junit-report/v2@latest

COPY . .

# Build server
RUN CGO_ENABLED=1 GOOS=linux \
    go build -o server ./cmd/server

# Build worker
RUN CGO_ENABLED=1 GOOS=linux \
    go build -o worker ./cmd/worker

# ---- Test stage ----
FROM builder AS test

COPY static ./static

CMD ["sh", "-c", "go test -v ./... 2>&1 | go-junit-report -set-exit-code > test-results.xml"]

# ---- Runtime stage ----
FROM alpine:3.19 AS runtime

WORKDIR /app

# Copy binaries
COPY --from=builder /app/server .
COPY --from=builder /app/worker .
COPY --from=builder /app/static ./static