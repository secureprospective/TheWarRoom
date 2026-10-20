package manifest_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

func sum(body string) string {
	s := sha256.Sum256([]byte(body))
	return hex.EncodeToString(s[:])
}

type bodies map[string]string

func (b bodies) Body(_ context.Context, sha string) ([]byte, error) {
	body, ok := b[sha]
	if !ok {
		return nil, errors.New("no archived body")
	}
	return []byte(body), nil
}

const (
	rosters2013 = "https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=rosters&W=1"
	polls2013   = "https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=polls"
	adp2026     = "https://api.myfantasyleague.com/2026/export?JSON=1&TYPE=adp"
)

func cutoffTime() time.Time { return time.Date(2026, 10, 6, 1, 42, 57, 0, time.UTC) }
func beforeTime() time.Time { return cutoffTime().Add(-time.Hour) }
func afterTime() time.Time  { return cutoffTime().Add(time.Hour) }

// archive holds an old and a newer capture of the same rosters request, an error body and an adp.
func archive() ([]manifest.Fetch, bodies) {
	b := bodies{sum("old"): "old", sum("new"): "new", sum("login"): "login", sum("adp"): "adp"}
	return []manifest.Fetch{
		{ID: 1, URL: rosters2013, SHA256: sum("old"), FetchedAt: beforeTime()},
		{ID: 2, URL: polls2013, SHA256: sum("login"), Error: "MFL error: API requires logged in user", FetchedAt: beforeTime()},
		{ID: 3, URL: adp2026, SHA256: sum("adp"), FetchedAt: beforeTime()},
		{ID: 4, URL: rosters2013, SHA256: sum("new"), FetchedAt: afterTime()},
	}, b
}

func pins() []manifest.Pin {
	return []manifest.Pin{
		{File: "2026/adp.json", SHA256: sum("adp")},
		{File: "2013/weekly/rosters_W01.json", SHA256: sum("old")},
		{File: "2013/polls.json", SHA256: sum("login")},
	}
}

func TestBuildPinsTheMatchingCaptureAndResolvesItNotTheNewest(t *testing.T) {
	fetches, b := archive()
	m, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 3 || len(m.Missing()) != 0 || len(m.SHA256) != 64 {
		t.Fatalf("manifest = %+v, want three held entries and a sha256", m)
	}
	r := m.Entries[1]
	if r.File != "2013/weekly/rosters_W01.json" || r.FetchID != 1 || r.League != "51719" || r.Season != 2013 ||
		r.Week != 1 || r.Scope != "league" || r.CapturedAt != "2026-10-06T00:42:57Z" {
		t.Errorf("rosters entry = %+v", r)
	}
	if a := m.Entries[2]; a.League != "" || a.Scope != "site" {
		t.Errorf("adp entry = %+v, want a site export with no league", a)
	}
	got, _, err := m.ResolveWithGaps(context.Background(), b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got[1].Body) != "old" {
		t.Errorf("rosters resolved to %q, want the pinned body, not the newest", got[1].Body)
	}
	if got[0].Body != nil || got[0].Entry.MFLError == "" {
		t.Errorf("polls resolved to %q, want an MFL error body never read as data", got[0].Body)
	}
}

func TestResolveNamesEveryMissingBodyAndNeverFallsBackToANewerOne(t *testing.T) {
	fetches, b := archive()
	m, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	delete(b, sum("old"))
	delete(b, sum("adp"))
	_, _, err = m.ResolveWithGaps(context.Background(), b, nil)
	if err == nil || !strings.Contains(err.Error(), "2013/weekly/rosters_W01.json (sha256 "+sum("old")) ||
		!strings.Contains(err.Error(), "2026/adp.json") {
		t.Fatalf("err = %v, want both missing entries named", err)
	}
	b[sum("adp")] = "tampered"
	b[sum("old")] = "old"
	if _, _, err = m.ResolveWithGaps(context.Background(), b, nil); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("err = %v, want a body that fails its sha256 refused", err)
	}
}

func TestBuildKeepsUnheldPinsAndIgnoresCapturesAfterTheCutoff(t *testing.T) {
	fetches, _ := archive()
	p := append(pins(), manifest.Pin{File: "2013/weekly/rosters_W01.json", SHA256: sum("new")})
	p[1] = manifest.Pin{File: "2013/assets.json", SHA256: sum("gone")}
	m, err := manifest.Build("frozen", p, fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	missing := m.Missing()
	if len(missing) != 2 || missing[0].File != "2013/assets.json" || missing[0].League != "51719" ||
		missing[1].SHA256 != sum("new") || missing[1].CapturedAt != "" {
		t.Fatalf("missing = %+v, want the unarchived assets and the capture afterTime() the cutoffTime()", missing)
	}
}

func TestIdentityIgnoresRowLocatorsAndSurvivesARoundTrip(t *testing.T) {
	fetches, _ := archive()
	m, _ := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	for i := range fetches {
		fetches[i].ID += 100
	}
	moved, _ := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if moved.SHA256 != m.SHA256 {
		t.Error("the same inputs in another database must keep the manifest's sha256")
	}
	enc, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := manifest.Decode("frozen", m.SHA256, enc)
	if err != nil || back.Entries[1].FetchID != 1 {
		t.Fatalf("round trip = %+v, %v", back, err)
	}
	tampered := strings.Replace(string(enc), sum("adp"), sum("other"), 1)
	if _, err := manifest.Decode("frozen", m.SHA256, []byte(tampered)); err == nil {
		t.Error("a changed entry decoded under the old sha256")
	}
}

func TestStatusKeepsTheLastGoodSnapshotStale(t *testing.T) {
	want := manifest.Want{ManifestSHA: "m1", Compiler: "c1", RulesHash: "r1"}
	good := manifest.Attempt{ManifestSHA: "m1", Compiler: "c1", RulesHash: "r1", ResultSHA: "res", BuiltAt: beforeTime()}
	failed := manifest.Attempt{ManifestSHA: "m1", Compiler: "c1", RulesHash: "r1", Error: "check 3 failed", BuiltAt: afterTime()}
	cases := []struct {
		name     string
		attempts []manifest.Attempt
		want     manifest.Want
		state    manifest.State
		reason   string
	}{
		{"nothing yet", nil, want, manifest.StateNone, "Nothing has been compiled"},
		{"only failures", []manifest.Attempt{failed}, want, manifest.StateNone, "check 3 failed"},
		{"current", []manifest.Attempt{good}, want, manifest.StateCurrent, ""},
		{"failed after good", []manifest.Attempt{failed, good}, want, manifest.StateStale, "last good copy is kept: check 3 failed"},
		{"inputs moved", []manifest.Attempt{good}, manifest.Want{ManifestSHA: "m2", Compiler: "c1", RulesHash: "r1"},
			manifest.StateStale, "inputs changed"},
		{"compiler moved", []manifest.Attempt{good}, manifest.Want{ManifestSHA: "m1", Compiler: "c2", RulesHash: "r1"},
			manifest.StateStale, "compiler changed"},
		{"rules moved", []manifest.Attempt{good}, manifest.Want{ManifestSHA: "m1", Compiler: "c1", RulesHash: "r2"},
			manifest.StateStale, "rules changed"},
	}
	for _, c := range cases {
		s := manifest.StatusOf(c.attempts, c.want)
		if s.State != c.state || !strings.Contains(s.Reason, c.reason) {
			t.Errorf("%s: status = %+v, want %s with %q", c.name, s, c.state, c.reason)
		}
		if c.state == manifest.StateStale && s.Good.ResultSHA != "res" {
			t.Errorf("%s: the last good snapshot was not kept: %+v", c.name, s.Good)
		}
	}
}

func TestUnheldEntryCannotResolveThroughAnUnrelatedBody(t *testing.T) {
	fetches, b := archive()
	p := []manifest.Pin{{File: "2013/weekly/rosters_W01.json", SHA256: sum("new")}}
	m, err := manifest.Build("frozen", p, fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ResolveWithGaps(context.Background(), b, nil); err == nil || !strings.Contains(err.Error(), "no pinned archive row") {
		t.Fatalf("resolution = %v; must refuse the matching bytes captured after the cutoff", err)
	}
}

func TestConflictingHistoricalLeagueIDsAreRefused(t *testing.T) {
	fetches, _ := archive()
	fetches = append(fetches, manifest.Fetch{ID: 5, URL: strings.Replace(rosters2013, "51719", "14432", 1), SHA256: sum("old"), FetchedAt: beforeTime()})
	if _, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime()); err == nil || !strings.Contains(err.Error(), "conflicting league ids") {
		t.Fatalf("conflicting leagues accepted: %v", err)
	}
}

func TestCuratedInputsArePinnedResolvedAndChangeTheIdentity(t *testing.T) {
	fetches, b := archive()
	m, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	withNotes, err := m.WithCurated(manifest.Pin{File: "note_readings.csv", SHA256: sum("readings")})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := m.WithCurated(manifest.Pin{File: "note_readings.csv", SHA256: sum("readings, edited")})
	if err != nil {
		t.Fatal(err)
	}
	if withNotes.SHA256 == m.SHA256 || withNotes.SHA256 == edited.SHA256 || len(withNotes.Missing()) != len(m.Missing()) {
		t.Fatal("a curated input must change the identity and never count as missing from the archive")
	}
	b[sum("readings")] = "readings"
	got, _, err := withNotes.ResolveWithGaps(context.Background(), b, nil)
	if err != nil {
		t.Fatal(err)
	}
	var curated, login int
	for _, r := range got {
		if r.Entry.Curated() && string(r.Body) == "readings" {
			curated++
		}
		if r.Entry.File == "2013/polls.json" && r.Body == nil && r.Bytes == int64(len("login")) {
			login++
		}
	}
	if curated != 1 || login != 1 {
		t.Fatalf("curated %d, error body sized %d", curated, login)
	}
	delete(b, sum("readings"))
	if _, _, err := withNotes.ResolveWithGaps(context.Background(), b, nil); err == nil || !strings.Contains(err.Error(), "note_readings.csv") {
		t.Fatalf("an unresolvable curated input must fail by name: %v", err)
	}
	if _, err := m.WithCurated(manifest.Pin{File: "x.csv", SHA256: "nothex"}); err == nil {
		t.Fatal("accepted an invalid curated pin")
	}
}
