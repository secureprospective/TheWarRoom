// Command d27replay is the D27 replay proof: team needs and surplus read at each trade's date,
// from the model as it would have valued players then, against what league 14432's owners did.
// Read-only: it opens a COPY of a history database and saved MFL exports.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// bestN is league 14432's starter count, 21 in every replayed season.
const bestN = 21

type trade struct {
	at         time.Time
	year, week int // week 0 is before the season
	a, b       string
	aGave      []string // players only
	bGave      []string
	aPicks     int
	bPicks     int
}

type season struct {
	year    int
	listing normalize.Lookup
	rosters map[int]map[string][]string // week → franchise → players
	trades  []trade
}

func main() {
	dbPath := flag.String("db", "", "COPY of a history database holding the player record")
	dir := flag.String("dir", ".", "saved MFL exports: players_Y, rosters_Y_W, trades_Y")
	years := flag.String("seasons", "2023,2024,2025", "seasons replayed")
	flag.Parse()
	ctx := context.Background()
	pools, err := db.Open(ctx, *dbPath, history.SchemaVersion())
	if err != nil {
		fail(err)
	}
	defer func() { _ = pools.Close() }()
	reg, err := measures.Embedded()
	if err != nil {
		fail(err)
	}
	h := history.New(pools, reg)
	if err := h.Initialize(ctx); err != nil {
		fail(err)
	}
	r := &replay{h: h, values: map[string]map[string]value{}}
	for _, ys := range strings.Split(*years, ",") {
		y, err := strconv.Atoi(ys)
		if err != nil {
			fail(err)
		}
		s, err := load(ctx, *dir, y)
		if err != nil {
			fail(err)
		}
		r.seasons = append(r.seasons, s)
	}
	if err := r.run(ctx); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "d27replay:", err)
	os.Exit(1)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// many decodes MFL's list fields, which hold a bare object when the list has one entry.
func many[T any](raw json.RawMessage) []T {
	var list []T
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var one T
	if json.Unmarshal(raw, &one) == nil {
		return []T{one}
	}
	return nil
}

func load(ctx context.Context, dir string, y int) (*season, error) {
	s := &season{year: y, rosters: map[int]map[string][]string{}}
	body, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("players_%d.json", y)))
	if err != nil {
		return nil, fmt.Errorf("players export: %w", err)
	}
	raws, err := players.Parse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("players %d: %w", y, err)
	}
	if s.listing, err = normalize.NewLookup(raws); err != nil {
		return nil, fmt.Errorf("listing %d: %w", y, err)
	}
	for w := 1; w <= 14; w++ {
		var env struct {
			Rosters struct {
				Franchise json.RawMessage `json:"franchise"`
			} `json:"rosters"`
		}
		if err := readJSON(filepath.Join(dir, fmt.Sprintf("rosters_%d_%d.json", y, w)), &env); err != nil {
			return nil, err
		}
		week := map[string][]string{}
		for _, f := range many[struct {
			ID     string          `json:"id"`
			Player json.RawMessage `json:"player"`
		}](env.Rosters.Franchise) {
			for _, p := range many[struct {
				ID string `json:"id"`
			}](f.Player) {
				week[f.ID] = append(week[f.ID], p.ID)
			}
		}
		s.rosters[w] = week
	}
	return s, s.loadTrades(filepath.Join(dir, fmt.Sprintf("trades_%d.json", y)))
}

// kickoff is each season's first NFL game; a trade's week is the lineup window it falls in.
func kickoff(y int) time.Time {
	k := map[int]string{2023: "2023-09-07", 2024: "2024-09-05", 2025: "2025-09-04"}[y]
	t, _ := time.Parse("2006-01-02", k)
	return t
}

func (s *season) loadTrades(path string) error {
	var env struct {
		Transactions struct {
			Transaction json.RawMessage `json:"transaction"`
		} `json:"transactions"`
	}
	if err := readJSON(path, &env); err != nil {
		return err
	}
	firstClose := kickoff(s.year).Add(-48 * time.Hour)
	for _, t := range many[struct {
		Type, Franchise, Franchise2, Timestamp string
		Gave1                                  string `json:"franchise1_gave_up"`
		Gave2                                  string `json:"franchise2_gave_up"`
	}](env.Transactions.Transaction) {
		ts, err := strconv.ParseInt(t.Timestamp, 10, 64)
		if t.Type != "TRADE" || err != nil {
			continue
		}
		at := time.Unix(ts, 0)
		week := 0
		if !at.Before(firstClose) {
			week = int(at.Sub(firstClose).Hours()/(24*7)) + 1
		}
		tr := trade{at: at, year: s.year, week: week, a: t.Franchise, b: t.Franchise2}
		tr.aGave, tr.aPicks = split(t.Gave1)
		tr.bGave, tr.bPicks = split(t.Gave2)
		s.trades = append(s.trades, tr)
	}
	slices.SortFunc(s.trades, func(x, y trade) int { return x.at.Compare(y.at) })
	return nil
}

// split separates players from draft picks (DP_ this year's slots, FP_ future rounds).
func split(csv string) (ids []string, picks int) {
	for _, f := range strings.Split(csv, ",") {
		f = strings.TrimSpace(f)
		switch {
		case f == "":
		case strings.HasPrefix(f, "DP_"), strings.HasPrefix(f, "FP_"):
			picks++
		default:
			ids = append(ids, f)
		}
	}
	return ids, picks
}

// before is both franchises' rosters just before t: the roster export of the trade's week (week
// 1 for an offseason trade) with every later trade in that window undone, latest first. ok is
// false when a side did not hold what it gave: the export and the ledger disagree.
func (s *season) before(t trade) (map[string][]string, bool) {
	base := max(t.week, 1)
	ros := map[string][]string{}
	for fid, ps := range s.rosters[base] {
		ros[fid] = slices.Clone(ps)
	}
	for i := len(s.trades) - 1; i >= 0; i-- {
		x := s.trades[i]
		if x.at.Before(t.at) || max(x.week, 1) > base {
			continue
		}
		move(ros, x.b, x.a, x.aGave)
		move(ros, x.a, x.b, x.bGave)
	}
	return ros, holds(ros[t.a], t.aGave) && holds(ros[t.b], t.bGave)
}

func move(ros map[string][]string, from, to string, ids []string) {
	for _, id := range ids {
		if i := slices.Index(ros[from], id); i >= 0 {
			ros[from] = slices.Delete(ros[from], i, i+1)
			ros[to] = append(ros[to], id)
		}
	}
}

func holds(roster, want []string) bool {
	for _, id := range want {
		if !slices.Contains(roster, id) {
			return false
		}
	}
	return true
}
