package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astoyanov87/web-scrapper/config"
	"github.com/astoyanov87/web-scrapper/handlers"
	"github.com/astoyanov87/web-scrapper/redis"
)

// Version is set at build time via ldflags
var Version string

func main() {
	// Log version if set
	if Version != "" {
		log.Printf("Starting web-scrapper version: %s", Version)
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Log configuration
	log.Printf("Starting web-scrapper with configuration:")
	log.Printf("Redis: %s:%s", cfg.Redis.Host, cfg.Redis.Port)
	log.Printf("RabbitMQ: %s:%s", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	log.Printf("Tournament ID: %s", cfg.Scraper.TournamentID)
	log.Printf("Scrape Interval: %v", cfg.Scraper.ScrapeInterval)

	// Initialize Redis client with config
	if err := redis.InitRedis(cfg); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Start health check server
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Starting health check server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("Health check server failed: %v", err)
		}
	}()

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for periodic execution
	ticker := time.NewTicker(cfg.Scraper.ScrapeInterval)
	defer ticker.Stop()

	// Run the first scrape immediately
	if err := scrapeAndStore(cfg); err != nil {
		log.Printf("Initial scrape failed: %v", err)
	}

	// Main loop
	for {
		select {
		case <-ticker.C:
			if err := scrapeAndStore(cfg); err != nil {
				log.Printf("Periodic scrape failed: %v", err)
			}
		case sig := <-sigChan:
			log.Printf("Received signal: %v", sig)
			cancel()
			return
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
	}
}

func scrapeAndStore(cfg *config.Config) error {
	// Fetch matches using config
	matches, err := handlers.FetchMatches(cfg)
	if err != nil {
		return fmt.Errorf("failed to fetch matches: %v", err)
	}

	// Dump matches for debugging (optional - set DUMP_MATCHES_FILE env var to save to file)
	dumpFile := os.Getenv("DUMP_MATCHES_FILE")
	if err := handlers.DumpMatches(matches, dumpFile); err != nil {
		log.Printf("Warning: failed to dump matches: %v", err)
	}

	// Store matches in Redis
	if err := handlers.StoreMatches(matches, cfg); err != nil {
		return fmt.Errorf("failed to store matches: %v", err)
	}

	log.Printf("Successfully scraped and stored %d matches", len(matches.Data.Attributes.Matches))
	return nil
}
