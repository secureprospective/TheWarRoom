package assembly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/veteranfilm"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// --- pure Madden offense backbone (offenseMaddenComposite) ---

// TestOffenseMaddenComposite_EqualWeightMeanAndBoundary pins the equal-weight-mean recipe
// (each rubric row a term, attrs averaged within a term) and the QB/RB/WR/TE-only boundary.
func TestOffenseMaddenComposite_EqualWeightMeanAndBoundary(t *testing.T) {
	t.Parallel()
	// WR term 1 = {speed,acceleration}; give 80/40 (avg 60) and leave the rest absent, so
	// the composite = that one present term = 60/99.
	rc := madden.RawMaddenRating{Attributes: map[string]int{"speed": 80, "acceleration": 40}}
	got, ok := offenseMaddenComposite(rc, domain.PosWR)
	if !ok {
		t.Fatal("WR composite reported no signal")
	}
	if want := 60.0 / 99.0; !idpFinite(got, want) {
		t.Fatalf("WR composite = %.6f, want %.6f (term = speed/accel avg, present-only)", got, want)
	}
	if _, ok := offenseMaddenComposite(rc, domain.PosDT); ok {
		t.Error("DT (defense) must report no offense film signal")
	}
	if _, ok := offenseMaddenComposite(rc, domain.PosK); ok {
		t.Error("K must report no offense film signal")
	}
	if _, ok := offenseMaddenComposite(
		madden.RawMaddenRating{Attributes: map[string]int{"unrelated": 90}}, domain.PosQB); ok {
		t.Error("a record carrying none of the curated QB attrs must report absent")
	}
}

// --- FTN role collapse (ftnRoleQuality) ---

func TestFTNRoleQuality_ReceiverAndPasser(t *testing.T) {
	t.Parallel()
	rf := veteranfilm.RawVeteranFilm{
		Receiver: &veteranfilm.ReceiverFilm{
			ContestedCatchRate: 0.8, CreatedReceptionRate: 0.4, DropRate: 0.1,
		},
		Passer: &veteranfilm.PasserFilm{InterceptionWorthyRate: 0.2},
	}
	q, ok := ftnRoleQuality(rf, roleReceiver)
	if want := (0.8 + 0.4 + 0.9) / 3.0; !ok || !idpFinite(q, want) {
		t.Fatalf("receiver quality = %.6f (ok=%v), want %.6f (drop inverted)", q, ok, want)
	}
	q, ok = ftnRoleQuality(rf, rolePasser)
	if !ok || !idpFinite(q, 0.8) {
		t.Fatalf("passer quality = %.6f (ok=%v), want 0.8 (1 - int-worthy)", q, ok)
	}
	// Absent role group → no signal.
	if _, ok := ftnRoleQuality(veteranfilm.RawVeteranFilm{}, roleReceiver); ok {
		t.Error("a nil Receiver group must report no FTN signal")
	}
}

// --- percentile (rolePercentiles) ---

func TestRolePercentiles_MidpointAndSingleton(t *testing.T) {
	t.Parallel()
	rows := []offenseFilmRow{
		{role: roleReceiver, ftnQuality: 0.0, hasFTN: true},
		{role: roleReceiver, ftnQuality: 1.0, hasFTN: true},
		{role: rolePasser, ftnQuality: 0.7, hasFTN: true}, // singleton population
	}
	r := newRolePercentiles(rows)
	if p := r.percentile(roleReceiver, 0.0); !idpFinite(p, 0.25) {
		t.Errorf("low receiver percentile = %.4f, want 0.25", p)
	}
	if p := r.percentile(roleReceiver, 1.0); !idpFinite(p, 0.75) {
		t.Errorf("high receiver percentile = %.4f, want 0.75", p)
	}
	if p := r.percentile(rolePasser, 0.7); !idpFinite(p, 0.5) {
		t.Errorf("singleton percentile = %.4f, want 0.5 (no peers)", p)
	}
}

// --- overlay blend (blendOffenseRow) ---

// TestBlendOffenseRow covers the three K3 regimes: below-floor = pure backbone; a bounded
// nudge within ±0.10; and the clamp when the discounted gap exceeds the bound.
func TestBlendOffenseRow(t *testing.T) {
	t.Parallel()
	// Two receivers so percentiles are 0.25 / 0.75; a passer singleton (0.5).
	ranker := newRolePercentiles([]offenseFilmRow{
		{role: roleReceiver, ftnQuality: 0.0, hasFTN: true},
		{role: roleReceiver, ftnQuality: 1.0, hasFTN: true},
	})

	// Below the floor → composite is exactly the backbone (no overlay).
	below := offenseFilmRow{role: roleReceiver, backbone: 0.42, hasFTN: false}
	if got := blendOffenseRow(below, ranker); !idpFinite(got, 0.42) {
		t.Errorf("below-floor composite = %.6f, want backbone 0.42", got)
	}

	// High-percentile (0.75) receiver, backbone 0.50: delta = 0.85·(0.75−0.50)=0.2125 →
	// clamps to +0.10 → 0.60.
	high := offenseFilmRow{role: roleReceiver, backbone: 0.50, ftnQuality: 1.0, hasFTN: true}
	if got := blendOffenseRow(high, ranker); !idpFinite(got, 0.60) {
		t.Errorf("clamped-up composite = %.6f, want 0.60 (bound ±0.10)", got)
	}

	// A small in-bound gap: backbone 0.30, percentile 0.25 → delta 0.85·(−0.05)=−0.0425,
	// unclamped → 0.2575.
	small := newRolePercentiles([]offenseFilmRow{
		{role: roleReceiver, ftnQuality: 0.0, hasFTN: true},
		{role: roleReceiver, ftnQuality: 1.0, hasFTN: true},
	})
	low := offenseFilmRow{role: roleReceiver, backbone: 0.30, ftnQuality: 0.0, hasFTN: true}
	if got := blendOffenseRow(low, small); !idpFinite(got, 0.30+ftnDiscount*(0.25-0.30)) {
		t.Errorf("in-bound composite = %.6f, want %.6f", got, 0.30+ftnDiscount*(0.25-0.30))
	}
}

// --- BuildOffenseFilm end-to-end (madden backbone + FTN overlay + boundary) ---

func serveBody(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

const offenseFilmCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id,name,birthdate
1001,G-1,,,Al Pha,1997-09-03
1002,G-2,,,Bo Beta,1999-01-02
1003,Q-3,,,Cy Gamma,2000-05-06
1004,G-4,,,De Delta,1996-04-04
`

// Two WR-target plays each for G-1 (all bad → quality 0) and G-2 (all good → quality 1),
// then two passer plays for Q-3 (never int-worthy → quality 1). play/passer/receiver ids
// are split so a play feeds exactly one role.
const offenseFilmFTN = `nflverse_game_id,nflverse_play_id,is_contested_ball,is_created_reception,is_drop,is_interception_worthy,is_throw_away
2024_01_AAA_BBB,1,FALSE,FALSE,TRUE,FALSE,FALSE
2024_01_AAA_BBB,2,FALSE,FALSE,TRUE,FALSE,FALSE
2024_01_AAA_BBB,3,TRUE,TRUE,FALSE,FALSE,FALSE
2024_01_AAA_BBB,4,TRUE,TRUE,FALSE,FALSE,FALSE
2024_01_AAA_BBB,5,FALSE,FALSE,FALSE,FALSE,FALSE
2024_01_AAA_BBB,6,FALSE,FALSE,FALSE,FALSE,FALSE
`

const offenseFilmPBP = `play_id,game_id,passer_player_id,receiver_player_id
1,2024_01_AAA_BBB,,G-1
2,2024_01_AAA_BBB,,G-1
3,2024_01_AAA_BBB,,G-2
4,2024_01_AAA_BBB,,G-2
5,2024_01_AAA_BBB,Q-3,
6,2024_01_AAA_BBB,Q-3,
`

func TestBuildOffenseFilm_ComposesBackboneAndBoundedOverlay(t *testing.T) {
	cw := crosswalkFixture(t, offenseFilmCrosswalkCSV)
	// Give the two WRs an IDENTICAL backbone (only the speed/accel term present = 50/99),
	// so their composites differ ONLY by the FTN overlay. Q-3 (QB) gets throwPower=50/99.
	// G-4 is a DT — resolves to a gsis but the offense boundary rejects it.
	maddenClient, maddenURL := maddenServer(t, []string{
		maddenDoc("Al", "Pha", "9/3/1997", map[string]int{"speed": 50, "acceleration": 50}),
		maddenDoc("Bo", "Beta", "1/2/1999", map[string]int{"speed": 50, "acceleration": 50}),
		maddenDoc("Cy", "Gamma", "5/6/2000", map[string]int{"throwPower": 50}),
		maddenDoc("De", "Delta", "4/4/1996", map[string]int{"tackle": 90}),
	})
	ftnURL := serveBody(t, offenseFilmFTN)
	pbpURL := serveBody(t, offenseFilmPBP)

	pos := fakePosLookup{
		"1001": domain.PosWR, "1002": domain.PosWR, "1003": domain.PosQB, "1004": domain.PosDT,
	}
	out, err := BuildOffenseFilm(context.Background(), maddenClient, maddenURL,
		[]veteranfilm.SeasonSource{{FTNURL: ftnURL, PBPURL: pbpURL}},
		2, 2, cw, []string{"1001", "1002", "1003", "1004"}, pos)
	if err != nil {
		t.Fatalf("BuildOffenseFilm: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("got %d composites, want 3 (DT must be skipped)", len(out))
	}

	backbone := 50.0 / 99.0
	wrLow, _ := playerid.New("1001")  // FTN quality 0 → percentile 0.25 → clamp −0.10
	wrHigh, _ := playerid.New("1002") // FTN quality 1 → percentile 0.75 → clamp +0.10
	if want := backbone - ftnOverlayBound; !idpFinite(out[wrLow], want) {
		t.Errorf("low-FTN WR composite = %.6f, want %.6f (backbone − bound)", out[wrLow], want)
	}
	if want := backbone + ftnOverlayBound; !idpFinite(out[wrHigh], want) {
		t.Errorf("high-FTN WR composite = %.6f, want %.6f (backbone + bound)", out[wrHigh], want)
	}
	// QB is a singleton passer (percentile 0.5 ≈ backbone) → composite within a hair of it.
	qb, _ := playerid.New("1003")
	if got := out[qb]; got < backbone-0.02 || got > backbone+0.02 {
		t.Errorf("QB composite = %.6f, want ≈ backbone %.6f (singleton overlay)", got, backbone)
	}
	dt, _ := playerid.New("1004")
	if _, present := out[dt]; present {
		t.Error("DT (defense) must not appear in the offense film map")
	}
}

func TestBuildOffenseFilm_FetchFailureLoud(t *testing.T) {
	cw := crosswalkFixture(t, offenseFilmCrosswalkCSV)
	client, maddenURL := maddenServer(t, []string{
		maddenDoc("Al", "Pha", "9/3/1997", map[string]int{"speed": 50, "acceleration": 50}),
	})
	// An empty FTN feed → veteranfilm.Fetch resolves zero above-floor players → loud.
	emptyFTN := serveBody(t, "nflverse_game_id,nflverse_play_id,is_contested_ball,is_created_reception,is_drop,is_interception_worthy,is_throw_away\n")
	emptyPBP := serveBody(t, "play_id,game_id,passer_player_id,receiver_player_id\n")
	if _, err := BuildOffenseFilm(context.Background(), client, maddenURL,
		[]veteranfilm.SeasonSource{{FTNURL: emptyFTN, PBPURL: emptyPBP}},
		2, 2, cw, []string{"1001"}, fakePosLookup{"1001": domain.PosWR}); err == nil {
		t.Fatal("expected a loud error on a zero-record FTN fetch")
	}
}

func TestBuildOffenseFilm_GuardsNilDeps(t *testing.T) {
	if _, err := BuildOffenseFilm(context.Background(), nil, "u", nil, 2, 2,
		crosswalk.Map{}, nil, fakePosLookup{}); err == nil {
		t.Error("expected an error for a nil *http.Client")
	}
	if _, err := BuildOffenseFilm(context.Background(), &http.Client{}, "u", nil, 2, 2,
		crosswalk.Map{}, nil, nil); err == nil {
		t.Error("expected an error for a nil PositionLookup")
	}
}
