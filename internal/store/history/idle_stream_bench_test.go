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

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// One operation is one 250ms polling round for 24 established idle streams: 24 committed-head
// queries, and on every second round (the 500ms session ticker) 24 device-session checks against
// the account tables sharing the ledger's pools, as hosted mode composes them. Stream startup,
// timers and TLS writes are excluded.
func BenchmarkIdleStreamHeads100k(b *testing.B) {
	s, pools := openBigLeague(b, filepath.Join(b.TempDir(), "idle.db"))
	seedIdleStreamLeague(b, s)
	sessions, devices := idleStreamSessions(b, pools)
	before := benchmarkProcessCPU(b)
	b.ReportAllocs()
	b.ResetTimer()
	rounds, checks := 0, 0
	for b.Loop() {
		for _, device := range devices {
			head, err := s.CommittedHead(b.Context(), bigLeagueID, commissioner.Stream)
			if err != nil || head != 100_000 {
				b.Fatalf("head=%d err=%v", head, err)
			}
			if rounds%2 == 1 {
				if _, err := sessions.Check(b.Context(), device); err != nil {
					b.Fatalf("session check: %v", err)
				}
				checks++
			}
		}
		rounds++
	}
	b.StopTimer()
	cpu := benchmarkProcessCPU(b) - before
	b.ReportMetric(float64(cpu.Nanoseconds())/float64(b.N), "cpu-ns/op")
	b.ReportMetric(24, "head-queries/op")
	b.ReportMetric(float64(checks)/float64(b.N), "session-checks/op")
	b.ReportMetric(0, "ledger-reads/op")
}

// idleStreamSessions signs one owner in on 24 devices, one per stream, in the ledger's database.
func idleStreamSessions(b *testing.B, pools *db.Pools) (*accounts.SessionService, []accounts.RawSession) {
	b.Helper()
	store := accountstore.New(pools)
	if err := store.Initialize(b.Context()); err != nil {
		b.Fatal(err)
	}
	passwords := accounts.New(store)
	credentials := accounts.RawCredentials{Username: "stream.owner", Password: "stream-owner-password"}
	if _, err := passwords.LocalBootstrap(b.Context(), credentials); err != nil {
		b.Fatal(err)
	}
	sessions := accounts.NewSessions(passwords, store)
	devices := make([]accounts.RawSession, 24)
	for i := range devices {
		session, err := sessions.Login(b.Context(), credentials, fmt.Sprintf("stream device %d", i))
		if err != nil {
			b.Fatal(err)
		}
		devices[i] = accounts.RawSession{Bearer: session.Bearer}
	}
	return sessions, devices
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
