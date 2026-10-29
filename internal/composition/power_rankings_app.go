package composition

import (
	"context"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	scoringapp "github.com/secureprospective/TheWarRoom/internal/app/scoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// m2Timeout bounds the power-rankings read behind the race board (one MFL standings fetch plus a
// read of two model runs) and the Target acts that borrowed it.
const m2Timeout = 120 * time.Second

// m4Timeout bounds the Target reads whose first call after a cold start may fetch the players
// directory.
const m4Timeout = 120 * time.Second

type PowerRow = season.PowerRow
type PowerRankingsResult = season.PowerRankingsResult

// The two M2 views (plan Stage 8): this season ranks on-field-now blended with the standings;
// the franchise ranks dynasty and the roster's age, because this season's record says little
// about the next five. Both views count the starting lineup by default; future picks add to
// franchise value only.
const (
	ViewSeason    = "season"
	ViewFranchise = "franchise"
)

func (a *App) powerRankings(ctx context.Context, standings []leaguestandings.RawStanding, fresh Freshness,
	schedule func(context.Context) []leagueschedule.RawScheduleWeek, weight float64, autoWeight bool, mode, view string,
) PowerRankingsResult {
	return wirePower(a.powerSource().Build(ctx, standings, fresh, schedule, weight, autoWeight, mode, view))
}

func (a *App) powerSource() scoringapp.Source {
	return scoringapp.Source{History: a.core.History, Season: a.core.Season, State: a.core.League.Reader, Rules: a.core.Rulebook,
		Picks: a.powerPicks, Ages: a.ages, SeasonWeeks: a.seasonWeeks, DeadCap: a.core.League.DeadCap}
}

func defaultAgg(view string) string { return scoringapp.DefaultAgg(view) }

// ages reads each player's age today from the players directory. When the directory cannot be
// read every age is unknown (NaN), and the board counts none.
func (a *App) ages(ctx context.Context) func(mflID string) float64 {
	lk, err := a.directory(ctx)
	today := time.Now()
	return scoringapp.PlayerAges(lk, err, today)
}

// seasonWeeks is the league's last regular-season week, 0 when unreadable.
func (a *App) seasonWeeks() int {
	last, _ := a.core.Rulebook.GetSetting("lastRegularSeasonWeek")
	return scoringapp.SeasonWeeks(last)
}

func wirePower(b season.PowerRankingsResult) PowerRankingsResult { return b }

// m1Timeout bounds reads of history reports.
const m1Timeout = 120 * time.Second

func (a *App) loadSeasonPoints(ctx context.Context) (string, error) {
	src := scoringapp.PointsSource{History: a.core.History, Season: a.core.Season,
		Fetch: func(ctx context.Context, year string, season int) (measures.Batch, error) {
			return playerscores.Fetch(ctx, a.core.MFLClient, year, ingestion.LeagueID, season)
		}}
	return src.Load(ctx) //nolint:wrapcheck // the service preserves the existing app error labels
}

// describeMeasures pairs each measure name with its dictionary meaning.
func (a *App) describeMeasures(names []string) []MissingMeasure {
	out := make([]MissingMeasure, len(names))
	for i, n := range names {
		m, _ := a.core.History.Registry().Measure(n)
		out[i] = MissingMeasure{Name: n, Meaning: m.Meaning}
	}
	return out
}

// MissingMeasure is a measure the board's model reads that no active source fed when it ran.
type MissingMeasure struct {
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}

func (a *App) powerPicks(ctx context.Context, run history.Run) m2service.PickCapital {
	return marketapp.PowerPicks(ctx, run, marketapp.PowerPickSource{Native: a.nativeValueSource(), Params: a.core.Params,
		Inventory: a.core.TradeFuturePicks(), Owners: a.core.League.Reader().Franchises(), At: a.timeNow()})
}

func (a *App) pickSettingChanged(key string, ok bool) {
	src := a.powerSource()
	src.Ages = func(context.Context) func(string) float64 {
		a.core.LookupMu.Lock()
		lookup := a.core.Lookup
		a.core.LookupMu.Unlock()
		return scoringapp.PlayerAges(lookup, nil, a.timeNow())
	}
	scoringapp.PickSettingChanged(a.core.Ctx, key, ok, scoringapp.PickSettingsSource{Lock: a.core.RefreshMu.Lock,
		Unlock: a.core.RefreshMu.Unlock, Timeout: m2Timeout, Model: a.publishAssetModel, Held: a.raceHeld.Snapshot, Board: src,
		Publish: func(held season.Held) { a.raceHeld.PublishStandings(held.Standings, held.Power, true) }})
}
