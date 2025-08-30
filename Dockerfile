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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tw2dynalist ./cmd/tw2dynalist

# Stage 2: Create a minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/tw2dynalist .

# Create directories for data storage
RUN mkdir -p /app/data

# Set default environment variables
ENV CACHE_FILE_PATH=/app/data/cache.json
ENV TOKEN_FILE_PATH=/app/data/token.json
ENV LOG_LEVEL=INFO
ENV TWITTER_REDIRECT_URL=http://localhost:8080/callback

# Expose the callback server port
EXPOSE 8080

# Run the application
CMD ["./tw2dynalist"]