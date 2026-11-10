package playerrecord

import (
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type listing map[string]normalize.PlayerFacts

func (l listing) Facts(id string) (normalize.PlayerFacts, bool) {
	f, ok := l[id]
	return f, ok
}
func (l listing) IDs() []string { return slices.Sorted(maps.Keys(l)) }

func date(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func assemble(t *testing.T, l listing, h map[string]*model.Player) map[string]Record {
	t.Helper()
	records, err := Assemble(l, h, func(mfl string) (string, error) { return HeldID(mfl), nil })
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Record{}
	for _, r := range records {
		out[r.MFL] = r
	}
	return out
}

// DynastyProcess once tied a 2026 rookie DT to a 1978 receiver: MFL wins and the old record's
// identity is set aside, his college kept.
func TestAssembleSetsAsideAnotherPlayersHistory(t *testing.T) {
	l := listing{"0300": {Name: "Rookie", Position: domain.PosDT, DraftYear: 2026, HasDraftYear: true,
		HasBirthdate: true, Birthdate: date(2004, 1, 1).Unix()}}
	h := map[string]*model.Player{"0300": {Position: domain.PosWR, Birth: date(1953, 1, 9), Rookie: 1978, DraftPick: 12,
		Combine: map[string]float64{"forty": 4.4}, College: map[int]map[string]float64{2025: {"sacks": 9}}}}
	r := assemble(t, l, h)["0300"]
	if !r.Mislinked || !r.Birth.Equal(date(2004, 1, 1)) || r.Rookie != 2026 || r.Draft.Overall != 0 ||
		r.Draft.Status != Undrafted {
		t.Fatalf("record = %+v", r)
	}
	b, err := NewBook([]Record{r}, []model.Obs{
		{Player: "0300", Measure: "prior.forty", Value: 4.4},
		{Player: "0300", Season: 2025, Measure: "prior.college_sacks", Value: 9},
	}, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	pl := b.Player(r.ID)
	if len(pl.Combine) != 0 || pl.College[2025]["sacks"] != 9 || pl.Position != domain.PosDT || pl.Rookie != 2026 {
		t.Errorf("valued player = %+v", pl)
	}
}

// The record never mints a second identity: each link asks for its ID once and an error stops
// the whole assembly rather than leaving a player without one.
func TestAssembleAsksOnceAndStopsOnAnIdentityError(t *testing.T) {
	asked := map[string]int{}
	l := listing{"0001": {Position: domain.PosWR}}
	h := map[string]*model.Player{"0001": {}, "0002": {}}
	if _, err := Assemble(l, h, func(mfl string) (string, error) { asked[mfl]++; return HeldID(mfl), nil }); err != nil {
		t.Fatal(err)
	}
	if asked["0001"] != 1 || asked["0002"] != 1 || len(asked) != 2 {
		t.Errorf("asked %v", asked)
	}
	if _, err := Assemble(l, h, func(string) (string, error) { return "", errors.New("store down") }); err == nil {
		t.Error("an identity error must fail the assembly")
	}
}

// The league scores a player at its own position, so a history player MFL lists at a position the
// model scores moves to it (and into that position's scale); one MFL lists elsewhere, or does not
// list, keeps nflverse's.
func TestBookUsesTheLeaguesPosition(t *testing.T) {
	positions := map[string]domain.Position{
		"edge": domain.PosLB, "unknown": domain.PosDE, "flagged": domain.PosCB, "db": "", "same": domain.PosDT,
	}
	l := listing{
		"edge":    {Position: domain.PosDE},
		"flagged": {Position: domain.PosFlag},
		"db":      {Position: domain.PosS},
		"same":    {Position: domain.PosDT},
	}
	var obs []model.Obs
	h := map[string]*model.Player{}
	for id, pos := range positions {
		h[id] = &model.Player{Position: pos}
		text := map[domain.Position]string{domain.PosLB: "LB", domain.PosDE: "DE", domain.PosCB: "CB", domain.PosDT: "DT", "": "DB"}[pos]
		obs = append(obs, model.Obs{Player: id, Measure: "context.nfl_position", Text: text})
	}
	records, err := Assemble(l, h, func(mfl string) (string, error) { return HeldID(mfl), nil })
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewBook(records, obs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]domain.Position{"edge": domain.PosDE, "unknown": domain.PosDE, "flagged": domain.PosCB,
		"db": domain.PosS, "same": domain.PosDT}
	for mfl, pos := range want {
		id, ok := b.ID(mfl)
		if !ok {
			t.Fatalf("no record for %s", mfl)
		}
		if got := b.Data.Players[id].Position; got != pos {
			t.Errorf("%s at %q, want %q", mfl, got, pos)
		}
	}
}

// Production is filed under our ID; a fact whose link belongs to no one fails the book.
func TestBookKeysProductionByOurID(t *testing.T) {
	r := Record{ID: HeldID("0001"), MFL: "0001", Listed: true, Position: domain.PosWR}
	obs := []model.Obs{
		{Player: "0001", Season: 2025, Week: 1, Measure: "exposure.offense_snaps", Value: 30},
		{Player: "0002", Season: 2025, Week: 1, Measure: "exposure.offense_snaps", Value: 30},
	}
	b, err := NewBook([]Record{r}, obs, map[int]int{2025: 17}, func(mfl string) (string, bool) { return "player.other", mfl == "0002" })
	if err != nil {
		t.Fatal(err)
	}
	for id := range b.Data.Seasons {
		if id != r.ID && id != "player.other" {
			t.Errorf("season filed under %q", id)
		}
	}
	if b.Data.Seasons[r.ID][2025].Games() != 1 || b.Data.Seasons["player.other"][2025].Games() != 1 {
		t.Errorf("seasons = %+v", b.Data.Seasons)
	}
	if _, err := NewBook([]Record{r}, obs, nil, func(string) (string, bool) { return "", false }); err == nil {
		t.Error("a fact nobody owns must fail the book")
	}
	if _, err := NewBook([]Record{r, r}, nil, nil, nil); err == nil {
		t.Error("two records for one player must fail the book")
	}
}
