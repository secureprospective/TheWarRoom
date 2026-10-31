package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/teamneeds"
)

type value struct {
	pos   string
	ppg   float64
	valid bool
}

type replay struct {
	h       *history.Store
	seasons []*season
	values  map[string]map[string]value // "season/lastWeek" → player → value
}

// asOf is the model pass a trade is judged with: before the season, last season's finished
// record; in season, this season through the week before the trade.
func asOf(t trade) (season, lastWeek int) {
	if t.week <= 1 {
		return t.year - 1, 0
	}
	return t.year, t.week - 1
}

// valuesAt runs the model once for (y, lastWeek) over every player on any roster that season.
// lastWeek 0 means y's full record.
func (r *replay) valuesAt(ctx context.Context, s *season, y, lastWeek int) (map[string]value, error) {
	key := fmt.Sprintf("%d/%d", y, lastWeek)
	if v, ok := r.values[key]; ok {
		return v, nil
	}
	start := time.Now()
	held, last, dropped, err := r.held(ctx, y, lastWeek)
	if err != nil {
		return nil, err
	}
	book, err := playerrecord.HeldBook(s.listing, held, last)
	if err != nil {
		return nil, fmt.Errorf("book %s: %w", key, err)
	}
	st := s.everyone()
	out := &capture{}
	runner, err := modelrun.New(st, fixedBook{book}, out, "d27replay")
	if err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	spec := modelrun.Spec{Season: y, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: last[y], Refresh: true}
	if _, err := runner.Run(ctx, spec); err != nil {
		return nil, fmt.Errorf("model %s: %w", key, err)
	}
	vals := map[string]value{}
	for _, sc := range out.run.Scores {
		vals[sc.MFLID] = value{pos: sc.Position, ppg: sc.DynastyPPG, valid: !sc.Unavailable}
	}
	say("model %s: %d players valued, %d in-season season-level/later rows withheld, %s",
		key, len(vals), dropped, time.Since(start).Round(time.Second))
	r.values[key] = vals
	return vals, nil
}

// held is the record as known after lastWeek of y (0: y finished): later weeks and y's
// season-level rows, which would carry later weeks, are withheld in season.
func (r *replay) held(ctx context.Context, y, lastWeek int) ([]model.Obs, map[int]int, int, error) {
	obs, err := modelrun.Observations(ctx, r.h, y-16, y, time.Now(), model.RuntimeMeasures())
	if err != nil {
		return nil, nil, 0, fmt.Errorf("observations %d: %w", y, err)
	}
	last := map[int]int{}
	for _, o := range obs {
		if o.Week > 0 {
			last[o.Season] = max(last[o.Season], o.Week)
		}
	}
	held := obs[:0:0]
	dropped := 0
	for _, o := range obs {
		if lastWeek > 0 && o.Season == y && (o.Week == 0 || o.Week > lastWeek) {
			dropped++
			continue
		}
		held = append(held, o)
	}
	if lastWeek > 0 {
		last[y] = lastWeek
	}
	return held, last, dropped, nil
}

// everyone is one franchise holding every player on any roster this season.
func (s *season) everyone() heldState {
	universe := map[string]bool{}
	for _, week := range s.rosters {
		for _, ps := range week {
			for _, p := range ps {
				universe[p] = true
			}
		}
	}
	st := heldState{}
	for p := range universe {
		st = append(st, state.PlayerState{MFLID: p, FranchiseID: "0001", RosterStatus: domain.RosterStatus("ROSTER")})
	}
	return st
}

// league reads every roster's needs and surplus with one pass's values.
func league(ros map[string][]string, vals map[string]value, pos func(string) string) teamneeds.League {
	in := map[string][]teamneeds.Player{}
	for fid, ps := range ros {
		for _, p := range ps {
			v := vals[p]
			position := v.pos
			if position == "" {
				position = pos(p)
			}
			in[fid] = append(in[fid], teamneeds.Player{ID: p, Position: position, Value: v.ppg, Valued: v.valid})
		}
	}
	return teamneeds.Compute(in, bestN)
}

func (s *season) pos(id string) string {
	if f, ok := s.listing.Facts(id); ok {
		return string(f.Position)
	}
	return "?"
}

// heldState is one franchise holding every player the replay values.
type heldState []state.PlayerState

func (h heldState) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{FranchiseID: "0001", Players: h}, true
}
func (h heldState) Roster(string) ([]state.PlayerState, bool) { return h, true }
func (h heldState) CapUsed(string) (domain.Money, bool)       { return 0, true }
func (h heldState) Player(id string) (state.PlayerState, bool) {
	for _, p := range h {
		if p.MFLID == id {
			return p, true
		}
	}
	return state.PlayerState{}, false
}
func (h heldState) Franchises() []string { return []string{"0001"} }

type fixedBook struct{ b playerrecord.Book }

func (f fixedBook) Book(context.Context, playerrecord.Query) (playerrecord.Book, error) {
	return f.b, nil
}

type capture struct{ run history.NewModelRun }

func (c *capture) WriteModelRun(_ context.Context, nr history.NewModelRun) (history.Run, bool, error) {
	c.run = nr
	return history.Run{ID: 1}, true, nil
}
