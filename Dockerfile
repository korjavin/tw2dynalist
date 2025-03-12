# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tw2dynalist .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/tw2dynalist .

# Create a directory for the cache
RUN mkdir -p /app/cache

# Set the cache file path
ENV CACHE_FILE_PATH=/app/cache/cache.json
ENV LOG_LEVEL=INFO

# Run the application
CMD ["./tw2dynalist"]