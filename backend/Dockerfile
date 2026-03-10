# ============================================================================
# Stage 1: Build
# ============================================================================
FROM golang:1.25-alpine AS builder

# Install git + ca-certificates (needed for fetching modules over HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source code
COPY . .

ARG TARGETOS
ARG TARGETARCH

# Build a statically linked binary with all optimizations
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /app/bin/server \
    ./cmd/server

# ============================================================================
# Stage 2: Production
# ============================================================================
FROM alpine:3.20 AS production

# Install ca-certificates for outbound HTTPS (Groq, Jina, Discord, Razorpay)
# and tzdata for timezone support in Go's time package
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /app/bin/server /app/server

# Copy migrations so they run on startup
COPY --from=builder /app/migrations /app/migrations

# The .env file is NOT copied — environment variables must be injected at
# runtime via docker run --env-file, docker-compose environment:, or the
# orchestrator's secret management (Kubernetes Secrets, Railway, Render, etc.)

# Switch to non-root user for security
USER appuser

# Expose the default server port
EXPOSE 8080

# Health check — the /healthz endpoint returns {"status":"ok"} when the
# server is ready to accept connections.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

# Run the server binary directly (no shell wrapper needed)
ENTRYPOINT ["/app/server"]
