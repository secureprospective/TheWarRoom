//go:build leaguebench

package history

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func addBooksReadings(t *testing.T, s *Store, g *bigLeagueBuilder, probes map[string]booksProbe) {
	t.Helper()
	at, team := g.now, bigTeam(0)
	probes["managed-schedule"] = readingProbe(func(r []commissioner.Record) (commissioner.ManagedSchedule, error) {
		return commissioner.ManagedScheduleReading(bigLeagueID, "creator", r)
	})
	probes["lineup-check"] = readingProbe(func(r []commissioner.Record) (commissioner.NativeLineupCheck, error) {
		return commissioner.NativeLineupCheckReading(commissioner.RawNativeLineup{League: bigLeagueID, Franchise: team}, r, at, g.facts)
	})
	for _, command := range []string{commissioner.WaiverProcess, commissioner.BidsProcess, commissioner.TradeReview} {
		probes["market/"+command] = readingProbe(func(r []commissioner.Record) (market.Market, error) {
			return market.ChoicesReading(bigLeagueID, command, "creator", r, at, time.UTC, g.facts)
		})
	}
	for _, kind := range []string{"setup", "rules", "people", "notices", "corrections"} {
		probes["filtered-ledger/"+kind] = readingProbe(func(r []commissioner.Record) (commissioner.Ledger, error) {
			state, err := commissioner.GovernanceState(r)
			if err != nil {
				return commissioner.Ledger{}, err
			}
			return commissioner.FilteredReading(bigLeagueID, "creator", r, commissioner.GovernanceAuthorizer{State: state}, commissioner.RawLedgerFilter{Kind: kind})
		})
	}
	for year := 2020; year <= 2025; year++ {
		probes[fmt.Sprintf("final-standings/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.NativeStandings, error) {
			return commissioner.FinalStandingsReading(t.Context(), bigLeagueID, r, s.nativeSources(bigLeagueID, year), year, at)
		})
		probes[fmt.Sprintf("stat-queue/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.StatQueue, error) {
			return commissioner.StatQueueReading(t.Context(), bigLeagueID, r, seasonStatFeed{store: s, season: year}, s.nativeSources(bigLeagueID, year), year, at)
		})
		probes[fmt.Sprintf("franchise-schedule/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.FranchiseSchedule, error) {
			return commissioner.FranchiseScheduleReading(bigLeagueID, team, year, r, nil, time.UTC, at)
		})
		probes[fmt.Sprintf("standings-adjustments/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.StandingsAdjustments, error) {
			return commissioner.StandingsAdjustmentsReading(r, year)
		})
		probes[fmt.Sprintf("title/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.BracketTitle, error) {
			return commissioner.TitleReading(r, year), nil
		})
		probes[fmt.Sprintf("player-changes/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.ScoreChanges, error) {
			return commissioner.PlayerScoreChanges(r, g.players[0].ID, year, 1)
		})
		probes[fmt.Sprintf("franchise-changes/%d", year)] = readingProbe(func(r []commissioner.Record) (commissioner.ScoreChanges, error) {
			return commissioner.FranchiseScoreChanges(r, team, year, 1)
		})
	}
}
