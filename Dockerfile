# Multi-stage build for Go Control Plane
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o controlplane cmd/controlplane/main.go

# Final stage - minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/controlplane .

# Copy default config (can be overridden with volume mount)
COPY --from=builder /app/config/config.yaml /config/

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./controlplane"]
CMD ["-config", "/config/config.yaml"]
