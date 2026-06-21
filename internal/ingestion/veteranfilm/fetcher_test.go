package veteranfilm

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serveCSV returns a client + URL serving body with status 200.
func serveCSV(t *testing.T, body string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

const ftnFixture = `nflverse_game_id,nflverse_play_id,is_contested_ball,is_created_reception,is_drop,is_interception_worthy,is_throw_away
2025_01_AAA_BBB,1,TRUE,TRUE,FALSE,FALSE,FALSE
2025_01_AAA_BBB,2,FALSE,FALSE,TRUE,FALSE,FALSE
2025_01_AAA_BBB,3,FALSE,FALSE,FALSE,TRUE,FALSE
2025_01_AAA_BBB,4,FALSE,FALSE,FALSE,FALSE,FALSE
`

// pbp play 5 is deliberately absent from FTN (uncharted) — it must be skipped.
const pbpFixture = `play_id,game_id,passer_player_id,receiver_player_id
1,2025_01_AAA_BBB,00-P1,00-R1
2,2025_01_AAA_BBB,00-P1,00-R1
3,2025_01_AAA_BBB,00-P1,00-R1
4,2025_01_AAA_BBB,00-P1,00-R2
5,2025_01_AAA_BBB,00-P1,00-R1
`

func approx(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

func TestFetch_JoinAttributeRateAndFloor(t *testing.T) {
	t.Parallel()
	ftnClient, ftnURL := serveCSV(t, ftnFixture)
	_, pbpURL := serveCSV(t, pbpFixture)

	out, err := Fetch(context.Background(), ftnClient,
		[]SeasonSource{{FTNURL: ftnURL, PBPURL: pbpURL}}, 2, 2)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// R1: 3 charted targets (play 5 uncharted, skipped). 1 contested, 1 created, 1 drop.
	r1, ok := out["00-R1"]
	if !ok || r1.Receiver == nil {
		t.Fatalf("R1 receiver group missing: %+v", out["00-R1"])
	}
	if r1.Receiver.Targets != 3 {
		t.Errorf("R1 targets = %d, want 3 (uncharted play 5 must not count)", r1.Receiver.Targets)
	}
	for name, got := range map[string]float64{
		"contested": r1.Receiver.ContestedCatchRate,
		"created":   r1.Receiver.CreatedReceptionRate,
		"drop":      r1.Receiver.DropRate,
	} {
		if !approx(got, 1.0/3.0) {
			t.Errorf("R1 %s rate = %v, want 1/3", name, got)
		}
	}

	// P1: 4 charted attempts (play 5 skipped), 1 interception-worthy.
	p1, ok := out["00-P1"]
	if !ok || p1.Passer == nil {
		t.Fatalf("P1 passer group missing: %+v", out["00-P1"])
	}
	if p1.Passer.Attempts != 4 {
		t.Errorf("P1 attempts = %d, want 4", p1.Passer.Attempts)
	}
	if !approx(p1.Passer.InterceptionWorthyRate, 0.25) {
		t.Errorf("P1 int-worthy rate = %v, want 0.25", p1.Passer.InterceptionWorthyRate)
	}
	if !approx(p1.Passer.ThrowawayRate, 0) {
		t.Errorf("P1 throwaway rate = %v, want 0", p1.Passer.ThrowawayRate)
	}

	// R2: 1 charted target, below the floor of 2 → suppressed entirely.
	if _, present := out["00-R2"]; present {
		t.Errorf("R2 has 1 target (< floor 2) and should be suppressed, got %+v", out["00-R2"])
	}
}

func TestFetch_BadBooleanFailsLoud(t *testing.T) {
	t.Parallel()
	bad := strings.Replace(ftnFixture, "FALSE,FALSE,TRUE,FALSE,FALSE", "FALSE,FALSE,MAYBE,FALSE,FALSE", 1)
	client, ftnURL := serveCSV(t, bad)
	_, pbpURL := serveCSV(t, pbpFixture)

	_, err := Fetch(context.Background(), client, []SeasonSource{{FTNURL: ftnURL, PBPURL: pbpURL}}, 2, 2)
	if err == nil || !strings.Contains(err.Error(), "not a boolean") {
		t.Fatalf("want a not-a-boolean error, got %v", err)
	}
}

func TestFetch_DuplicateFTNPlayFailsLoud(t *testing.T) {
	t.Parallel()
	dup := ftnFixture + "2025_01_AAA_BBB,1,FALSE,FALSE,FALSE,FALSE,FALSE\n"
	client, ftnURL := serveCSV(t, dup)
	_, pbpURL := serveCSV(t, pbpFixture)

	_, err := Fetch(context.Background(), client, []SeasonSource{{FTNURL: ftnURL, PBPURL: pbpURL}}, 2, 2)
	if err == nil || !strings.Contains(err.Error(), "duplicate play") {
		t.Fatalf("want a duplicate-play error, got %v", err)
	}
}

func TestFetch_EmptyResolvesLoud(t *testing.T) {
	t.Parallel()
	// FTN charts nothing pbp references → zero players above the floor.
	emptyFTN := "nflverse_game_id,nflverse_play_id,is_contested_ball,is_created_reception,is_drop,is_interception_worthy,is_throw_away\n"
	client, ftnURL := serveCSV(t, emptyFTN)
	_, pbpURL := serveCSV(t, pbpFixture)

	_, err := Fetch(context.Background(), client, []SeasonSource{{FTNURL: ftnURL, PBPURL: pbpURL}}, 2, 2)
	if err == nil {
		t.Fatal("want errEmpty when nothing resolves above the floor, got nil")
	}
}
