package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/go-redis/redis"
)

// Match struct represents individual match objects in the "matches" array
type Match struct {
	MatchID string `json:"matchID"`
	Name    string `json:"name"`
	Status  string `json:"status"`
}

// Attributes struct to represent the nested attributes of the tournament
type Attributes struct {
	TournamentID string  `json:"tournamentID"`
	Name         string  `json:"name"`
	Season       int     `json:"season"`
	StartDate    string  `json:"startDate"`
	EndDate      string  `json:"endDate"`
	Matches      []Match `json:"matches"` // A slice of Match structs
}

// Data struct to represent the data object that holds the type, id, and attributes
type Data struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Attributes Attributes `json:"attributes"`
}

// Response struct to represent the entire JSON structure
type Response struct {
	Data Data `json:"data"`
}

func main() {
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
		os.Exit(1)
	}

	// Parse the JSON into a slice of structs
	var response Response

	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshaling the JSON:", err)
		os.Exit(1)
	}

	// Now cache matches into Redis

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: "192.168.100.254:6379",
		DB:   0,
	})

	//  Store all matches from given tournament in Redis
	for _, match := range response.Data.Attributes.Matches {
		// Serialize match data as JSON
		matchJSON, err := json.Marshal(match)
		if err != nil {
			fmt.Println("Error marshaling match:", err)
			continue
		}
		// Store the match details as a JSON string in Redis hash
		err = rdb.HSet("match:"+match.MatchID, "data", matchJSON).Err()
		if err != nil {
			log.Fatalf("Error storing match in Redis: %v", err)
		}

		// Add the match ID to the appropriate set based on status
		switch match.Status {
		case "Live":
			err = rdb.SAdd("live_matches", match.MatchID).Err()
		case "Completed":
			err = rdb.SAdd("completed_matches", match.MatchID).Err()
		case "Scheduled":
			err = rdb.SAdd("scheduled_matches", match.MatchID).Err()
		}

		if err != nil {
			log.Fatalf("Error adding match ID to set by status: %v", err)
		}
	}

	fmt.Println("All matches stored in Redis by status!")

	// Example: Retrieve all live matches
	liveMatchIDs, err := rdb.SMembers("live_matches").Result()
	if err != nil {
		log.Fatalf("Error retrieving live matches: %v", err)
	}

	fmt.Println("Live Match IDs:", liveMatchIDs)

	// Retrieve match details for each live match
	for _, matchID := range liveMatchIDs {
		matchData, err := rdb.HGet("match:"+matchID, "data").Result()
		if err != nil {
			log.Fatalf("Error retrieving match data for match ID %s: %v", matchID, err)
		}

		var match Match
		err = json.Unmarshal([]byte(matchData), &match)
		if err != nil {
			log.Fatalf("Error unmarshaling match data: %v", err)
		}

		fmt.Printf("Live Match: %+v\n", match)
	}
	// Print match details
	// fmt.Println("Scheduled matches:")
	// for _, match := range response.Data.Attributes.Matches {
	// 	if match.Status == "Live" {
	// 		fmt.Printf("Match ID: %s\n , Name: %s\n , Status: %s\n", match.MatchID, match.Name, match.Status)
	// 	}

	// }

}
