# Multi-stage build for GNOSISAGENT Platform

# Stage 1: Build environment
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    g++ \
    musl-dev \
    make \
    pkgconfig \
    mlpack-dev \
    armadillo-dev

# Install Templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate Templ templates
RUN templ generate

# Build MLPack wrapper
WORKDIR /app/internal/ml/mlpack
RUN if [ -f anomaly_detector.cpp ]; then \
        g++ -std=c++17 -c anomaly_detector.cpp -o anomaly_detector.o \
            $(pkg-config --cflags mlpack armadillo) -fPIC && \
        ar rcs libanomalydetector.a anomaly_detector.o; \
    fi

# Build the application
WORKDIR /app
RUN CGO_ENABLED=1 go build \
    -ldflags="-w -s" \
    -o /app/bin/GNOSISAGENT \
    ./cmd/GNOSISAGENT

# Stage 2: Runtime environment
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    mlpack \
    armadillo \
    libstdc++

# Create app user
RUN addgroup -g 1000 GNOSISAGENT && \
    adduser -D -u 1000 -G GNOSISAGENT GNOSISAGENT

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/GNOSISAGENT /usr/local/bin/GNOSISAGENT

# Copy configuration (optional, can be mounted)
COPY configs/org.example.yaml /app/config/org.yaml

# Change ownership
RUN chown -R GNOSISAGENT:GNOSISAGENT /app

# Switch to app user
USER GNOSISAGENT

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/GNOSISAGENT"]
CMD ["start", "--config", "/app/config/org.yaml"]
