package main

import (
	"log"

	"github.com/astoyanov87/web-scrapper/handlers"
	"github.com/astoyanov87/web-scrapper/redis"
)

func main() {

	// Initialize Redis client
	redis.InitRedis()

	// Fetch matches (simulating a web scraping or API request)
	matches, err := handlers.FetchMatches()
	if err != nil {
		log.Fatalf("Error fetching matches: %v", err)
	}

	// Store matches in Redis
	err = handlers.StoreMatches(matches)
	if err != nil {
		log.Fatalf("Error storing matches in Redis: %v", err)
	}

}
