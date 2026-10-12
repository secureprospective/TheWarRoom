// Package livegame holds personal-use NFL game facts behind a source-neutral adapter.
package livegame

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Team struct {
	ID           string `json:"id"`
	Abbreviation string `json:"abbreviation"`
	Score        int    `json:"score"`
}
type State struct {
	Possession string `json:"possession"`
	Down       int    `json:"down"`
	Distance   int    `json:"distance"`
	YardLine   int    `json:"yardLine"`
	ToEndzone  int    `json:"toEndzone"`
}
type Play struct {
	ID           string    `json:"id"`
	Sequence     int       `json:"sequence"`
	Text         string    `json:"text"`
	Clock        string    `json:"clock"`
	Period       int       `json:"period"`
	At           time.Time `json:"at" ts_type:"string"`
	HomeScore    int       `json:"homeScore"`
	AwayScore    int       `json:"awayScore"`
	Start        *State    `json:"start"`
	End          *State    `json:"end"`
	Participants []string  `json:"participants"`
}
type Box struct {
	Team     string   `json:"team"`
	Category string   `json:"category"`
	Labels   []string `json:"labels"`
	Rows     []BoxRow `json:"rows"`
}
type BoxRow struct {
	Name  string   `json:"name"`
	Stats []string `json:"stats"`
}
type Game struct {
	ID      string    `json:"id"`
	Kickoff time.Time `json:"kickoff" ts_type:"string"`
	Home    Team      `json:"home"`
	Away    Team      `json:"away"`
	Status  string    `json:"status"`
	Clock   string    `json:"clock"`
	Period  int       `json:"period"`
	State   *State    `json:"state"`
	Last    *Play     `json:"last"`
	Box     []Box     `json:"box"`
	Plays   []Play    `json:"plays"`
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func number(field, s string, upper int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", field, err)
	}
	if n < 0 || n > upper {
		return 0, fmt.Errorf("%s: out of range", field)
	}
	return n, nil
}
func abbreviation(s string) bool {
	if len(s) < 2 || len(s) > 3 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
func clockOK(s string) bool {
	a, b, ok := strings.Cut(s, ":")
	if !ok {
		return false
	}
	_, e := number("clock minutes", a, 15)
	_, f := number("clock seconds", b, 59)
	return e == nil && f == nil && len(b) == 2
}
func referenceID(ref, kind string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("%s reference: %w", kind, err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(parts) < 2 || parts[len(parts)-2] != kind || !digits(parts[len(parts)-1]) {
		return "", fmt.Errorf("%s reference: invalid path", kind)
	}
	return parts[len(parts)-1], nil
}
