package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/astoyanov87/web-scrapper/config"
	eventhandlers "github.com/astoyanov87/web-scrapper/eventhandlers"
	"github.com/astoyanov87/web-scrapper/models"
	"github.com/astoyanov87/web-scrapper/redis"
	"github.com/chromedp/chromedp"
)

type MatchDetailsFromCache struct {
	ID     string `json:"matchID"`
	Status string `json:"status"`
}

// FetchMatches fetches match data from a URL and returns it as a models.Response.
// It uses chromedp to scrape the match data from the WST website and then fetches the JSON data from a specific URL.
func FetchMatches(cfg *config.Config) (models.Response, error) {

	// Create chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", cfg.Chromium.NoSandbox),
		chromedp.ExecPath(cfg.Chromium.Path),
	)

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create browser context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Add timeout to context
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Store the content that will be scraped
	var pageContent string

	// Run chromedp tasks
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.wst.tv/matches/"),
		chromedp.WaitVisible(`section.h-full`),
		chromedp.OuterHTML(`section.h-full`, &pageContent),
	); err != nil {
		return models.Response{}, fmt.Errorf("failed to scrape matches page: %v", err)
	}

	// Parse the scraped HTML
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
	if err != nil {
		return models.Response{}, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Try to get tournament ID from the page
	section := dom.Find("section.h-full")
	id, exists := section.Attr("id")
	
	// If tournament ID is provided in config, use it
	if cfg.Scraper.TournamentID != "" {
		id = cfg.Scraper.TournamentID
		log.Printf("Using tournament ID from config: %s", id)
	} else if !exists {
		log.Printf("Tournament ID not found in page, using default")
		id = "b964199d-4b71-4d26-8436-e141ca3f2751" // default ID if not found
	} else {
		log.Printf("Found tournament ID from page: %s", id)
	}

	// Check if tournament has changed
	tournamentIdInCache := getTournamentIdFromCache()
	if tournamentIdInCache != id {
		log.Printf("New tournament detected (old: %s, new: %s), flushing cache", tournamentIdInCache, id)
		if result := redis.Rdb.FlushAll(); result.Err() != nil {
			return models.Response{}, fmt.Errorf("failed to flush Redis cache: %v", result.Err())
		}
		if err := storeTournamentId(id); err != nil {
			return models.Response{}, fmt.Errorf("failed to store new tournament ID: %v", err)
		}
	} else {
		log.Printf("Continuing with existing tournament: %s", id)
	}

	url := "https://tournaments.snooker.web.gc.wstservices.co.uk/v2/" + id
	//url := "https://tournaments.snooker.web.gc.wstservices.co.uk/v2/b964199d-4b71-4d26-8436-e141ca3f2751"
	fmt.Println("The url of matches is :", url)

	// Fetch the JSON with matches from the URL
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalf("Failed to fetch the URL: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body using io.ReadAll
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
	}

	// Parse the JSON into a slice of structs
	var matches models.Response

	err = json.Unmarshal(body, &matches)
	if err != nil {
		fmt.Println("Error unmarshaling the JSON:", err)

	}
	return matches, err
}

func StoreMatches(matches models.Response, cfg *config.Config) error {
	// Store all matches from given tournament in Redis
	matchCount := 0
	for _, match := range matches.Data.Attributes.Matches {
		matchCount++

		matchFromCache, err := getMatchfromCacheById(match.MatchID)
		if err != nil {
			fmt.Println("Error retrieving match")

		}
		if matchFromCache != nil {
			log.Printf("Match status in cache is : %+v", matchFromCache.Status)
			log.Printf("Match status in response is : %+v", match.Status)

			if matchFromCache.Status != match.Status {
				//Match status has changed since was stored in cache
				//trigger an MatchStatusChanged event and send it to RabbitMq for cunsumer services
				event := eventhandlers.MatchStatusChangedEvent{
					MatchId:   match.MatchID,
					NewStatus: match.Status,
					MatchName: match.Name,
					Round:     match.Round,
				}

				if err := eventhandlers.PublishEvent(event, cfg); err != nil {
					log.Printf("Failed to publish status change event for match %s: %v", match.MatchID, err)
					// Continue processing other matches even if event publishing fails
				} else {
					log.Printf("Published status change event for match %s: %s -> %s", 
						match.MatchID, matchFromCache.Status, match.Status)
				}

			}
		}
		//Serialize match data as JSON
		matchJSON, err := json.Marshal(match)
		if err != nil {
			fmt.Println("Error marshaling match:", err)
			continue
		}
		// Store the match details as a JSON string in Redis hash
		err = redis.Rdb.HSet("match:"+match.MatchID, "data", matchJSON).Err()
		if err != nil {
			log.Fatalf("Error storing match in Redis: %v", err)
		}

		// Add the match ID to the appropriate set based on status
		switch match.Status {
		case "Live":
			err = redis.Rdb.SAdd("live_matches", match.MatchID).Err()
		case "Completed":
			err = redis.Rdb.SAdd("completed_matches", match.MatchID).Err()
		case "Scheduled":
			err = redis.Rdb.SAdd("scheduled_matches", match.MatchID).Err()
		}

		if err != nil {
			log.Fatalf("Error adding match ID to set by status: %v", err)
		}
	}

	fmt.Println("All matches stored in Redis by status!")
	return nil
}

func getMatchfromCacheById(matchID string) (*MatchDetailsFromCache, error) {
	// Construct the key
	key := "match:" + matchID

	exists, err := redis.Rdb.Exists(key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("match not found")
	}

	// Retrieve the match status field
	data, err := redis.Rdb.HGet(key, "data").Result()
	fmt.Println(data)
	if err != nil {
		fmt.Println("Can not retrieve value from Redis")
	}

	var match MatchDetailsFromCache
	err = json.Unmarshal([]byte(data), &match)
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func storeTournamentId(id string) error {

	result := redis.Rdb.Set("tournamentId", id, 0)
	fmt.Println(result)
	return nil
}

func getTournamentIdFromCache() string {

	result := redis.Rdb.Get("tournamentId")
	fmt.Println("Id from cache: " + result.Val())
	return result.Val()
}
