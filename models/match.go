package models

// Match struct represents individual match objects in the "matches" array
type Match struct {
	MatchID            string `json:"matchID"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	Round              string `json:"round"`
	HomePlayer         Player `json:"homePlayer"`
	HomePlayerID       string `json:"homePlayerId"`
	HomePlayerImage    string `json:"homePlayerImage"`
	HomePlayerScore    int    `json:"homePlayerScore"`
	AwayPlayer         Player `json:"awayPlayer"`
	AwayPlayerID       string `json:"awayPlayerId"`
	AwayPlayerImage    string `json:"awayPlayerImage"`
	AwayPlayerScore    int    `json:"awayPlayerScore"`
	TournamentName     string `json:"tournamentName"`
	MatchStartDateTime string `json:"startDateTime"`
}

type Player struct {
	PlayerId  string `json:"playerID"`
	FirstName string `json:"firstName"`
	LastName  string `json:"surname"`
	Media     Media  `json:"media"`
}

type Media struct {
	Image string `json:"profile"`
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
