# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with version info
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE} -w -s" \
    -o platform-api .

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 platform && \
    adduser -D -u 1000 -G platform platform

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/platform-api .

# Use non-root user
USER platform

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/platform-api"]
