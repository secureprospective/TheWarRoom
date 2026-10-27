package history

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func sleeperImport(t *testing.T) RawFootballEvidence {
	t.Helper()
	raw := scoringImport(t)
	player := raw.Players["0099"]
	player.Identity, player.Sleeper, player.Facts.Name, player.Facts.Position = "", "12944", "Dorian Mausi", leaguerules.LB
	raw.Players["17367"] = player
	player.Sleeper, player.Facts.Name = "13000", "Second direct mapping"
	raw.Players["17368"] = player
	raw.Stats[0].Sleeper = []source.RawSleeperWeek{{Season: raw.Season, Week: 1, Body: `{"12944":{"st_snp":19,"st_tkl_solo":1,"penalty":1}}`}}
	return raw
}

func TestSleeperEvidenceRoundTrip(t *testing.T) {
	for _, split := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline", true: "split"}[split], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.db")
			s, pools := statStore(t, path)
			raw := sleeperImport(t)
			want := raw.Stats[0].Sleeper
			if split {
				stat := raw.Stats[0]
				body, sha := weekPart(t, RawFootballWeek{Version: 1, Season: raw.Season, Week: 1, Weekly: stat.Weekly, Plays: stat.Plays, Sleeper: stat.Sleeper})
				if err := s.ImportFootballWeek(t.Context(), body, sha); err != nil {
					t.Fatal(err)
				}
				raw.Stats[0] = RawFootballStats{AsOf: stat.AsOf, Parts: []string{sha}}
			}
			pinned := installScoringImport(t, s, raw)
			digest := pinned.Digest()
			if err := pools.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, _ := statStore(t, path)
			restored, err := reopened.LoadFootball(t.Context(), digest)
			if err != nil {
				t.Fatal(err)
			}
			if restored.raw.Players["17367"].Sleeper != "12944" || restored.raw.Players["17367"].Identity != "" {
				t.Fatal("direct MFL bridge changed")
			}
			bodies, sha, err := restored.raw.revisionBodies(restored.raw.Stats[0], reopened.storeParts(t.Context()))
			if err != nil || len(bodies) != 1 || !reflect.DeepEqual(bodies[0].Sleeper, want) {
				t.Fatalf("round trip: %+v, %v", bodies, err)
			}
			if sha != pinned.stats[0].sha {
				t.Fatal("revision fingerprint changed after reload")
			}
			held, err := reopened.Body(t.Context(), digest)
			if err != nil || footballDigest(held) != digest {
				t.Fatalf("selected body changed: %v", err)
			}
		})
	}
}

func TestSleeperLegacyEvidenceBytesAndDigests(t *testing.T) {
	stats := RawFootballStats{AsOf: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Weekly: "weekly", Plays: "plays"}
	body, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"asOf":"2025-01-01T00:00:00Z","weekly":"weekly","plays":"plays"}` {
		t.Fatalf("legacy inline bytes changed: %s", body)
	}
	part := RawFootballWeek{Version: 1, Season: 2025, Week: 1, Weekly: "weekly", Plays: "plays"}
	body, err = json.Marshal(part)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"version":1,"season":2025,"week":1,"weekly":"weekly","plays":"plays"}` {
		t.Fatalf("legacy part bytes changed: %s", body)
	}
	raw := RawFootballEvidence{Season: 2025}
	_, sha, err := raw.revisionBodies(stats, nil)
	if err != nil || sha != "b29c61613eac6df24326df3f756fc6544303fbdefbbda523461aa78dc4705d7e" {
		t.Fatalf("legacy inline SHA = %s (%v)", sha, err)
	}
	// The split fingerprint is still the exact digest of the ordered part list.
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	legacy := splitImport(t, s)
	_, sha, err = legacy.revisionBodies(legacy.Stats[0], s.storeParts(t.Context()))
	want := footballDigest([]byte("parts:" + strings.Join(legacy.Stats[0].Parts, ",")))
	if err != nil || sha != want {
		t.Fatalf("legacy split SHA = %s, want %s (%v)", sha, want, err)
	}
}

func TestSleeperFingerprintIncludesBodyScopeAndBridge(t *testing.T) {
	raw := sleeperImport(t)
	_, before, err := raw.revisionBodies(raw.Stats[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*RawFootballEvidence){
		"body":  func(r *RawFootballEvidence) { r.Stats[0].Sleeper[0].Body += " " },
		"scope": func(r *RawFootballEvidence) { r.Stats[0].Sleeper[0].Week = 2 },
		"bridge": func(r *RawFootballEvidence) {
			p := r.Players["17367"]
			delete(r.Players, "17367")
			r.Players["17369"] = p
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := sleeperImport(t)
			change(&changed)
			_, after, err := changed.revisionBodies(changed.Stats[0], nil)
			if err != nil || after == before {
				t.Fatalf("change did not fingerprint: %s (%v)", after, err)
			}
		})
	}
}

func TestSleeperFormatRefusesInvalidImports(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	for name, change := range map[string]func(*RawFootballEvidence){
		"other season":      func(r *RawFootballEvidence) { r.Stats[0].Sleeper[0].Season-- },
		"duplicate week":    func(r *RawFootballEvidence) { r.Stats[0].Sleeper = append(r.Stats[0].Sleeper, r.Stats[0].Sleeper[0]) },
		"invalid body":      func(r *RawFootballEvidence) { r.Stats[0].Sleeper[0].Body = `null` },
		"duplicate bridge":  func(r *RawFootballEvidence) { p := r.Players["0099"]; p.Sleeper = "12944"; r.Players["0099"] = p },
		"team bridge":       func(r *RawFootballEvidence) { p := r.Players["17367"]; p.Sleeper = "DAL"; r.Players["17367"] = p },
		"parts and Sleeper": func(r *RawFootballEvidence) { r.Stats[0].Parts = []string{strings.Repeat("a", 64)} },
	} {
		t.Run(name, func(t *testing.T) {
			raw := sleeperImport(t)
			change(&raw)
			body, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			sha := footballDigest(body)
			if err := s.ImportFootball(t.Context(), body, sha); err == nil {
				t.Fatal("invalid import accepted")
			}
			if _, err := s.Body(t.Context(), sha); err == nil {
				t.Fatal("invalid import was saved")
			}
		})
	}
}
