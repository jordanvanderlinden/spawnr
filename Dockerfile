# Build stage
FROM golang:1.25-alpine AS builder

# Install git, ca-certificates, and AWS CLI (needed for EKS access)
RUN apk add --no-cache git ca-certificates tzdata aws-cli

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o spawnr .

# Final stage - use alpine for AWS CLI support
FROM alpine:latest

# Install ca-certificates and AWS CLI
RUN apk add --no-cache ca-certificates aws-cli

# Copy the binary from builder stage
COPY --from=builder /app/spawnr /spawnr

# Copy web assets
COPY --from=builder /app/web /web

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/spawnr"]
