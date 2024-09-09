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

	"github.com/PuerkitoBio/goquery"
	eventhandlers "github.com/astoyanov87/web-scrapper/eventhandlers"
	"github.com/astoyanov87/web-scrapper/models"
	"github.com/astoyanov87/web-scrapper/redis"
	"github.com/chromedp/chromedp"
)

type MatchDetailsFromCache struct {
	ID     string `json:"matchID"`
	Status string `json:"status"`
}

// FetchMatches fetches match data from a URL (simulating a web scraping or API request)
func FetchMatches() (models.Response, error) {

	// Create a context for chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Store the content that will be scraped
	var pageContent string

	// Run chromedp tasks
	chromedp.Run(ctx,
		chromedp.Navigate("https://www.wst.tv/matches/"),
		// Wait for the match data to be loaded
		chromedp.WaitVisible(`section.h-full`),
		// Scrape the HTML content of the matches section
		chromedp.OuterHTML(`section.h-full`, &pageContent),
	)
	//fmt.Println(pageContent)
	// Print the scraped HTML
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
	if err != nil {
		panic(err)
	}

	section := dom.Find("section.h-full")
	id, exists := section.Attr("id")
	if exists {
		fmt.Println("ID found:", id)
	} else {
		fmt.Println("ID not found")
	}

	tournamentIdInCache := getTournamentIdFromCache()
	if tournamentIdInCache != id {
		// there is new tournament in play
		// flush all data in Redis and cache the new tournament ID
		result := redis.Rdb.FlushAll()
		fmt.Println("Flushing the cache: " + result.Val())
		storeTournamentId(id)
	} else {
		fmt.Println("Tournament ID found in cache! Continue ...")
	}

	url := "https://tournaments.snooker.web.gc.wstservices.co.uk/v2/" + id
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

func StoreMatches(matches models.Response) error {

	//  Store all matches from given tournament in Redis
	for _, match := range matches.Data.Attributes.Matches {

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

				err := eventhandlers.PublishEvent(event)
				if err != nil {
					log.Printf("Failed to publish status change event: %v", err)
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
