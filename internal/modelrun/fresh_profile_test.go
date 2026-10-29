package modelrun

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

func TestFreshProfileMissingPointsAreExcludedNotZero(t *testing.T) {
	h, _, dir := league()
	// A fresh import holds snap counts before the league-specific points arrive.
	h.feats[2025] = slices.DeleteFunc(h.feats[2025], func(f history.Feature) bool { return f.Measure == "outcome.fantasy_points" })
	spec := Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17}
	book, err := (Held{dir, h}).Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := newPass(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := book.ID("0200")
	_, _, why := ps.value(id)
	if why == "" {
		t.Fatal("missing points became a computed zero")
	}
	if ps.scale(domain.PosWR, 2025).Len() != 0 {
		t.Fatal("snap-only seasons became a zero-valued scale")
	}
}

func TestMissingPlayerPointsStayOutOfPublishedRun(t *testing.T) {
	h, st, dir := league()
	h.feats[2025] = slices.DeleteFunc(h.feats[2025], func(f history.Feature) bool {
		return f.PlayerID == "0200" && f.Measure == "outcome.fantasy_points"
	})
	runner, err := New(st, Held{dir, h}, h, "fresh-profile")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := runner.Run(t.Context(), Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(rep.Excluded, func(e Exclusion) bool {
		return e.MFLID == "0200" && e.Reason == "league points for 2025 are not ready"
	}) {
		t.Fatalf("missing reason: %+v", rep)
	}
	for _, sc := range h.written[0].Scores {
		if sc.MFLID == "0200" {
			t.Fatalf("published missing points as %+v", sc)
		}
	}
}

func TestCompleteInputReplay(t *testing.T) {
	h, _, dir := league()
	for i := range 25 {
		id := fmt.Sprintf("%04d", 200+i)
		dir[id] = normalize.PlayerFacts{Name: id, Position: domain.PosWR}
	}
	spec := Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17}
	book, err := (Held{dir, h}).Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := newPass(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	// Freeze the complete-input calculation independently of the runtime admission check.
	for _, mfl := range dir.IDs() {
		id, _ := book.ID(mfl)
		sc, in, why := ps.value(id)
		if why != "" {
			t.Fatal(why)
		}
		pl := book.Player(id)
		want := ps.params[domain.PosWR].Value(&pl, in.Past, in.Current, ps.horizon, book.Data.Scales()[domain.PosWR][2025])
		if sc.Value != want {
			t.Fatalf("%s moved: %+v != %+v", mfl, sc.Value, want)
		}
	}
	t.Logf("%d complete-input players, zero movers", len(dir))
}
