package modelrun

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
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
		return e.MFLID == "0200" && strings.Contains(e.Reason, "league points for 2025 are not ready")
	}) {
		t.Fatalf("missing reason: %+v", rep)
	}
	for _, sc := range h.written[0].Scores {
		if sc.MFLID == "0200" && (!sc.Unavailable || sc.Reason == "") {
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

func TestUnusableSeasonsAreSkipped(t *testing.T) {
	for _, tc := range []struct {
		name          string
		past, current bool
		wantReason    string
		unavailable   bool
	}{
		{"one bad season", false, true, "league points for 2025", false},
		{"all bad seasons", false, false, "league points for 2025", true},
		{"missing games", true, true, "games played for 2024", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, dir := league()
			spec := Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17}
			book, err := (Held{dir, h}).Book(t.Context(), BookQuery(spec))
			if err != nil {
				t.Fatal(err)
			}
			id, _ := book.ID("0200")
			book.Data.Seasons[id] = map[int]*model.Season{
				2025: {Points: map[int]float64{1: 0, 2: 0, 3: 0, 4: 0}, HasTotal: tc.past, Total: 40},
				2026: {Points: map[int]float64{1: 0}, HasTotal: tc.current, Total: 12},
			}
			if tc.name == "missing games" {
				book.Data.Seasons[id][2024] = &model.Season{HasTotal: true, Total: 10}
			}
			ps, err := newPass(book, spec)
			if err != nil {
				t.Fatal(err)
			}
			sc, in, why := ps.value(id)
			if tc.unavailable {
				if !strings.Contains(why, tc.wantReason) {
					t.Fatalf("reason %q", why)
				}
				return
			}
			if why != "" || !sc.ReducedInputs || !strings.Contains(sc.Reason, tc.wantReason) {
				t.Fatalf("score %+v, reason %q", sc, why)
			}
			if !tc.past && len(in.Past) != 0 {
				t.Fatalf("bad season fed projection: %+v", in.Past)
			}
			if sc.NowPPG <= 0 || sc.DynastyPPG <= 0 {
				t.Fatalf("lost usable evidence: %+v", sc)
			}
		})
	}
}

func TestAllUnavailableRunPublishesReasonsInsteadOfRetainingOldValues(t *testing.T) {
	h, st, dir := league()
	st.roster = st.roster[:1]
	h.feats[2025] = slices.DeleteFunc(h.feats[2025], func(f history.Feature) bool { return f.PlayerID == "0200" && f.Measure == "outcome.fantasy_points" })
	runner, err := New(st, Held{dir, h}, h, "all-unavailable")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := runner.Run(t.Context(), Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17})
	if err != nil || rep.Scored != 0 || len(rep.Excluded) != 1 || len(h.written) != 1 {
		t.Fatal(rep, err)
	}
	scores := h.written[0].Scores
	if len(scores) != 1 || !scores[0].Unavailable || !strings.Contains(scores[0].Reason, "league points for 2025") {
		t.Fatal(scores)
	}
}

func TestZeroGameZeroPointsKeepsTheRookiesPriorValue(t *testing.T) {
	h, _, dir := league()
	spec := Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17}
	book, err := (Held{dir, h}).Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	id, _ := book.ID("0300")
	book.Data.Seasons[id] = map[int]*model.Season{2026: {HasTotal: true, Total: 0}}
	ps, err := newPass(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	sc, in, why := ps.value(id)
	if why != "" || sc.DynastyPPG <= 0 || len(in.Past) != 0 || in.Current.Games != 0 || sc.Unavailable || !sc.ReducedInputs || !strings.Contains(sc.Reason, "no games recorded for 2026") {
		t.Fatal(sc, in, why)
	}
}
