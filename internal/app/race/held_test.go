package race

import (
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func heldFixture() season.Held {
	return season.Held{
		Weeks:     season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Week: 1, Games: []leaguefeed.Game{{Home: leaguefeed.Side{Players: []leaguefeed.WeekPlayer{{Points: 1}}}, Away: leaguefeed.Side{Players: []leaguefeed.WeekPlayer{{Points: 2}}}}}, Others: []leaguefeed.Side{{Players: []leaguefeed.WeekPlayer{{Points: 3}}}}}}},
		Brackets:  season.Feed[[]leaguefeed.Bracket]{Value: []leaguefeed.Bracket{{Games: []leaguefeed.BracketGame{{ID: 1}}}}},
		Standings: season.Feed[[]leaguestandings.RawStanding]{Value: []leaguestandings.RawStanding{{FranchiseID: "0001"}}},
		Power:     map[string]season.PowerRankingsResult{"season": {Rows: []season.PowerRow{{Name: "held"}}}},
	}
}

func publish(h *Held, in season.Held) {
	h.PublishWeeks(in.Weeks)
	h.PublishBrackets(in.Brackets)
	h.PublishStandings(in.Standings, in.Power, true)
}
