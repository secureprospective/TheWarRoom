// Package season holds shared season view models.
package season

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

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

	PickValue   float64 `json:"pickValue"`
	HasPicks    bool    `json:"hasPicks"`
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
	PickWeight float64 `json:"pickWeight"`
	// ModelRunID is the model run ranked; PreviousRunID the one the deltas compare with (0: none).
	ModelRunID    int64            `json:"modelRunID"`
	PreviousRunID int64            `json:"previousRunID"`
	AggMode       string           `json:"aggMode"`
	StarterN      int              `json:"starterN"`
	Freshness     domain.Freshness `json:"freshness"`
	Performance   string           `json:"performance"` // which result the blend read: all-play, points for, none
	WeeksScored   int              `json:"weeksScored"`
	SeasonWeeks   int              `json:"seasonWeeks"`
	Rows          []PowerRow       `json:"rows"`
}

// The two M2 views (plan Stage 8): this season ranks on-field-now blended with the standings;
// the franchise ranks dynasty and the roster's age, because this season's record says little
// about the next five. Both views count the starting lineup by default; future picks add to
// franchise value only.
const (
	ViewSeason    = "season"
	ViewFranchise = "franchise"
)

// DefaultAgg counts the legal starting lineup in both views, so unusable depth cannot inflate value.
func DefaultAgg(string) string { return m2service.AggLineup }

// PowerKey keeps the default boards available to shared season readers.
func PowerKey(view, mode string) string {
	if mode == "" || mode == DefaultAgg(view) {
		return view
	}
	return view + ":" + mode
}
