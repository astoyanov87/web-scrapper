# Web Scraper Service

A microservice that scrapes match data from wst.tv/matches/ and caches it in Redis. Part of the snooker live score application microservices architecture.

## Features

- Scrapes snooker match data from WST website
- Caches match information in Redis
- Publishes match status changes to RabbitMQ
- Configurable scraping interval
- Support for tournament-specific scraping

## Configuration

The service can be configured using environment variables or a `dev.env` file. For local development:

1. Copy the sample configuration file:
```bash
cp config/sample.env config/dev.env
```

2. Edit `config/dev.env` with your local settings:
```env
# Redis Configuration
REDIS_HOST=your-redis-host
REDIS_PORT=6379

# RabbitMQ Configuration
RABBITMQ_HOST=your-rabbitmq-host
RABBITMQ_PORT=5672
RABBITMQ_USERNAME=guest
RABBITMQ_PASSWORD=guest

# Scraper Configuration
TOURNAMENT_ID=        # Optional: specific tournament to scrape
SCRAPE_INTERVAL=300  # Interval in seconds

# Chrome Configuration
CHROME_PATH=/usr/bin/chromium
CHROME_NO_SANDBOX=true
```

### Configuration Priority

1. Environment variables
2. dev.env file
3. Default values

Note: The `dev.env` file is ignored by git to prevent committing sensitive information.

## Running the Service

### Local Development
```bash
go run main.go
```

### Docker
```bash
docker build -t web-scraper .
docker run -p 8080:8080 web-scraper
```

### Kubernetes
```bash
kubectl apply -f k8s/deployment.yaml
```

## Dependencies

- Redis server
- RabbitMQ server
- Chromium/Chrome browser
- Go 1.20 or higher