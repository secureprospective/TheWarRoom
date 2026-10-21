package main

import (
	"context"
	"math"
	"strconv"
	"time"

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

// PowerRow is one franchise's power-ranking row: the blended score, its two components, and
// MFL's report columns from the same standings call. Display values only. RankDelta is the move
// since the board built from the previous model run; DeltaOK is false when there is none.
type PowerRow struct {
	Rank        int    `json:"rank"`
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`
	RankDelta   int    `json:"rankDelta"`
	DeltaOK     bool   `json:"deltaOK"`

	PowerScore float64 `json:"powerScore"` // weighted z-blend, scaled to [0,1]
	RosterZ    float64 `json:"rosterZ"`    // 0 = league average
	MFLPerfZ   float64 `json:"mflPerfZ"`
	AgeZ       float64 `json:"ageZ"`

	RosterValue float64 `json:"rosterValue"`
	Results     float64 `json:"results"` // what the blend read, in [0,1]; Performance names it
	Age         float64 `json:"age"`     // the counted players' value-weighted age; 0 when none is known

	// Context columns, display only (m2service.Outlook); each Has* says the column was read.
	CapRoom float64 `json:"capRoom"`
	DeadCap float64 `json:"deadCap"`
	HasCap  bool    `json:"hasCap"`
	Luck    float64 `json:"luck"`
	HasLuck bool    `json:"hasLuck"`
	ProjW   float64 `json:"projW"`
	ProjL   float64 `json:"projL"`
	HasProj bool    `json:"hasProj"`

	// MFL report columns, from leagueStandings.
	H2HW     int     `json:"h2hW"`
	H2HL     int     `json:"h2hL"`
	H2HT     int     `json:"h2hT"`
	AllPlayW int     `json:"allPlayW"`
	AllPlayL int     `json:"allPlayL"`
	AllPlayT int     `json:"allPlayT"`
	PF       float64 `json:"pf"`
	PA       float64 `json:"pa"`
	PP       float64 `json:"pp"`
	Pwr      float64 `json:"pwr"`
	AltPwr   float64 `json:"altPwr"`
}

// PowerRankingsResult echoes the view, weights, aggregation mode and N actually applied, so the
// controls and the rows never disagree. WeightAuto is true when the roster weight came from the
// weeks played rather than the slider. Zero rows means the league has no model run yet.
// WeeksScored is how many weeks the MFL standings hold, read from the head-to-head records, and
// SeasonWeeks the league's last regular-season week: the standings are final when they meet. Phase is separate from
// Freshness: an offseason board is fresh data about a finished season. An empty Phase means the
// phase read failed, which affects the label only.
type PowerRankingsResult struct {
	OK         bool    `json:"ok"`
	Error      string  `json:"error"`
	Label      string  `json:"label"`
	Season     int     `json:"season"`
	View       string  `json:"view"`
	Weight     float64 `json:"weight"`
	WeightAuto bool    `json:"weightAuto"`
	AgeWeight  float64 `json:"ageWeight"`
	// ModelRunID is the model run ranked; PreviousRunID the one the deltas compare with (0: none).
	ModelRunID    int64      `json:"modelRunID"`
	PreviousRunID int64      `json:"previousRunID"`
	AggMode       string     `json:"aggMode"`
	StarterN      int        `json:"starterN"`
	Freshness     Freshness  `json:"freshness"`
	Performance   string     `json:"performance"` // which result the blend read: all-play, points for, none
	WeeksScored   int        `json:"weeksScored"`
	SeasonWeeks   int        `json:"seasonWeeks"`
	Rows          []PowerRow `json:"rows"`
}

// The two M2 views (plan Stage 8): this season ranks on-field-now blended with the standings;
// the franchise ranks dynasty and the roster's age, because this season's record says little
// about the next five. Each view counts the roster its own way by default: this season the
// lineup the rules allow, the franchise the whole roster, whose young bench grows into starters
// (docs/modules/M2_Power_Ranking_Factors.md).
const (
	ViewSeason    = "season"
	ViewFranchise = "franchise"
)

func (a *App) powerRankings(ctx context.Context, standings []leaguestandings.RawStanding, fresh Freshness,
	schedule func(context.Context) []leagueschedule.RawScheduleWeek, weight float64, autoWeight bool, mode, view string,
) PowerRankingsResult {
	src := scoringapp.Source{History: a.core.History, Season: a.core.Season, State: a.core.League.Reader, Rules: a.core.Rulebook,
		Ages: a.ages, SeasonWeeks: a.seasonWeeks, DeadCap: a.core.League.DeadCap}
	return wirePower(src.Build(ctx, standings, fresh, schedule, weight, autoWeight, mode, view))
}

func defaultAgg(view string) string { return scoringapp.DefaultAgg(view) }

// remainingSchedule is the league schedule for the projected record, nil when it cannot be read:
// a missing column must not stop the board.
func (a *App) remainingSchedule(ctx context.Context) []leagueschedule.RawScheduleWeek {
	weeks, _, err := a.leagueScheduleOrArchive(ctx)
	if err != nil {
		return nil
	}
	return weeks
}

// ages reads each player's age today from the players directory. When the directory cannot be
// read every age is unknown (NaN), and the board counts none.
func (a *App) ages(ctx context.Context) func(mflID string) float64 {
	lk, err := a.directory(ctx)
	today := time.Now()
	return func(mflID string) float64 {
		if err != nil {
			return math.NaN()
		}
		f, ok := lk.Facts(mflID)
		if !ok || !f.HasBirthdate {
			return math.NaN()
		}
		return today.Sub(time.Unix(f.Birthdate, 0)).Hours() / (24 * 365.2425)
	}
}

// seasonWeeks is the league's last regular-season week, 0 when unreadable.
func (a *App) seasonWeeks() int {
	last, _ := a.core.Rulebook.GetSetting("lastRegularSeasonWeek")
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}

func wirePower(b season.PowerRankingsResult) PowerRankingsResult {
	out := PowerRankingsResult{OK: b.OK, Error: b.Error, Label: b.Label, Season: b.Season, View: b.View,
		Weight: b.Weight, WeightAuto: b.WeightAuto, AgeWeight: b.AgeWeight, ModelRunID: b.ModelRunID,
		PreviousRunID: b.PreviousRunID, AggMode: b.AggMode, StarterN: b.StarterN, Freshness: b.Freshness,
		Performance: b.Performance, WeeksScored: b.WeeksScored, SeasonWeeks: b.SeasonWeeks}
	if b.Rows != nil {
		out.Rows = make([]PowerRow, len(b.Rows))
		for i, r := range b.Rows {
			out.Rows[i] = PowerRow{Rank: r.Rank, FranchiseID: r.FranchiseID, Name: r.Name,
				RankDelta: r.RankDelta, DeltaOK: r.DeltaOK, PowerScore: r.PowerScore, RosterZ: r.RosterZ,
				MFLPerfZ: r.MFLPerfZ, AgeZ: r.AgeZ, RosterValue: r.RosterValue, Results: r.Results, Age: r.Age,
				CapRoom: r.CapRoom, DeadCap: r.DeadCap, HasCap: r.HasCap, Luck: r.Luck, HasLuck: r.HasLuck,
				ProjW: r.ProjW, ProjL: r.ProjL, HasProj: r.HasProj,
				H2HW: r.H2HW, H2HL: r.H2HL, H2HT: r.H2HT, AllPlayW: r.AllPlayW, AllPlayL: r.AllPlayL,
				AllPlayT: r.AllPlayT, PF: r.PF, PA: r.PA, PP: r.PP, Pwr: r.Pwr, AltPwr: r.AltPwr}
		}
	}
	return out
}

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
