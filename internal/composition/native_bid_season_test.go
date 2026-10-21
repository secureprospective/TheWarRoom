package composition

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// bidBracketScores supplies complete, distinct native points without any outside service.
type bidBracketScores struct{}

func (bidBracketScores) Lineup(_ context.Context, _ []commissioner.Record, franchise string, _ int) (commissioner.WeekLineup, error) {
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: strings.TrimPrefix(franchise, "franchise.")}}}, nil
}
func (bidBracketScores) PlayerScore(_ context.Context, _ []commissioner.Record, player string, _ int) (commissioner.NativePlayerScore, error) {
	n, err := strconv.Atoi(player)
	if err != nil {
		return commissioner.NativePlayerScore{}, err
	}
	points := leaguerules.Decimal(n * 10000)
	return commissioner.NativePlayerScore{Status: "Complete", Position: "LB", Points: &points}, nil
}

func TestNativeBidPlayoffStandingComesFromNativeBracket(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	for _, rule := range []struct{ field, value string }{
		{"league.conferences", "1"}, {"schedule.regularWeeks", "1"}, {"lineup.invalidPenalty", `"none"`},
		{"playoffs.divisionWinners", "0"}, {"playoffs.wildCards", "7"},
	} {
		f.set(t, rule.field, rule.value, "now")
	}
	f.command(t, commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, "league")
	f.command(t, commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}, "league")
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	bracket, err := commissioner.NativeBracketReading(t.Context(), legalityLeague, records, bidBracketScores{}, 2026, f.now, commissioner.LineupFacts{})
	if err != nil || len(bracket.Fields) == 0 {
		t.Fatalf("native field: %s %s %v", bracket.Status, bracket.Words, err)
	}
	directory, err := commissioner.NativeDirectoryReading(legalityLeague, records, f.now)
	if err != nil {
		t.Fatal(err)
	}
	field := nativeapp.PlayoffField(bracket, directory)
	in := bracket.Fields[0].Seeds[0].Franchise
	if field[in] != nativecheck.PlayoffIn {
		t.Fatal("a playoff team was treated as out of the field")
	}
	out := ""
	for _, team := range directory.Franchises {
		if field[team.Franchise] == nativecheck.PlayoffOut {
			out = team.Franchise
			break
		}
	}
	if out == "" {
		t.Fatal("no excluded team")
	}
	policy, err := nativecheck.NewBidPolicy(leaguerules.LegacyNFL())
	if err != nil {
		t.Fatal(err)
	}
	proposal := nativecheck.BidProposal{Salary: moneyOf(5 * million), Years: 1}
	check := func(franchise string) string {
		return policy.Check(proposal, nativecheck.BidFacts{Season: nativecheck.BidSeason{Week: 14, WeekKnown: true}, Playoff: field[franchise]}).Refusal
	}
	if refusal := check(in); refusal != "" {
		t.Fatalf("playoff team's bid refused: %s", refusal)
	}
	if refusal := check(out); !strings.Contains(refusal, "yours is not one") {
		t.Fatalf("excluded team's bid: %s", refusal)
	}
	unknown := nativeapp.PlayoffField(commissioner.NativeBracket{}, directory)
	if len(unknown) != 0 {
		t.Fatal("an unavailable field invented exclusions")
	}
	if refusal := policy.Check(proposal, nativecheck.BidFacts{Season: nativecheck.BidSeason{Week: 14, WeekKnown: true}}).Refusal; !strings.Contains(refusal, "cannot tell whether yours is one") {
		t.Fatal(refusal)
	}
}

func TestNativeBidOffseasonAndMissingInSeasonWeek(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	// The first recorded rollover starts 2027; absent schedules are not permission to bid in October.
	f.set(t, "calendar.rollover", `"2027-03-01T00:00:00Z"`, "now")
	f.now = time.Date(2027, 5, 1, 12, 0, 0, 0, time.UTC)
	f.app.core.History.SetBidEnvironment(appBidEnv{f.app})
	env, err := (appBidEnv{f.app}).BidEnvironment(t.Context(), legalityLeague, f.now)
	if err != nil || !env.Season.Offseason {
		t.Fatalf("May with no schedule: %+v %v", env, err)
	}
	f.accepted(t, "franchise.01", "5004", 5*million, 2)
	reading := f.bids(t, "owner.one", "franchise.01")
	if !strings.Contains(strings.Join(reading.Notes, " "), "offseason") {
		t.Fatal("reading does not explain offseason limits")
	}
	f.now = time.Date(2027, 10, 1, 12, 0, 0, 0, time.UTC)
	f.refused(t, "franchise.01", "5002", 5*million, 2, "holds no NFL week")
	kickoff := time.Date(2027, 9, 10, 17, 0, 0, 0, time.UTC)
	held := []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Kickoff: kickoff}}}}
	phase := nativeapp.WeekAt(held, kickoff.Add(-time.Microsecond))
	if !phase.Offseason {
		t.Fatal("before the first held kickoff is not offseason")
	}
	if phase := nativeapp.WeekAt(held, kickoff); phase.Offseason || !phase.WeekKnown {
		t.Fatal("kickoff did not end offseason")
	}
	policy, err := nativecheck.NewBidPolicy(leaguerules.LegacyNFL())
	if err != nil {
		t.Fatal(err)
	}
	// An offseason flag must bypass both phase-week limits, not merely the missing-week refusal.
	verdict := policy.Check(nativecheck.BidProposal{Salary: moneyOf(5 * million), Years: 2}, nativecheck.BidFacts{
		Season: nativecheck.BidSeason{Week: 18, Offseason: true}, Playoff: nativecheck.PlayoffOut})
	if verdict.Refusal != "" {
		t.Fatal(verdict.Refusal)
	}
}

func TestNativeBidOffseasonBeforeHeldKickoffAndUnknownPlayoffField(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	kickoff := f.now.Add(time.Hour)
	heldNativeWeek(t, f.app, 14, kickoff, false)
	f.app.core.History.SetBidEnvironment(appBidEnv{f.app})
	f.accepted(t, "franchise.01", "5004", 5*million, 2)
	f.now = kickoff
	env, err := (appBidEnv{f.app}).BidEnvironment(t.Context(), legalityLeague, f.now)
	if err != nil || env.Season.Offseason || env.Season.Week != 14 || !env.Season.WeekKnown {
		t.Fatalf("held kickoff did not end offseason: %+v %v", env, err)
	}
	if env.Playoff["franchise.01"] != nativecheck.PlayoffUnknown {
		t.Fatal("unsettled native bracket invented a field")
	}
	f.refused(t, "franchise.01", "5004", 5*million, 1, "cannot tell whether yours is one")
}

func TestNativeBidUndatedOffseasonUsesHeldRolloverAndLeagueZone(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "calendar.rollover", `"2027-03-01T00:00:00Z"`, "now")
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		instant string
		want    bool
	}{
		{"2027-02-28T23:59:59Z", false},
		{"2027-03-01T00:00:00Z", true},
		{"2027-09-01T03:59:59Z", true},
		{"2027-09-01T04:00:00Z", false},
	} {
		at, err := time.Parse(time.RFC3339, c.instant)
		if err != nil {
			t.Fatal(err)
		}
		got, err := commissioner.BidOffseasonAt(records, at, 2027)
		if err != nil || got != c.want {
			t.Fatalf("offseason at %s: %t %v", c.instant, got, err)
		}
	}
}
