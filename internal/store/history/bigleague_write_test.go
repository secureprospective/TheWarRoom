//go:build leaguebench

package history

import (
	"context"
	"fmt"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"path/filepath"
	"testing"
	"time"
)

// Minimal valid rule-only history isolates ledger length from roster workload. Setup uses
// the production append serializer in one transaction, not the O(n²) command trial path.
func seedLeagueWrite(t testing.TB, s *Store, size int) *bigLeagueBuilder {
	t.Helper()
	g := &bigLeagueBuilder{t: t, s: s, now: time.Date(2013, 4, 1, 12, 0, 0, 0, time.UTC)}
	s.SetClock(func() time.Time { return g.now })
	g.command("creator", commissioner.Create, commissioner.RawArgs{Name: "Write growth", Preset: commissioner.PresetLegacy, Season: 2013, FranchiseCount: 32}, false)
	tx, err := s.pools.Write().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for g.revision < int64(size) {
		raw := commissioner.RawCommand{League: bigLeagueID, Command: commissioner.RulesSet,
			Args: commissioner.RawArgs{Field: "cap.amount", Value: []byte(`100000000000000`)}, Expected: g.revision,
			Request: fmt.Sprintf("generated.%08d", g.revision), Reason: "Deterministic growth fixture", Effective: "now", Visibility: "league"}
		prepared, canonical, err := preparedCommand(raw, "creator", true)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := s.appendCommissioner(context.Background(), tx, appendRequest{raw: prepared, principal: "creator", role: "commissioner", canonical: canonical, at: g.now})
		if err != nil {
			t.Fatal(err)
		}
		g.revision = receipt.Revision
		g.now = g.now.Add(time.Microsecond)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return g
}

func BenchmarkLeagueWrite(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000, 20000, 50000} {
		b.Run(fmt.Sprintf("records=%d", size), func(b *testing.B) {
			s, _ := openBigLeague(b, filepath.Join(b.TempDir(), "write.db"))
			g := seedLeagueWrite(b, s, size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				g.set("cap.amount", `100000000000000`)
			}
			b.StopTimer()
			b.Logf("ledger start=%d end=%d (use -benchtime=1x for exact size)", size, g.revision)
		})
	}
}
