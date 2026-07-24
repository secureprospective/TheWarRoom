package pfrpassrush

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// header is the LIVE nflverse advstats_season_def header (30 cols, incl. the `age` column PFR
// added after pfrcoverage was first written — by-name binding makes column order irrelevant).
const header = "season,player,pfr_id,tm,age,pos,g,gs,int,tgt,cmp,cmp_percent,yds,yds_cmp,yds_tgt,td,rat,dadot,air,yac,bltz,hrry,qbkd,sk,prss,comb,m_tkl,m_tkl_percent,loaded,bats"

// serveGzCSV serves the given CSV body gzip-compressed (the real source is .csv.gz).
func serveGzCSV(t *testing.T, body string) (*http.Client, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(body)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	raw := buf.Bytes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

// row builds a defense CSV line matching header. Only the columns the fetcher reads (season,
// player, pfr_id, tm, pos, g, bltz, hrry, qbkd, sk, prss) are meaningful; the rest are padded.
func row(season, player, pfr, tm, pos, g, bltz, hrry, qbkd, sk, prss string) string {
	// header order (30 cols): season,player,pfr_id,tm,age,pos,g,gs,int,tgt,cmp,cmp_percent,
	// yds,yds_cmp,yds_tgt,td,rat,dadot,air,yac,bltz,hrry,qbkd,sk,prss,comb,m_tkl,
	// m_tkl_percent,loaded,bats
	c := []string{
		season, player, pfr, tm, "27", pos, g, "16", "0", "0", "0", "", "0", "", "", "0", "", "", "", "",
		bltz, hrry, qbkd, sk, prss, "", "", "", "", "",
	}
	out := c[0]
	for _, v := range c[1:] {
		out += "," + v
	}
	return out
}

func fetch(t *testing.T, body, season string, m map[string]string) (map[string]RawPassRush, error) {
	t.Helper()
	client, url := serveGzCSV(t, body)
	return Fetch(context.Background(), client, url, season, m)
}

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	// Myles Garrett 2023 shape: g=17, bltz=0, hrry=6, qbkd=16, sk=14, prss=37.
	body := header + "\n" +
		row("2023", "Myles Garrett", "GarrMy00", "CLE", "DE", "17", "0", "6", "16", "14", "37") + "\n"
	m := map[string]string{"GarrMy00": "00-0033446"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	pr, ok := out["00-0033446"]
	if !ok {
		t.Fatalf("expected gsis 00-0033446 in output, got %v", out)
	}
	if pr.Position != "DE" || pr.Games != 17 || pr.Blitzes != 0 || pr.Hurries != 6 ||
		pr.QBKnockdowns != 16 || pr.Pressures != 37 {
		t.Errorf("counting fields wrong: %+v", pr)
	}
	if pr.Sacks != 14.0 {
		t.Errorf("Sacks = %v, want 14.0", pr.Sacks)
	}
}

func TestFetch_FractionalSacks(t *testing.T) {
	t.Parallel()
	// PFR records half-sacks (e.g. 2.5) — the one float pass-rush stat.
	body := header + "\n" +
		row("2023", "Half Sack", "HalfSk00", "ATL", "LB", "16", "40", "5", "3", "2.5", "14") + "\n"
	m := map[string]string{"HalfSk00": "00-0035501"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := out["00-0035501"].Sacks; got != 2.5 {
		t.Errorf("Sacks = %v, want 2.5 (half-sack preserved)", got)
	}
}

func TestFetch_NoPassRushSignalSkipped(t *testing.T) {
	t.Parallel()
	// A pure coverage CB: 0 across every pass-rush stat -> no signal -> skipped -> errEmpty.
	body := header + "\n" +
		row("2023", "Cover Corner", "CoveCo00", "NYJ", "CB", "16", "0", "0", "0", "0", "0") + "\n"
	m := map[string]string{"CoveCo00": "00-0035502"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only defender had no pass-rush production")
	}
}

func TestFetch_BlitzesAloneAreNotSignal(t *testing.T) {
	t.Parallel()
	// Blitzes is a scheme stat, not production: a defender blitzed 20 times but recording no
	// pressure/sack/knockdown/hurry carries no pass-rush SIGNAL and is skipped.
	body := header + "\n" +
		row("2023", "Empty Blitz", "EmptBl00", "CHI", "S", "16", "20", "0", "0", "0", "0") + "\n"
	m := map[string]string{"EmptBl00": "00-0035503"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected errEmpty: blitzes alone are not a pass-rush signal")
	}
}

func TestFetch_HurriesOnlyIsSignal(t *testing.T) {
	t.Parallel()
	// A defender with only hurries (no sacks/knockdowns/pressures aggregate) still produced a
	// pass-rush event and must be emitted.
	body := header + "\n" +
		row("2023", "Hurry Only", "HurrOn00", "DEN", "LB", "16", "10", "4", "0", "0", "0") + "\n"
	m := map[string]string{"HurrOn00": "00-0035504"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if out["00-0035504"].Hurries != 4 {
		t.Fatalf("want the hurries-only defender emitted, got %v", out)
	}
}

func TestFetch_MissingCountIsZero(t *testing.T) {
	t.Parallel()
	// Blank pass-rush cells are real zeros (missing-is-zero), but a present sack keeps the
	// defender in: Sacks=1, everything else blank -> emitted with zeros for the blanks.
	body := header + "\n" +
		row("2023", "Sparse Rush", "SparRu00", "SF", "DT", "15", "", "", "", "1.0", "") + "\n"
	m := map[string]string{"SparRu00": "00-0035505"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	pr := out["00-0035505"]
	if pr.Sacks != 1.0 || pr.Hurries != 0 || pr.QBKnockdowns != 0 || pr.Pressures != 0 || pr.Blitzes != 0 {
		t.Errorf("missing-is-zero failed: %+v", pr)
	}
}

func TestFetch_TradedPlayerPrefersAggregate(t *testing.T) {
	t.Parallel()
	// Traded player: a 2TM aggregate plus two per-team split rows. The aggregate wins.
	body := header + "\n" +
		row("2023", "Trade Rush", "TradRu00", "2TM", "DE", "16", "10", "8", "6", "9.0", "30") + "\n" +
		row("2023", "Trade Rush", "TradRu00", "DAL", "DE", "8", "5", "4", "3", "5.0", "16") + "\n" +
		row("2023", "Trade Rush", "TradRu00", "NYG", "DE", "8", "5", "4", "3", "4.0", "14") + "\n"
	m := map[string]string{"TradRu00": "00-0035506"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 record (the aggregate), got %d: %v", len(out), out)
	}
	if pr := out["00-0035506"]; pr.Pressures != 30 || pr.Sacks != 9.0 {
		t.Errorf("did not pick the 2TM aggregate row: %+v", pr)
	}
}

func TestFetch_TradedPlayerNoAggregateDropped(t *testing.T) {
	t.Parallel()
	// Splits with NO combined aggregate row -> ambiguous -> dropped entirely.
	body := header + "\n" +
		row("2023", "Split Rush", "SpltRu00", "DAL", "DE", "8", "5", "4", "3", "5.0", "16") + "\n" +
		row("2023", "Split Rush", "SpltRu00", "NYG", "DE", "8", "5", "4", "3", "4.0", "14") + "\n"
	m := map[string]string{"SpltRu00": "00-0035507"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only player had ambiguous splits and is dropped")
	}
}

func TestFetch_UnresolvablePfrSkipped(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2023", "Known", "KnowRu00", "ARI", "DE", "16", "0", "5", "4", "8.0", "20") + "\n" +
		row("2023", "Unknown", "UnknRu00", "ARI", "DE", "16", "0", "3", "2", "4.0", "12") + "\n"
	m := map[string]string{"KnowRu00": "00-0035508"} // UnknRu00 absent from crosswalk

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(out) != 1 || out["00-0035508"].Pressures != 20 {
		t.Fatalf("want only the resolvable player, got %v", out)
	}
}

func TestFetch_PoisonGsisDropped(t *testing.T) {
	t.Parallel()
	// Two DISTINCT pfr ids resolving to one gsis -> drop both (ras precedent).
	body := header + "\n" +
		row("2023", "Player A", "PlayA000", "ARI", "DE", "16", "0", "5", "4", "8.0", "20") + "\n" +
		row("2023", "Player B", "PlayB000", "SEA", "DE", "16", "0", "3", "2", "4.0", "12") + "\n"
	m := map[string]string{"PlayA000": "00-0035509", "PlayB000": "00-0035509"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only gsis was poisoned by two pfr ids")
	}
}

func TestFetch_SeasonFilter(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2022", "Old Year", "YearRu00", "ARI", "DE", "16", "0", "9", "7", "12.0", "40") + "\n" +
		row("2023", "New Year", "YearRu00", "ARI", "DE", "16", "0", "5", "4", "8.0", "20") + "\n"
	m := map[string]string{"YearRu00": "00-0035510"}

	out, err := fetch(t, body, "2023", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if out["00-0035510"].Pressures != 20 {
		t.Fatalf("season filter failed: got %+v, want the 2023 row (prss=20)", out["00-0035510"])
	}
}

func TestFetch_MalformedCountFailsLoud(t *testing.T) {
	t.Parallel()
	// A present-but-unparseable counting cell (prss="oops") is corruption, not zero.
	body := header + "\n" +
		row("2023", "Bad Row", "BadrRu00", "ARI", "DE", "16", "0", "5", "4", "8.0", "oops") + "\n"
	m := map[string]string{"BadrRu00": "00-0035511"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected a malformed-cell error on prss=oops")
	}
}

func TestFetch_MalformedSackFailsLoud(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2023", "Bad Sack", "BadsRu00", "ARI", "DE", "16", "0", "5", "4", "two", "20") + "\n"
	m := map[string]string{"BadsRu00": "00-0035512"}

	_, err := fetch(t, body, "2023", m)
	if err == nil {
		t.Fatal("expected a malformed-cell error on sk=two")
	}
}

func TestIsTeamAggregate(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"2TM", "3TM", "10TM"} {
		if !isTeamAggregate(s) {
			t.Errorf("isTeamAggregate(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"TB", "LA", "TM", "2T", "2TMX", ""} {
		if isTeamAggregate(s) {
			t.Errorf("isTeamAggregate(%q) = true, want false", s)
		}
	}
}
