FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files from backend directory
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code from backend directory
COPY backend/ .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o pocketbase .

# Final stage
FROM alpine:latest

# Create a non-root user
RUN addgroup -g 1001 -S pocketbase && \
    adduser -u 1001 -S pocketbase -G pocketbase

WORKDIR /app

# Install ca-certificates for HTTPS and wget for health checks
RUN apk add --no-cache ca-certificates wget

# Copy binary from builder
COPY --from=builder /app/pocketbase .

# Create data directory and set ownership
RUN mkdir -p /app/pb_data && \
    chown -R pocketbase:pocketbase /app

# Switch to non-root user
USER pocketbase

# Expose port
EXPOSE 8090

# Run the application
CMD ["./pocketbase", "serve", "--http=0.0.0.0:8090"]