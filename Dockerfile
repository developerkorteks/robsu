# Build stage
FROM golang:1.24-alpine AS builder

# Install git, ca-certificates, and build tools (needed for CGO and SQLite)
RUN apk add --no-cache git ca-certificates gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates and sqlite (needed for the app)
RUN apk --no-cache add ca-certificates sqlite

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy .env file
COPY --from=builder /app/.env .

# Create directory for database
RUN mkdir -p /root/data

# Expose port 1462
EXPOSE 1462

# Set environment variable for port
ENV PORT=1462

# Run the application
CMD ["./main"]