//go:build leaguebench && linux

package history

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// One operation is one polling round for 24 established idle streams. This isolates
// the committed-head gate from stream startup, session checks, timers and TLS writes.
func BenchmarkIdleStreamHeads100k(b *testing.B) {
	s, _ := openBigLeague(b, filepath.Join(b.TempDir(), "idle.db"))
	seedIdleStreamLeague(b, s)
	before := benchmarkProcessCPU(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for range 24 {
			head, err := s.CommittedHead(b.Context(), bigLeagueID, commissioner.Stream)
			if err != nil || head != 100_000 {
				b.Fatalf("head=%d err=%v", head, err)
			}
		}
	}
	b.StopTimer()
	cpu := benchmarkProcessCPU(b) - before
	b.ReportMetric(float64(cpu.Nanoseconds())/float64(b.N), "cpu-ns/op")
	b.ReportMetric(24, "head-queries/op")
	b.ReportMetric(0, "ledger-reads/op")
}

func benchmarkProcessCPU(b *testing.B) time.Duration {
	b.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}

// Use the same rule-only 100k workload as the replay benchmark, with explicit durable
// revisions: recipient tokens returned by hosted commands are not stream revisions.
func seedIdleStreamLeague(b *testing.B, s *Store) {
	b.Helper()
	g := seedLeagueWrite(b, s, 1)
	tx, err := s.pools.Write().BeginTx(b.Context(), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for revision := int64(2); revision <= 100_000; revision++ {
		raw := commissioner.RawCommand{League: bigLeagueID, Command: commissioner.RulesSet,
			Args:     commissioner.RawArgs{Field: "cap.amount", Value: []byte(fmt.Sprintf("%d", 100000000000000+revision%2))},
			Expected: revision - 1, Request: fmt.Sprintf("idle.%d", revision), Reason: "Deterministic growth fixture", Effective: "now", Visibility: "league"}
		prepared, canonical, err := preparedCommand(raw, "creator", true)
		if err != nil {
			b.Fatal(err)
		}
		receipt := commissioner.Receipt{Entry: raw.Request, League: bigLeagueID, Revision: revision, RecordedAt: g.now}
		record := commissioner.Record{CommandVersion: commissioner.DurableCommandVersion, Command: prepared,
			Principal: "creator", Role: "commissioner", Gravity: prepared.Gravity, Canonical: canonical, Receipt: receipt}
		hash := sha256.Sum256([]byte(canonical))
		event := LeagueEvent{ID: receipt.Entry, League: bigLeagueID, Stream: commissioner.Stream,
			EffectiveAt: g.now, RecordedAt: g.now, Actor: "creator", Scope: "league", Kind: raw.Command,
			Payload: EventPayload{Version: eventPayloadVersion, Commissioner: &record}, InputHash: hex.EncodeToString(hash[:]), CorrelationID: raw.Request}
		if _, err := s.insertEvent(b.Context(), tx, event, revision-1); err != nil {
			b.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
}
