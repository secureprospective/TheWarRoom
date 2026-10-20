package comms

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// MFL freezes weekly rosters at the week's last kickoff, not at the archive import time.
func RosterWeekTimes(games facts.NflGameColumns) map[[2]int64]int64 {
	times := map[[2]int64]int64{}
	for i, year := range games.Year {
		key := [2]int64{year, games.Week[i]}
		if games.Kickoff[i] != nil && *games.Kickoff[i] > times[key] {
			times[key] = *games.Kickoff[i]
		}
	}
	return times
}

func (in Input) droppedContract(id, franchise string, drop time.Time) string {
	last := -1
	rows := in.Rosters
	var at int64
	for i, year := range rows.Year {
		if rows.Pid[i] != id || rows.Fid[i] != franchise || rows.Week[i] == 0 {
			continue
		}
		t := in.RosterTimes[[2]int64{year, rows.Week[i]}]
		if t > at && t < drop.Unix() {
			last, at = i, t
		}
	}
	if last < 0 || rows.Salary[last] == nil {
		return " (salary not held)"
	}
	salary, err := domain.ParseMoneyMillions(strconv.FormatFloat(*rows.Salary[last], 'f', -1, 64))
	if err != nil {
		return " (salary not held)"
	}
	year := 0
	if rows.ContractYear[last] != nil {
		year, err = strconv.Atoi(*rows.ContractYear[last])
	}
	if err != nil || year < int(rows.Year[last]) {
		return " (" + bid.MoneyText(salary) + "; contract end not held)"
	}
	return fmt.Sprintf(" (%s, contract through %d)", bid.MoneyText(salary), year)
}

func URL(destination, host string, season int) (string, error) {
	if destination == "league-board" {
		return "https://legacynfl.boards.net/", nil
	}
	option := map[string]string{"message-board": "28", "polls": "69"}[destination]
	if option == "" {
		return "", fmt.Errorf("comms: invalid hand-off destination")
	}
	if host == "" {
		return "", fmt.Errorf("comms: MFL host not held; refresh from MFL first")
	}
	return fmt.Sprintf("https://%s/%d/options?L=%s&O=%s", host, season, url.QueryEscape(ingestion.LeagueID), option), nil
}
