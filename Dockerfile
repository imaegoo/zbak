# Multi-stage build for zbak
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o zbak ./cmd/zbak

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install p7zip package
RUN apk add --no-cache p7zip

# Copy the binary from builder stage
COPY --from=builder /build/zbak /usr/local/bin/zbak

# Set working directory
WORKDIR /data

# Set default command
CMD ["zbak"]
