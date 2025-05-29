# Use an official Go image as the build stage
FROM golang:tip-bookworm AS builder

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go binary
RUN go build -o web-scraper 

# Use a minimal base image for final container
#FROM debian:bullseye-slim

# Install CA certificates (needed for HTTPS requests, etc.)
#RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Set working directory
#WORKDIR /app

# Copy the binary from builder
#COPY --from=builder /app/web-scraper .

# Set environment variables (can be overridden at runtime)
ENV REDIS_HOST=redis:6379
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Command to run the service
CMD ["./web-scraper"]
