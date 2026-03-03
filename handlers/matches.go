package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// matchDetailResponse models the single-match API response used to fetch history.
type matchDetailResponse struct {
	Data matchDetailData `json:"data"`
}

type matchDetailData struct {
	Attributes matchDetailAttributes `json:"attributes"`
}

type matchDetailAttributes struct {
	History models.History `json:"history"`
}

// FetchMatches fetches match data from a URL and returns it as a models.Response.
// It uses chromedp to scrape the match data from the WST website and then fetches the JSON data from a specific URL.
func FetchMatches(cfg *config.Config) (models.Response, error) {

	// Verify Chrome/Chromium exists
	if _, err := os.Stat(cfg.Chromium.Path); err != nil {
		return models.Response{}, fmt.Errorf("browser not found at %s: %v", cfg.Chromium.Path, err)
	}

	log.Printf("Using browser at path: %s", cfg.Chromium.Path)

	// Create chrome options with detailed logging
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", cfg.Chromium.NoSandbox),
		chromedp.Flag("disable-dev-shm-usage", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.ExecPath(cfg.Chromium.Path),
	)

	// Create allocator context with timeout
	allocCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	allocCtx, cancel = chromedp.NewExecAllocator(allocCtx, opts...)
	defer cancel()

	// Create browser context with logging
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Store the content that will be scraped
	var pageContent string

	log.Printf("Attempting to scrape WST matches page...")

	//Run chromedp tasks with better error handling
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.wst.tv/matches/"),
		chromedp.WaitVisible(`section.h-full`, chromedp.ByQuery),
		chromedp.OuterHTML(`section.h-full`, &pageContent, chromedp.ByQuery),
	); err != nil {
		return models.Response{}, fmt.Errorf("failed to scrape matches page (browser: %s): %v",
			cfg.Chromium.Path, err)
	}

	log.Printf("Successfully scraped matches page")

	// Parse the scraped HTML
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
	if err != nil {
		return models.Response{}, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var id string

	// First priority: Check configuration
	if cfg.Scraper.TournamentID != "" {
		id = cfg.Scraper.TournamentID
		log.Printf("Using tournament ID from configuration: %s", id)
	} else {
		// Second priority: Try to get tournament ID from the page
		section := dom.Find("section.h-full")
		var exists bool
		id, exists = section.Attr("id")

		if exists {
			log.Printf("Found tournament ID from page: %s", id)
		} else {
			// Third priority: Use default ID
			id = "b964199d-4b71-4d26-8436-e141ca3f2751"
			log.Printf("No tournament ID found in config or page, using default: %s", id)
		}
	}

	// Check if tournament has changed
	tournamentIdInCache := getTournamentIdFromCache()
	if tournamentIdInCache != id {
		log.Printf("New tournament detected (old: %s, new: %s), flushing cache", tournamentIdInCache, id)
		if err := redis.ClearAppCache(); err != nil {
			return models.Response{}, fmt.Errorf("failed to clear app cache: %v", err)
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

func FetchAndStoreMatchDetails(match *models.Match, cfg *config.Config) error {
	detailsURL := fmt.Sprintf("https://matches.snooker.web.gc.wstservices.co.uk/v2/%s", match.MatchID)
	resp, err := http.Get(detailsURL)
	if err != nil {
		return fmt.Errorf("failed to fetch match details for match %s: %v", match.MatchID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read match details response for match %s: %v", match.MatchID, err)
	}

	var detail matchDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return fmt.Errorf("failed to parse match details for match %s: %v", match.MatchID, err)
	}

	match.History = &detail.Data.Attributes.History
	log.Printf("Fetched history for match %s", match.MatchID)
	return nil
}

// DumpMatches prints the fetched matches data in a readable format
func DumpMatches(matches models.Response, outputFile string) error {
	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal matches to JSON: %v", err)
	}

	if outputFile != "" {
		// Save to file
		err = os.WriteFile(outputFile, prettyJSON, 0644)
		if err != nil {
			return fmt.Errorf("failed to write matches to file %s: %v", outputFile, err)
		}
		log.Printf("Matches dumped to file: %s", outputFile)
	} else {
		// Print to console
		fmt.Println("=== FETCHED MATCHES DUMP ===")
		fmt.Println(string(prettyJSON))
		fmt.Println("=== END DUMP ===")
	}

	log.Printf("Dumped %d matches", len(matches.Data.Attributes.Matches))
	return nil
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
		// Populate player images from media data
		match.HomePlayerImage = match.HomePlayer.Media.Image
		match.AwayPlayerImage = match.AwayPlayer.Media.Image
		match.TournamentName = matches.Data.Attributes.Name

		// Store player images in Redis for easy access
		if err := storePlayerImage(match.HomePlayer.PlayerId, match.HomePlayerImage); err != nil {
			log.Printf("Warning: failed to store home player image for %s: %v", match.HomePlayer.PlayerId, err)
		}
		if err := storePlayerImage(match.AwayPlayer.PlayerId, match.AwayPlayerImage); err != nil {
			log.Printf("Warning: failed to store away player image for %s: %v", match.AwayPlayer.PlayerId, err)
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

// storePlayerImage stores a player's profile image in Redis
func storePlayerImage(playerID, imageFilename string) error {
	if playerID == "" || imageFilename == "" {
		return fmt.Errorf("playerID and imageFilename cannot be empty")
	}

	key := "player:" + playerID
	err := redis.Rdb.HSet(key, "image", imageFilename).Err()
	if err != nil {
		return fmt.Errorf("failed to store player image in Redis: %v", err)
	}

	log.Printf("Stored player image: %s -> %s", playerID, imageFilename)
	return nil
}

// getPlayerImage retrieves a player's profile image from Redis
func getPlayerImage(playerID string) (string, error) {
	if playerID == "" {
		return "", fmt.Errorf("playerID cannot be empty")
	}

	key := "player:" + playerID
	image, err := redis.Rdb.HGet(key, "image").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get player image from Redis: %v", err)
	}

	return image, nil
}
