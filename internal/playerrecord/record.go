// Package playerrecord is the player record: one record per permanent player ID (player.<hex>),
// the one real person behind every league, feed and outside ID. It holds who he is — his NFL
// draft capital, his bio and his outside IDs — and it is the door the value model reads his
// production through, so every per-player input the model reads is keyed by our ID.
//
// Feeds write facts into the record; the record resolves them (which source wins a field) and
// never mints a second identity for a player it already holds. It is pure: the store persists
// it (internal/store/playerrecord) and callers hand it the facts.
package playerrecord

import (
	"fmt"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// DraftStatus is what the record knows of a player's NFL draft. Undrafted is a real value: the
// player entered the league without being picked.
type DraftStatus string

const (
	Drafted      DraftStatus = "drafted"
	Undrafted    DraftStatus = "undrafted"
	DraftUnknown DraftStatus = "unknown"
)

// Draft is a player's NFL draft capital. Round, Pick and Team are MFL's; Overall is nflverse's
// overall pick, or derived from MFL's round and pick when nflverse has none; 0 when neither.
type Draft struct {
	Status  DraftStatus
	Year    int // the draft he was picked in, or entered the league after undrafted; 0 when unknown
	Round   int
	Pick    int // the pick within the round
	Overall int
	Team    string
}

// Record is one player.
type Record struct {
	ID       string          // our permanent ID
	MFL      string          // his MFL link
	Listed   bool            // MFL's players database lists him (not a team aggregate or unknown)
	Name     string          // as MFL lists him
	Position domain.Position // the league's position for him: MFL's when listed, else nflverse's
	Birth    time.Time       // zero when unknown
	College  string
	Rookie   int // first NFL season; 0 when unknown
	Career   Career
	Draft    Draft
	// Mislinked is set when history's facts under his MFL link are another player's (their first
	// NFL season is far from MFL's draft year); those identity facts are set aside.
	Mislinked bool
	Sources   map[string]string // field → the feed it came from
}

// Field names in Sources.
const (
	FieldPosition = "position"
	FieldBirth    = "birth"
	FieldRookie   = "rookie"
	FieldDraft    = "draft"
	FieldOverall  = "overall"
	FieldBio      = "bio"
)

// Feed names in Sources.
const (
	SourceMFL      = "mfl"
	SourceNFLverse = "nflverse"
	SourceDerived  = "derived" // computed from other record facts (an overall pick from round and pick)
)

// Career holds supplied NFL career counts. Nil means unknown, not zero. StartingSeasons
// is the feed's seasons as a primary starter; no held feed supplies game starts or wins.
type Career struct {
	StartingSeasons *int `json:"startingSeasons,omitempty"`
	Wins            *int `json:"wins,omitempty"`
	AllPro          *int `json:"allPro,omitempty"`
	ProBowls        *int `json:"proBowls,omitempty"`
}

// CareerMeasures are record facts only, never valuation inputs.
func CareerMeasures() []string {
	return []string{"context.career_starting_seasons", "context.career_all_pro", "context.career_pro_bowls"}
}

// AddCareer attaches supplied career facts without changing the valuation's player population.
func AddCareer(records []Record, obs []model.Obs) error {
	byMFL := make(map[string]*Record, len(records))
	for i := range records {
		byMFL[records[i].MFL] = &records[i]
	}
	for _, o := range obs {
		r := byMFL[o.Player]
		if r == nil || r.Mislinked {
			continue
		}
		var field **int
		switch o.Measure {
		case "context.career_starting_seasons":
			field = &r.Career.StartingSeasons
		case "context.career_all_pro":
			field = &r.Career.AllPro
		case "context.career_pro_bowls":
			field = &r.Career.ProBowls
		default:
			continue
		}
		if o.Season != 0 || o.Week != 0 || o.Value < 0 || o.Value > 1000 || math.Trunc(o.Value) != o.Value {
			return fmt.Errorf("playerrecord: invalid career count %s for %s: %v", o.Measure, o.Player, o.Value)
		}
		n := int(o.Value)
		*field = &n
		r.Sources[o.Measure] = SourceNFLverse
	}
	return nil
}
