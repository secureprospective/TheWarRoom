package history

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func emptyLeagueFile(t *testing.T) *db.Pools {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "league.db"), LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	if err := PrepareNewLeague(t.Context(), pools); err != nil {
		t.Fatal(err)
	}
	return pools
}

const otherLeagueRow = `INSERT INTO league_recovery_log(sequence,league,revision,recorded_at,message) VALUES(999,'native.other',1,'2026-11-14T00:00:00Z','other league')`

// Each league of a shared file moves into its own file with exactly its own rows and a new audit chain
// over its own events; the file then refuses any other league's rows; the split commit drops the shared
// league tables only while the shared chain still ends where the split began.
func TestSplitMovesEachLeagueIntoItsOwnFile(t *testing.T) {
	ctx := t.Context()
	s, _, shared := newStore(t)
	if _, err := s.StandUpRosters(ctx, perSidePoint()); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Write().ExecContext(ctx, otherLeagueRow); err != nil {
		t.Fatal(err)
	}
	leagues, err := SharedLeagues(ctx, shared)
	if err != nil || !slices.Equal(leagues, []string{"native.mirror", "native.other"}) {
		t.Fatalf("shared leagues %v, %v", leagues, err)
	}
	head, err := SharedAuditHead(ctx, shared)
	if err != nil || head.Size == 0 {
		t.Fatalf("shared head %+v, %v", head, err)
	}
	files := map[string]*db.Pools{}
	for _, league := range leagues {
		target := emptyLeagueFile(t)
		if err := CopyLeague(ctx, shared, target, LeagueFile{League: league, Shared: head}); err != nil {
			t.Fatal(err)
		}
		if _, err := ProveLeague(ctx, shared, target, league); err != nil {
			t.Fatal(err)
		}
		files[league] = target
	}
	var events, logs int
	if err := files["native.mirror"].Read().QueryRowContext(ctx, `SELECT (SELECT count(*) FROM league_event),(SELECT count(*) FROM league_recovery_log WHERE league!='native.mirror')`).Scan(&events, &logs); err != nil {
		t.Fatal(err)
	}
	if int64(events) != head.Size || logs != 0 {
		t.Fatalf("mirror file: %d events (shared chain %d), %d foreign log rows", events, head.Size, logs)
	}
	if err := files["native.other"].Read().QueryRowContext(ctx, `SELECT (SELECT count(*) FROM league_event),(SELECT count(*) FROM league_recovery_log)`).Scan(&events, &logs); err != nil {
		t.Fatal(err)
	}
	if events != 0 || logs != 1 {
		t.Fatalf("other file: %d events, %d log rows", events, logs)
	}
	file, err := ReadLeagueFile(ctx, files["native.mirror"])
	if err != nil || file.League != "native.mirror" || file.Shared != head {
		t.Fatalf("league file %+v, %v", file, err)
	}
	if _, err := files["native.mirror"].Write().ExecContext(ctx, otherLeagueRow); err == nil || !strings.Contains(err.Error(), "belongs to another league") {
		t.Fatalf("a mirror file took another league's row: %v", err)
	}
	stale := head
	stale.Size--
	if err := CommitLeagueSplit(ctx, shared, LeagueSplit{Leagues: leagues, Shared: stale, At: time.Now()}); err == nil {
		t.Fatal("the split committed over a shared chain that moved")
	}
	if found, _ := db.HasTable(ctx, shared, "league_event"); !found {
		t.Fatal("a refused split dropped the shared tables")
	}
	at := time.Date(2026, 11, 14, 16, 0, 0, 0, time.UTC)
	if err := CommitLeagueSplit(ctx, shared, LeagueSplit{Leagues: leagues, Shared: head, At: at}); err != nil {
		t.Fatal(err)
	}
	split, ok, err := ReadLeagueSplit(ctx, shared)
	if err != nil || !ok || !slices.Equal(split.Leagues, leagues) || split.Shared != head || !split.At.Equal(at) {
		t.Fatalf("split record %+v, %v, %v", split, ok, err)
	}
	for _, table := range append(leagueTables(), "audit_chain") {
		if found, _ := db.HasTable(ctx, shared, table); found {
			t.Fatalf("shared file still holds %s", table)
		}
	}
	if err := MigrateLeague(ctx, shared); err != nil {
		t.Fatalf("the split shared file does not open: %v", err)
	}
}
