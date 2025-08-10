# Use an official Go image as the build stage
FROM golang:tip-bookworm AS builder

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go binary with version information
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=${VERSION}" -o web-scraper

# Use a minimal base image for final container
FROM debian:bullseye-slim

# Install Chrome/Chromium dependencies and CA certificates
RUN apt-get update && apt-get install -y \
    ca-certificates \
    chromium \
    chromium-driver \
    dumb-init \
    fonts-liberation \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpango-1.0-0 \
    libx11-6 \
    libxcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxrandr2 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/web-scraper .

# Create /dev/shm volume directory
RUN mkdir -p /dev/shm

# Set environment variables (can be overridden at runtime)
ENV REDIS_HOST=redis-service \
    REDIS_PORT=6379 \
    RABBITMQ_HOST=rabbitmq-service \
    RABBITMQ_PORT=5672 \
    # Add scraping configuration
    SCRAPE_INTERVAL=300 \
    # Default tournament ID
    TOURNAMENT_ID="" \
    # Chrome/Chromium configurations
    CHROME_PATH=/usr/bin/chromium \
    CHROME_NO_SANDBOX=true

# Expose health check port
EXPOSE 8080

# Use dumb-init as entrypoint to handle signals properly
ENTRYPOINT ["/usr/bin/dumb-init", "--"]

# Command to run the service
CMD ["./web-scraper"]
