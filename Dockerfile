# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install templ CLI for generating templates
RUN go install github.com/a-h/templ/cmd/templ@latest

# Generate templ templates
RUN templ generate

# Build the binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o server \
    ./cmd/server/main.go

# Stage 2: Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Copy necessary runtime files
COPY --from=builder /build/internal/database/migrations ./internal/database/migrations

# Create directories for static files (if needed in future)
RUN mkdir -p web/static && chown -R appuser:appuser /app

# Create directory for database with proper permissions
RUN mkdir -p /data && chown -R appuser:appuser /data

# Switch to non-root user
USER appuser

# Set environment variables
ENV PORT=8080
ENV DB_PATH=/data/cleaning.db

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/login || exit 1

# Run the application
CMD ["./server"]
