# BUILD STAGE
FROM golang:1.24.2-alpine3.21 AS builder

WORKDIR /app

# Install git for downloading Go modules
RUN apk add --no-cache git

# Set build environment variables
ENV CGO_ENABLED=0 GOOS=linux

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary
RUN go build -o employee-management ./main.go

# FINAL STAGE
FROM alpine:3.18

# Set working directory
WORKDIR /app

# Install CA certificates for HTTPS/TLS (required for MongoDB Atlas connection)
RUN apk add --no-cache ca-certificates

# Copy the built Go binary and env file from builder stage
COPY --from=builder /app/employee-management .

# Expose the application port
EXPOSE 8080

# Run container as non-root user
RUN adduser -D appuser
USER appuser

# Start the application
CMD ["./employee-management"]
