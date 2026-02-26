# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin-notifier ./cmd/notifier

# Runtime stage with headless Chrome
FROM chromedp/headless-shell:stable

# Install ca-certificates for HTTPS requests (Twilio API)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/bin-notifier /usr/local/bin/bin-notifier

# Create non-root user for security
RUN useradd -r -u 1001 appuser
USER appuser

ENTRYPOINT ["/usr/local/bin/bin-notifier"]
