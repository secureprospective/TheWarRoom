package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// binaryBox is the real twr-server (practice build) running in a temp directory with a fake clock file,
// live-frames directory and official-statistics directory (cmd/twr-server/clock_practice.go), and twr
// connected to it as an owner's agent.
type binaryBox struct {
	dir, clockFile, liveDir, officialDir string
	at                                   time.Time
	env                                  env
	out                                  *bytes.Buffer
}

// setClock writes the server's fake clock; the server reads it plus the wall time since the write.
func (b *binaryBox) setClock(t *testing.T, at time.Time) {
	t.Helper()
	b.at = at
	if err := os.WriteFile(b.clockFile, []byte(at.UTC().Format(time.RFC3339Nano)), 0o600); err != nil {
		t.Fatal(err)
	}
}

// twr runs one twr command against the server and returns what it printed.
func (b *binaryBox) twr(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	b.out.Reset()
	err := b.env.runTwr(t, stdin, args...)
	return b.out.String(), err
}

// read is `twr read <command> -` with the scope on stdin, decoded into out.
func (b *binaryBox) read(t *testing.T, command string, scope map[string]int, out any) {
	t.Helper()
	body, _ := json.Marshal(scope)
	printed, err := b.twr(t, string(body), "read", command, "-")
	if err != nil {
		t.Fatalf("twr read %s: %v", command, err)
	}
	if err := json.Unmarshal([]byte(printed), out); err != nil {
		t.Fatalf("twr read %s printed %q: %v", command, printed, err)
	}
}

// useOfficial puts one weekly file (and the play-by-play) where the server's official job reads them.
func (b *binaryBox) useOfficial(t *testing.T, weekly string) {
	t.Helper()
	for from, to := range map[string]string{weekly: "stats_player_week_2026.csv.gz", "play_by_play_2026.csv.gz": "play_by_play_2026.csv.gz"} {
		body, err := os.ReadFile(filepath.Join("testdata/official", from))
		if err != nil {
			t.Fatal(err)
		}
		temp := filepath.Join(b.officialDir, to+".tmp")
		if err := os.WriteFile(temp, body, 0o600); err != nil { //nolint:gosec // G703: fixed names under the test temp dir.
			t.Fatal(err)
		}
		if err := os.Rename(temp, filepath.Join(b.officialDir, to)); err != nil {
			t.Fatal(err)
		}
	}
}

// useLiveFrame puts one recorded ESPN summary frame where the server's practice live job reads it.
func (b *binaryBox) useLiveFrame(t *testing.T, bodies map[string][]byte, name string) {
	t.Helper()
	body, ok := bodies[name]
	if !ok {
		t.Fatalf("missing replay frame %s", name)
	}
	temp := filepath.Join(b.liveDir, name+".tmp")
	if err := os.WriteFile(temp, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temp, filepath.Join(b.liveDir, name)); err != nil {
		t.Fatal(err)
	}
}

// awaitProvisional polls results.read via twr read until franchise.01's week-1 provisional points satisfy ok.
func (b *binaryBox) awaitProvisional(t *testing.T, ok func(int64) bool) int64 {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		var results commissioner.NativeResults
		b.read(t, "results.read", map[string]int{"season": 2026, "week": 1}, &results)
		if score, found := franchiseScore(results, "franchise.01"); found && score.Provisional != nil && score.Points == nil && ok(int64(*score.Provisional)) {
			return int64(*score.Provisional)
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(b.dir, "server.log"))
			t.Fatalf("franchise.01 week 1 provisional points never moved; server log:\n%s", log)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

// awaitPoints polls results.read until franchise.01's week-1 final points satisfy ok (the server's official job
// runs every second in the practice build).
func (b *binaryBox) awaitPoints(t *testing.T, ok func(int64) bool) int64 {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		var results commissioner.NativeResults
		b.read(t, "results.read", map[string]int{"season": 2026, "week": 1}, &results)
		if score, found := franchiseScore(results, "franchise.01"); found && score.Points != nil && ok(int64(*score.Points)) {
			return int64(*score.Points)
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(b.dir, "server.log"))
			t.Fatalf("franchise.01 week 1 final points never arrived; server log:\n%s", log)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// seedBinaryBox builds the server database offline, as tools/twr-practice does: the workflow league, the
// commissioner's franchise.01 lineup and an agent token for franchise.01. It returns the football digest and token.
func seedBinaryBox(t *testing.T, dir string) (string, accounts.AgentToken) {
	t.Helper()
	ctx := t.Context()
	pools, err := db.Open(ctx, filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	credentials := accountstore.New(pools)
	raw := accounts.RawCredentials{Username: "commissioner", Password: rand.Text()}
	principal, err := hostedfixture.Bootstrap(ctx, credentials, raw)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := hostedfixture.CreateWorkflowLeague(ctx, store, principal)
	if err != nil {
		t.Fatal(err)
	}
	football, err := store.LoadFootball(ctx, digest)
	if err != nil {
		t.Fatal(err)
	}
	store.SetPinnedFootball(football)
	assets, err := store.NativeAssets(ctx, "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExecuteCommissioner(ctx, boxLineup(assets.Revision, "box.lineup", "0099"), string(principal), nil); err != nil {
		t.Fatalf("record franchise.01 lineup: %v", err)
	}
	if _, err := store.LinkPlayerIDs(ctx, "dynastyprocess", []history.PlayerIDLink{
		{IDType: "espn", IDValue: "4360310", PlayerID: "0099"},
		{IDType: "gsis", IDValue: "00-0000099", PlayerID: "0099"},
	}); err != nil {
		t.Fatalf("link replay quarterback ids: %v", err)
	}
	human, err := accounts.NewSessions(accounts.New(credentials), credentials).Login(ctx, raw, "season box")
	if err != nil {
		t.Fatal(err)
	}
	scope := accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Modes: []string{"read", "preview"},
		Commands: []string{"results.read", "standings.read", "commish.ledger.read", "native.assets.read",
			"native.directory.read", commissioner.LineupSet}}
	token, err := accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store}).Issue(ctx, human.RawSession, scope)
	if err != nil {
		t.Fatal(err)
	}
	return digest, token
}

// boxLineup is franchise.01's week-1 lineup command.
func boxLineup(expected int64, request string, starters ...string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet, Request: request, Effective: "now",
		Visibility: "private", Expected: expected,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: starters}}}
}

// startBinaryBox builds twr-server (practice tag), seeds it, starts it on a free loopback port and connects twr.
func startBinaryBox(t *testing.T) *binaryBox {
	t.Helper()
	dir := t.TempDir()
	b := &binaryBox{dir: dir, clockFile: filepath.Join(dir, "clock"), liveDir: filepath.Join(dir, "live"), officialDir: filepath.Join(dir, "official"), out: &bytes.Buffer{}}
	if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // G302: directories need owner traversal.
		t.Fatal(err)
	}
	// The practice build insists on a private disposable layout (cmd/twr-server/backup_practice.go).
	for _, name := range []string{"live", "official", "home", "config", "data", "cache", "runtime"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// The recorded ESPN summaries are week 9; replay them as league week 1 (same as TestSeasonInABoxSpike).
	if err := os.WriteFile(filepath.Join(b.liveDir, "week"), []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := filepath.Join(dir, "twr-server")
	key, pem := filepath.Join(dir, "server.key"), filepath.Join(dir, "server.pem")
	for _, args := range [][]string{
		{"go", "build", "-tags", "practice", "-o", server, "../twr-server"},
		{"openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes", "-keyout", key, "-out", pem,
			"-days", "1", "-subj", "/CN=Season box", "-addext", "subjectAltName=IP:127.0.0.1"},
	} {
		if output, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec // G204: fixed test commands.
			t.Fatalf("%s: %v\n%s", args[0], err, output)
		}
	}
	if err := os.Chmod(key, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, token := seedBinaryBox(t, dir)
	b.setClock(t, time.Now().UTC()) // football kickoffs are at seed time + 24h
	registry, err := os.ReadFile("../../docs/ui/endpoint-registry.csv")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{"registry.csv": registry, "authenticator.key": bytes.Repeat([]byte{23}, 32)} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	db := filepath.Join(dir, "server.db")
	if output, err := exec.CommandContext(t.Context(), server, "upgrade", "-db", db, "-save", filepath.Join(dir, "pre-upgrade")).CombinedOutput(); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, output)
	}
	log, err := os.Create(filepath.Join(dir, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	child := exec.CommandContext(ctx, server, "hosted", "-football", digest, "-db", db, "-listen", "127.0.0.1:0",
		"-tls-cert", pem, "-tls-key", key, "-totp-key-file", filepath.Join(dir, "authenticator.key"),
		"-preview-registry", filepath.Join(dir, "registry.csv"), "-official-stats")
	child.Env = append(os.Environ(), "TWR_PRACTICE_CLOCK_FILE="+b.clockFile, "TWR_PRACTICE_LIVE_DIR="+b.liveDir, "TWR_PRACTICE_OFFICIAL_DIR="+b.officialDir)
	child.Stdout, child.Stderr = log, log
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); _ = child.Wait(); _ = log.Close() })
	address := awaitOrigin(t, filepath.Join(dir, "server.log"))
	certificate, err := os.ReadFile(pem)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certificate)
	memory := &memoryStore{tokens: map[string]string{address + " native.hosted": token.Bearer}}
	b.env = env{configDir: t.TempDir(), roots: roots, out: b.out,
		keychain: func(origin, league string) tokenStore { return memoryKey{store: memory, key: origin + " " + league} }}
	if err := b.env.save(connection{Address: address, League: "native.hosted", Helper: token.ID}); err != nil {
		t.Fatal(err)
	}
	return b
}

// awaitOrigin waits for the server to print its https loopback origin.
func awaitOrigin(t *testing.T, logPath string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		data, _ := os.ReadFile(logPath)
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, "https://127.0.0.1:") {
				return strings.TrimSpace(line)
			}
		}
		if time.Now().After(deadline) || strings.Contains(string(data), "practice ") {
			t.Fatalf("server did not start:\n%s", data)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestSeasonInABoxBinary is the season box against the real twr-server binary (B4): a subprocess on a temp port
// and temp data, a fake clock it obeys, live frames replayed from testdata/live-week9.tar.gz by the server's
// practice live job, official statistics replayed from testdata/official by the server's own official job, and
// every league action through twr. Draft stays in TestSeasonInABoxSpike.
func TestSeasonInABoxBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the twr-server binary")
	}
	b := startBinaryBox(t)

	// STEP 1 league created: twr ledger over HTTPS from the binary.
	printed, err := b.twr(t, "", "ledger")
	if err != nil {
		t.Fatal(err)
	}
	var ledger commissioner.Ledger
	if err := json.Unmarshal([]byte(printed), &ledger); err != nil || ledger.Revision == 0 {
		t.Fatalf("league not served: %v %s", err, printed)
	}
	t.Log("STEP 1 binary serves the league; twr ledger revision", ledger.Revision)

	// STEP 3 lineup: recorded before start (G2 needs a human confirmation); twr preview admits a change before lock.
	early := boxLineup(ledger.Revision, "box.lineup.early", "0099")
	body, _ := json.Marshal(early)
	if printed, err = b.twr(t, string(body), "preview", "-"); err != nil || !strings.Contains(printed, `"blockers": []`) {
		t.Fatalf("twr preview before kickoff refused: %v %s", err, printed)
	}
	t.Log("STEP 3 twr preview of franchise.01's lineup admitted before kickoff")

	// STEP 4 lineup lock: the fake clock passes kickoff; the server refuses the same preview.
	b.setClock(t, b.at.Add(25*time.Hour))
	late := boxLineup(ledger.Revision, "box.lineup.late", "0099", "0101")
	body, _ = json.Marshal(late)
	printed, err = b.twr(t, string(body), "preview", "-")
	if err != nil || !strings.Contains(printed, "locked at kickoff") {
		t.Fatalf("a lineup previewed after kickoff was admitted: %s", printed)
	}
	t.Log("STEP 4 lineup lock held by the binary after the fake clock passed kickoff: the preview names \"locked at kickoff\"")

	// STEP 5 live frames: recorded ESPN summaries replayed from disk; twr read sees provisional points move.
	frames := replayFrames(t)
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-01-progress.json")
	first := b.awaitProvisional(t, func(p int64) bool { return p > 0 })
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-02-progress.json")
	second := b.awaitProvisional(t, func(p int64) bool { return p > first })
	t.Logf("STEP 5 live scores move (twr read): franchise.01 %d -> %d (provisional, frames 01 -> 02)", first, second)

	// STEP 6 final: the clock passes week 1's games; the server's official job pins the files on its own.
	b.setClock(t, b.at.Add(6*time.Hour))
	b.useOfficial(t, "stats_player_week_2026.csv.gz")
	final := b.awaitPoints(t, func(int64) bool { return true })
	t.Logf("STEP 6 official final pinned by the server's official job: franchise.01 %d", final)

	// STEP 7 stat correction: a changed weekly file; the server re-pins and the points move.
	b.setClock(t, b.at.Add(time.Hour))
	b.useOfficial(t, "stats_player_week_2026_corrected.csv.gz")
	corrected := b.awaitPoints(t, func(p int64) bool { return p != final })
	t.Logf("STEP 7 stat correction: franchise.01 %d -> %d", final, corrected)

	// STEP 8 standings.
	var standings commissioner.NativeStandings
	b.read(t, "standings.read", map[string]int{"season": 2026}, &standings)
	if standings.Status == "Needs a ruling" {
		regressed(t, 8, "standings need a ruling: "+standings.Words)
	}
	t.Logf("STEP 8 standings.read: status %q, %d decided, %d pending", standings.Status, standings.Decided, standings.Pending)
	notReached(t, 2, "binary box: draft runs only in-process (TestSeasonInABoxSpike); "+
		"owner picks are refused at the door (B7)")
}
