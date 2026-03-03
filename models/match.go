package models

// Match struct represents individual match objects in the "matches" array
type Match struct {
	MatchID            string   `json:"matchID"`
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	Round              string   `json:"round"`
	HomePlayer         Player   `json:"homePlayer"`
	HomePlayerID       string   `json:"homePlayerId"`
	HomePlayerImage    string   `json:"homePlayerImage"`
	HomePlayerScore    int      `json:"homePlayerScore"`
	AwayPlayer         Player   `json:"awayPlayer"`
	AwayPlayerID       string   `json:"awayPlayerId"`
	AwayPlayerImage    string   `json:"awayPlayerImage"`
	AwayPlayerScore    int      `json:"awayPlayerScore"`
	TournamentName     string   `json:"tournamentName"`
	MatchStartDateTime string   `json:"startDateTime"`
	History            *History `json:"history,omitempty"`
}

type History struct {
	MatchID    string      `json:"matchID"`
	PlayerData interface{} `json:"playerData"`
	MatchData  *MatchData  `json:"matchData"`
}

type MatchData struct {
	Index                 int                   `json:"index"`
	Status                string                `json:"status"`
	MatchId               string                `json:"matchId"`
	ClockTime             string                `json:"clockTime"`
	HomePlayer            bool                  `json:"homePlayer"`
	StatusMeta            string                `json:"statusMeta"`
	CurrentBreak          int                   `json:"currentBreak"`
	MatchHistory          FrameHistory          `json:"matchHistory"`
	BallsRemaining        BallsRemaining        `json:"ballsRemaining"`
	SequenceNumber        int                   `json:"sequenceNumber"`
	AwayPlayerFrames      int                   `json:"awayPlayerFrames"`
	HomePlayerFrames      int                   `json:"homePlayerFrames"`
	CurrentFrameStartTime string                `json:"currentFrameStartTime"`
	MatchPlayerStatistics MatchPlayerStatistics `json:"matchPlayerStatistics"`
}

type FrameHistory struct {
	Frames []Frame `json:"frames"`
}

type Frame struct {
	FrameNumber               int `json:"frameNumber"`
	AwayPlayerPoints          int `json:"awayPlayerPoints"`
	HomePlayerPoints          int `json:"homePlayerPoints"`
	AwayPlayerFiftyPlusBreaks int `json:"awayPlayerFiftyPlusBreaks"`
	HomePlayerFiftyPlusBreaks int `json:"homePlayerFiftyPlusBreaks"`
}

type BallsRemaining struct {
	Red            int `json:"red"`
	Blue           int `json:"blue"`
	Pink           int `json:"pink"`
	Black          int `json:"black"`
	Brown          int `json:"brown"`
	Green          int `json:"green"`
	Yellow         int `json:"yellow"`
	PossiblePoints int `json:"possiblePoints"`
}

type MatchPlayerStatistics struct {
	Players []PlayerStats `json:"players"`
}

type PlayerStats struct {
	Index             int    `json:"index"`
	PotRate           int    `json:"potRate"`
	HomePlayer        bool   `json:"homePlayer"`
	ShotsTaken        int    `json:"shotsTaken"`
	TimeOnTable       int    `json:"timeOnTable"`
	TotalPoints       int    `json:"totalPoints"`
	HighestBreak      int    `json:"highestBreak"`
	AverageShotTime   string `json:"averageShotTime"`
	FiftyPlusBreaks   int    `json:"fiftyPlusBreaks"`
	HundredPlusBreaks int    `json:"hundredPlusBreaks"`
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
