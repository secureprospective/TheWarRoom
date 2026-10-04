package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// The fixture registry: one measure fed by source "alpha"; "beta" exists but feeds nothing until
// the source-gain drill adds its mapping row.
const (
	fixtureMeasures = "measure,grain,unit,positions,meaning\n" +
		"outcome.tackles_solo,season,count,DE DT LB CB S,Solo tackles.\n" +
		"availability.game_status,week,text,all,Game-day injury designation.\n" +
		"outcome.sacks,week,count,DE DT LB CB S,Sacks.\n" +
		"prior.forty,player,seconds,all,40-yard dash.\n"
	fixtureSources = "source,name,status,max_age_days,host,path_prefix\n" +
		"alpha,Alpha stats,active,7,127.0.0.1,\n" +
		"beta,Beta stats,active,7,beta.example.com,\n"
	fixtureFields = "source,field,measure,priority\n" +
		"alpha,Solo,outcome.tackles_solo,1\n" +
		"alpha,Status,availability.game_status,1\n" +
		"alpha,Sacks,outcome.sacks,1\n" +
		"alpha,Forty,prior.forty,1\n"
	betaField = "beta,soloTackles,outcome.tackles_solo,2\n"
)

func t0() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) }

func registry(t *testing.T, extraFields string) *measures.Registry {
	t.Helper()
	reg, err := measures.Load(fstest.MapFS{
		"measures.csv":      {Data: []byte(fixtureMeasures)},
		"sources.csv":       {Data: []byte(fixtureSources)},
		"source_fields.csv": {Data: []byte(fixtureFields + extraFields)},
		"feeds.csv":         {Data: []byte("feed,source,url,first_season,id_column,id_type,season_column,week_column,filter\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// clock is a settable store clock.
type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func newStore(t *testing.T) (*Store, *clock, *db.Pools) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	return reopen(t, pools, registry(t, ""))
}

// reopen starts a store over existing pools, as a new release of the app would.
func reopen(t *testing.T, pools *db.Pools, reg *measures.Registry) (*Store, *clock, *db.Pools) {
	t.Helper()
	s := New(pools, reg)
	c := &clock{now: t0()}
	s.SetClock(c.Now)
	if err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, c, pools
}

func count(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.pools.Read().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func solo(source, field, id, raw string) measures.Batch {
	return measures.Batch{Source: source, Facts: []measures.Fact{
		{IDType: measures.IDTypeMFL, ID: id, Season: 2025, Field: field, Raw: raw}}}
}

func features(t *testing.T, s *Store, asOf time.Time) []Feature {
	t.Helper()
	got, err := s.Features(context.Background(), FeatureQuery{
		AsOf: asOf, Season: 2025, Measures: []string{"outcome.tackles_solo"}})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestFetchArchivesOnceAndLoadIsIdempotent(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	const body = "player,Solo\n13604,41\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()
	client := &http.Client{Transport: &archive.Transport{Sink: s}}

	for range 2 {
		resp, err := client.Get(srv.URL + "/tackles.csv?apikey=hunter2")
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if string(got) != body {
			t.Fatalf("caller read %q", got)
		}
		if _, err := s.Ingest(ctx, solo("alpha", "Solo", "13604", "41")); err != nil {
			t.Fatal(err)
		}
		c.now = c.now.Add(time.Hour)
	}

	if n := count(t, s, "raw_archive"); n != 1 {
		t.Errorf("raw_archive rows = %d, want 1 (same body stored once)", n)
	}
	if n := count(t, s, "fetch_log WHERE source = 'alpha' AND status = 200"); n != 2 {
		t.Errorf("fetch_log rows for alpha = %d, want 2 (every fetch logged)", n)
	}
	if n := count(t, s, "fetch_log WHERE url LIKE '%hunter2%'"); n != 0 {
		t.Error("a credential in the query string reached the fetch log")
	}
	if n := count(t, s, "observations"); n != 1 {
		t.Errorf("observations = %d, want 1 (the reload changed nothing)", n)
	}
	if n := count(t, s, "loads"); n != 2 {
		t.Errorf("loads = %d, want 2 (both attempts recorded)", n)
	}

	var sha string
	if err := s.pools.Read().QueryRow(`SELECT sha256 FROM raw_archive`).Scan(&sha); err != nil {
		t.Fatal(err)
	}
	replay, err := s.Body(ctx, sha)
	if err != nil || string(replay) != body {
		t.Fatalf("Body = %q, %v; want the bytes as received", replay, err)
	}
}

func TestAbandonedAndFailedFetchesAreLoggedWithoutBodies(t *testing.T) {
	s, _, _ := newStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 1<<16))
	}))
	client := &http.Client{Transport: &archive.Transport{Sink: s}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = resp.Body.Read(make([]byte, 10))
	_ = resp.Body.Close()
	srv.Close()
	if resp, err := client.Get(srv.URL); err == nil {
		_ = resp.Body.Close()
		t.Fatal("fetch from a closed server succeeded")
	}
	if n := count(t, s, "fetch_log WHERE sha256 IS NULL AND error <> ''"); n != 2 {
		t.Errorf("logged failures without bodies = %d, want 2", n)
	}
	if n := count(t, s, "raw_archive"); n != 0 {
		t.Errorf("raw_archive rows = %d, want 0", n)
	}
}

func TestCorrectionAppendsAndAsOfReadsTheOldValue(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	if _, err := s.Ingest(ctx, solo("alpha", "Solo", "13604", "41")); err != nil {
		t.Fatal(err)
	}
	c.now = t0().Add(24 * time.Hour)
	rep, err := s.Ingest(ctx, solo("alpha", "Solo", "13604", "43"))
	if err != nil || rep.Added != 1 {
		t.Fatalf("correction: %+v, %v", rep, err)
	}

	for _, tc := range []struct {
		asOf time.Time
		want []float64
	}{
		{t0().Add(-time.Second), nil},
		{t0(), []float64{41}},
		{t0().Add(23 * time.Hour), []float64{41}},
		{t0().Add(48 * time.Hour), []float64{43}},
	} {
		var got []float64
		for _, f := range features(t, s, tc.asOf) {
			got = append(got, f.Value)
		}
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("as of %s: %v, want %v", tc.asOf.Format(time.RFC3339), got, tc.want)
		}
	}
	if n := count(t, s, "observations"); n != 2 {
		t.Errorf("observations = %d, want 2: a correction appends", n)
	}
}

func TestWeekZerosAreWrittenOnlyAsCorrections(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	week := func(raw string) measures.Batch {
		return measures.Batch{Source: "alpha", Facts: []measures.Fact{
			{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Week: 3, Field: "Sacks", Raw: raw},
			{IDType: measures.IDTypeMFL, ID: "13604", Field: "Forty", Raw: "4.52"}}}
	}
	if _, err := s.Ingest(ctx, week("0")); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s, "observations"); n != 1 {
		t.Fatalf("a first zero and a forty wrote %d rows, want only the forty", n)
	}
	for _, raw := range []string{"1.5", "0", "0"} {
		c.now = c.now.Add(time.Hour)
		if _, err := s.Ingest(ctx, week(raw)); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(t, s, "observations"); n != 3 {
		t.Fatalf("1.5 then a corrected zero then the same zero wrote %d rows in all, want 3", n)
	}
	got, err := s.Features(ctx, FeatureQuery{AsOf: c.now, Season: 0, Measures: []string{"prior.forty"}})
	if err != nil || len(got) != 1 || got[0].Value != 4.52 || got[0].Week != 0 {
		t.Fatalf("player-grain forty = %+v, %v", got, err)
	}
	c.now = c.now.Add(time.Hour)
	if _, err := s.Ingest(ctx, week("2")); err != nil {
		t.Fatal(err)
	}
	c.now = c.now.Add(time.Hour)
	whole := measures.Batch{Source: "alpha", Scope: &measures.Scope{Seasons: []int{2025}, Measures: []string{"outcome.sacks"}}}
	if rep, err := s.Ingest(ctx, whole); err != nil || rep.Added != 1 {
		t.Fatalf("a whole-season batch without the sack = %+v, %v; want one zero written", rep, err)
	}
	got, err = s.Features(ctx, FeatureQuery{AsOf: c.now, Season: 2025, Measures: []string{"outcome.sacks"}})
	if err != nil || len(got) != 1 || got[0].Value != 0 {
		t.Fatalf("sacks after a whole-season batch dropped them = %+v, %v", got, err)
	}
	bad := measures.Batch{Source: "alpha", Facts: []measures.Fact{
		{IDType: measures.IDTypeMFL, ID: "13604", Season: 2025, Field: "Forty", Raw: "4.5"}}}
	if _, err := s.Ingest(ctx, bad); err == nil || !strings.Contains(err.Error(), "player-grain") {
		t.Fatalf("a forty with a season = %v, want a grain error", err)
	}
}

func TestHistoryRejectsUpdateAndDelete(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.Ingest(context.Background(), solo("alpha", "Solo", "13604", "41")); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{`UPDATE observations SET value = 0`, `DELETE FROM observations`, `DELETE FROM loads`} {
		if _, err := s.pools.Write().Exec(stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: err = %v, want append-only abort", stmt, err)
		}
	}
}

func TestIngestRejectsBadBatchesAndRecordsTheFailure(t *testing.T) {
	cases := []struct {
		name string
		b    measures.Batch
		want string
	}{
		{"unregistered source", solo("gamma", "Solo", "13604", "1"), "unregistered source"},
		{"unmapped field", solo("alpha", "Tkl", "13604", "1"), "no row in source_fields.csv"},
		{"not a number", solo("alpha", "Solo", "13604", "forty"), "not a finite number"},
		{"bad mfl id", solo("alpha", "Solo", "13x", "1"), "playerid"},
		{"week on a season measure", measures.Batch{Source: "alpha", Facts: []measures.Fact{
			{IDType: "mfl", ID: "13604", Season: 2025, Week: 3, Field: "Solo", Raw: "1"}}}, "season-grain"},
		{"season measure twice", measures.Batch{Source: "alpha", Facts: []measures.Fact{
			{IDType: "mfl", ID: "13604", Season: 2025, Field: "Solo", Raw: "1"},
			{IDType: "mfl", ID: "13604", Season: 2025, Field: "Solo", Raw: "2"}}}, "twice in one load"},
		{"empty status", measures.Batch{Source: "alpha", Facts: []measures.Fact{
			{IDType: "mfl", ID: "13604", Season: 2025, Week: 2, Field: "Status", Raw: " "}}}, "is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newStore(t)
			if _, err := s.Ingest(context.Background(), tc.b); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Ingest error = %v, want it to mention %q", err, tc.want)
			}
			if n := count(t, s, "observations"); n != 0 {
				t.Errorf("observations = %d after a rejected batch", n)
			}
			if n := count(t, s, "loads WHERE error <> ''"); n != 1 {
				t.Errorf("failed loads recorded = %d, want 1", n)
			}
		})
	}
}

func TestUnmatchedFactsWaitThenResolveWithTheirAsOf(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	b := measures.Batch{Source: "alpha", Facts: []measures.Fact{
		{IDType: "gsis", ID: "00-0036355", Season: 2025, Field: "Solo", Raw: "55"},
		{IDType: "gsis", ID: "00-0036355", Season: 2025, Week: 4, Field: "Status", Raw: "Questionable"}}}
	rep, err := s.Ingest(ctx, b)
	if err != nil || rep.Unresolved != 2 || rep.Added != 0 {
		t.Fatalf("Ingest = %+v, %v; want 2 unresolved", rep, err)
	}
	if rep, _ = s.Ingest(ctx, b); rep.Unresolved != 0 {
		t.Errorf("reloading waiting facts queued %d again", rep.Unresolved)
	}
	if got := features(t, s, t0()); len(got) != 0 {
		t.Fatalf("an unmatched fact reached the features: %+v", got)
	}

	c.now = t0().Add(72 * time.Hour)
	n, err := s.LinkPlayerIDs(ctx, "alpha", []PlayerIDLink{{IDType: "gsis", IDValue: "00-0036355", PlayerID: "15241"}})
	if err != nil || n != 2 {
		t.Fatalf("LinkPlayerIDs = %d, %v; want 2 promoted", n, err)
	}
	if linked, err := s.LinkedPlayers(ctx, "gsis"); err != nil || !linked["15241"] || len(linked) != 1 {
		t.Errorf("LinkedPlayers(gsis) = %v, %v", linked, err)
	}
	got := features(t, s, t0())
	if len(got) != 1 || got[0].PlayerID != "15241" || got[0].Value != 55 || !got[0].AsOf.Equal(t0()) {
		t.Fatalf("after linking: %+v, want 15241 = 55 known since t0()", got)
	}
	status, err := s.Features(ctx, FeatureQuery{AsOf: t0(), Season: 2025, Measures: []string{"availability.game_status"}})
	if err != nil || len(status) != 1 || status[0].Text != "Questionable" || status[0].Week != 4 {
		t.Fatalf("text feature = %+v, %v", status, err)
	}
	if n := count(t, s, "unresolved_observations"); n != 0 {
		t.Errorf("unresolved left = %d, want 0", n)
	}
}

// TestSourceGainNeedsOnlyMappingRows is the source-gain drill: a second source for an existing
// measure arrives as one source_fields row. No code changes.
func TestSourceGainNeedsOnlyMappingRows(t *testing.T) {
	s, _, pools := newStore(t)
	ctx := context.Background()
	if _, err := s.Ingest(ctx, solo("alpha", "Solo", "13604", "41")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ingest(ctx, solo("beta", "soloTackles", "13604", "41")); err == nil {
		t.Fatal("beta loaded before it had a mapping row")
	}

	s, c, _ := reopen(t, pools, registry(t, betaField))
	c.now = t0().Add(time.Hour)
	beta := measures.Batch{Source: "beta", Facts: []measures.Fact{
		{IDType: "mfl", ID: "13604", Season: 2025, Field: "soloTackles", Raw: "40"},
		{IDType: "mfl", ID: "14777", Season: 2025, Field: "soloTackles", Raw: "12"}}}
	if _, err := s.Ingest(ctx, beta); err != nil {
		t.Fatal(err)
	}

	got := map[string]Feature{}
	for _, f := range features(t, s, c.now) {
		got[f.PlayerID] = f
	}
	if f := got["13604"]; f.Source != "alpha" || f.Value != 41 {
		t.Errorf("13604 = %+v, want alpha's 41: alpha has priority", f)
	}
	if f := got["14777"]; f.Source != "beta" || f.Value != 12 {
		t.Errorf("14777 = %+v, want beta's 12: beta fills the gap", f)
	}
	d, err := s.Disagreements(ctx, c.now)
	if err != nil || len(d) != 1 || d[0].PlayerID != "13604" || d[0].ValueA != "41.0" || d[0].ValueB != "40.0" {
		t.Fatalf("Disagreements = %+v, %v; want 13604 alpha 41 vs beta 40", d, err)
	}
}

func TestSourceIsLostOnlyAfterFailingPastItsMaxAge(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	state := func() SourceState {
		t.Helper()
		h, err := s.SourceHealth(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return h[0].State
	}
	if got := state(); got != SourceActive {
		t.Fatalf("a source with no loads is %s, want active", got)
	}
	fail := errors.New("503 from alpha")
	steps := []struct {
		at    time.Duration
		ok    bool
		state SourceState
	}{
		{0, true, SourceActive},
		{24 * time.Hour, false, SourceActive},   // failed, but succeeded within 7 days
		{8 * 24 * time.Hour, false, SourceLost}, // failing, last success 8 days ago
		{9 * 24 * time.Hour, true, SourceActive},
	}
	for _, st := range steps {
		c.now = t0().Add(st.at)
		var err error
		if st.ok {
			_, err = s.Ingest(ctx, solo("alpha", "Solo", "13604", fmt.Sprint(st.at.Hours())))
		} else {
			err = s.LoadFailed(ctx, "alpha", fail)
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := state(); got != st.state {
			t.Errorf("after %v: alpha is %s, want %s", st.at, got, st.state)
		}
	}
}

func score(id string, adjusted float64) Score {
	return Score{MFLID: id, Result: engine.Result{BasePoints: 100, AgePull: 1, AdjustedScore: adjusted,
		ScoutingAdjusted: adjusted, CapMultiplier: 1, CapTier: engine.CapTierNeutral}}
}

func board(params float64, inputs string) NewRun {
	return NewRun{Kind: RunBoard, Season: 2026, AsOf: t0(), InputsHash: inputs, Engine: "v-test",
		Params: ParamSet{Params: map[string]float64{"layer3.decay_rate": params}, Measures: []string{"outcome.tackles_solo"}},
		Scores: []Score{score("13604", 90), score("0042", 95)}}
}

func TestRunsKeepEveryBoardReadable(t *testing.T) {
	s, _, _ := newStore(t)
	ctx := context.Background()
	first, written, err := s.WriteRun(ctx, board(0.03, "in-1"))
	if err != nil || !written {
		t.Fatalf("first run: %v, written %v", err, written)
	}
	if again, written, err := s.WriteRun(ctx, board(0.03, "in-1")); err != nil || written || again.ID != first.ID {
		t.Fatalf("same params and inputs wrote a new run: %+v %v %v", again, written, err)
	}
	rescored := board(0.03, "in-1")
	rescored.Scores[0].AdjustedScore = 91 // same inputs, new engine code: a new board
	changed, written, err := s.WriteRun(ctx, rescored)
	if err != nil || !written || changed.ScoresHash == first.ScoresHash {
		t.Fatalf("changed scores under the same inputs: %+v %v %v", changed, written, err)
	}
	second, written, err := s.WriteRun(ctx, board(0.05, "in-1"))
	if err != nil || !written || second.ParamSetID == first.ParamSetID {
		t.Fatalf("changed params: %+v %v %v", second, written, err)
	}
	proposal := board(0.04, "in-1")
	proposal.Kind = RunRebalance
	if _, _, err := s.WriteRun(ctx, proposal); err != nil {
		t.Fatal(err)
	}

	latest, ok, err := s.LatestRun(ctx, 2026, RunBoard)
	if err != nil || !ok || latest.ID != second.ID {
		t.Fatalf("board = %+v, want run %d: a rebalance run never becomes the board", latest, second.ID)
	}
	prev, ok, err := s.PreviousRun(ctx, latest)
	if err != nil || !ok || prev.ID != changed.ID {
		t.Fatalf("previous board = %+v, want run %d", prev, changed.ID)
	}
	for _, r := range []Run{first, changed, second} {
		sc, err := s.RunScores(ctx, r.ID)
		if err != nil || len(sc) != 2 || sc[0].MFLID != "0042" {
			t.Errorf("run %d scores = %+v, %v; want 0042 first", r.ID, sc, err)
		}
	}
	if one, ok, err := s.RunScore(ctx, first.ID, "0042"); err != nil || !ok || one.AdjustedScore != 95 {
		t.Errorf("RunScore = %+v %v %v", one, ok, err)
	}
	if runs, err := s.Runs(ctx, 2026); err != nil || len(runs) != 4 {
		t.Errorf("Runs = %d, %v; want 4", len(runs), err)
	}
}

func TestRunRecordsMeasuresThatStoppedFlowing(t *testing.T) {
	s, c, _ := newStore(t)
	ctx := context.Background()
	c.now = t0().Add(30 * 24 * time.Hour)
	if err := s.LoadFailed(ctx, "alpha", errors.New("gone")); err != nil {
		t.Fatal(err)
	}
	r, _, err := s.WriteRun(ctx, board(0.03, "in-1"))
	if err != nil || fmt.Sprint(r.MissingMeasures) != "[outcome.tackles_solo]" {
		t.Fatalf("run = %+v, %v; want outcome.tackles_solo missing", r, err)
	}
}

func TestWriteRunRejectsBadScores(t *testing.T) {
	cases := map[string]func(*NewRun){
		"no scores":     func(r *NewRun) { r.Scores = nil },
		"no inputs":     func(r *NewRun) { r.InputsHash = "" },
		"no engine":     func(r *NewRun) { r.Engine = "" },
		"unknown kind":  func(r *NewRun) { r.Kind = "draft" },
		"bad id":        func(r *NewRun) { r.Scores[0].MFLID = "13x" },
		"repeated id":   func(r *NewRun) { r.Scores[1].MFLID = r.Scores[0].MFLID },
		"unknown tier":  func(r *NewRun) { r.Scores[0].CapTier = "Warm" },
		"NaN score":     func(r *NewRun) { r.Scores[0].AdjustedScore = math.NaN() },
		"NaN parameter": func(r *NewRun) { r.Params.Params["x"] = math.NaN() },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s, _, _ := newStore(t)
			r := board(0.03, "in-1")
			mutate(&r)
			if _, _, err := s.WriteRun(context.Background(), r); err == nil {
				t.Fatal("WriteRun accepted it")
			}
			if n := count(t, s, "scoring_runs"); n != 0 {
				t.Errorf("scoring_runs = %d", n)
			}
		})
	}
}

func TestArchivedBodiesOfferNewestFirstUntilOneIsTaken(t *testing.T) {
	s, _, _ := newStore(t)
	ctx := context.Background()
	for i, body := range []string{"first", "second", "third"} {
		if err := s.Record(ctx, archiveFetch("https://api.myfantasyleague.com/2026/export?TYPE=leagueStandings", body,
			t0().Add(time.Duration(i)*time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	var offered []string
	found, err := s.ArchivedBodies(ctx, "TYPE=leagueStandings", func(_ string, body []byte, _ time.Time) bool {
		offered = append(offered, string(body))
		return string(body) == "second"
	})
	if err != nil || !found || strings.Join(offered, ",") != "third,second" {
		t.Fatalf("offered %v, found %v, err %v", offered, found, err)
	}
}

// archiveFetch is a complete fetch of body from url, as the archive transport records it.
func archiveFetch(url, body string, at time.Time) archive.Fetch {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(body))
	_ = zw.Close()
	sum := sha256.Sum256([]byte(body))
	return archive.Fetch{URL: url, Status: http.StatusOK, SHA256: hex.EncodeToString(sum[:]),
		Size: int64(len(body)), Gzip: buf.Bytes(), FetchedAt: at}
}

func modelRun(knob float64) NewModelRun {
	score := func(id string, dyn float64) ModelScore {
		return ModelScore{MFLID: id, Position: "WR", Inputs: []string{"draft", "season.2025"},
			Value: model.Value{Now: 0.6, NowPPG: 12, Dynasty: dyn, DynastyPPG: dyn * 20, Prior: 0.4, PastGames: 15, OnField: 0.9}}
	}
	return NewModelRun{Season: 2026, AsOf: t0(), InputsHash: "in-1", Engine: "v-test",
		Params: ParamSet{Params: map[string]float64{"model.k_now@WR": knob}, Measures: []string{"outcome.fantasy_points"}},
		Scores: []ModelScore{score("13604", 0.5), score("0042", 0.7)}}
}

// A model run is kept apart from the boards, deduplicated like them, and every one stays readable.
func TestModelRunsAreKeptBesideTheBoard(t *testing.T) {
	s, _, _ := newStore(t)
	ctx := context.Background()
	if _, _, err := s.WriteRun(ctx, board(0.03, "in-1")); err != nil {
		t.Fatal(err)
	}
	first, written, err := s.WriteModelRun(ctx, modelRun(2))
	if err != nil || !written || first.Kind != RunModel {
		t.Fatalf("first model run: %+v %v %v", first, written, err)
	}
	if again, written, err := s.WriteModelRun(ctx, modelRun(2)); err != nil || written || again.ID != first.ID {
		t.Fatalf("the same model run was written twice: %+v %v %v", again, written, err)
	}
	second, written, err := s.WriteModelRun(ctx, modelRun(3))
	if err != nil || !written || second.ParamSetID == first.ParamSetID {
		t.Fatalf("an edited param must make a new run: %+v %v %v", second, written, err)
	}
	if latest, ok, err := s.LatestRun(ctx, 2026, RunModel); err != nil || !ok || latest.ID != second.ID {
		t.Fatalf("latest model run = %+v %v %v", latest, ok, err)
	}
	if b, ok, err := s.LatestRun(ctx, 2026, RunBoard); err != nil || !ok || b.Kind != RunBoard {
		t.Fatalf("the board must be untouched: %+v %v %v", b, ok, err)
	}
	old, err := s.ModelScores(ctx, first.ID)
	if err != nil || len(old) != 2 || old[0].MFLID != "0042" || old[0].DynastyPPG != 14 || old[0].Inputs[1] != "season.2025" {
		t.Errorf("first run's scores = %+v, %v", old, err)
	}
	bad := modelRun(4)
	bad.Scores[0].NowPPG = math.NaN()
	if _, _, err := s.WriteModelRun(ctx, bad); err == nil {
		t.Error("a NaN must never freeze into the append-only table")
	}
}

func TestMeasuresHeldListsASourcesMeasuresBySeason(t *testing.T) {
	s, _, _ := newStore(t)
	ctx := context.Background()
	if _, err := s.Ingest(ctx, solo("alpha", "Solo", "13604", "41")); err != nil {
		t.Fatal(err)
	}
	held, err := s.MeasuresHeld(ctx, "alpha", 2025)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 1 || !held["outcome.tackles_solo"] {
		t.Errorf("held in 2025 = %v, want outcome.tackles_solo", held)
	}
	for _, q := range []struct {
		source string
		season int
	}{{"alpha", 2024}, {"beta", 2025}} {
		if held, err := s.MeasuresHeld(ctx, q.source, q.season); err != nil || len(held) != 0 {
			t.Errorf("%s %d: held %v, err %v; want none", q.source, q.season, held, err)
		}
	}
}
