package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// binaryBox is the real twr-server (practice build) running in a temp directory with a fake clock file,
// live-frames directory and official-statistics directory (cmd/twr-server/clock_practice.go), twr
// connected to it as an owner's agent, and twr-check built beside it for the Tuesday report card.
type binaryBox struct {
	dir, clockFile, liveDir, officialDir string
	address, certPath, checkBin          string
	checkTokenFile                       string
	commishBearer, owner04Bearer         string
	at                                   time.Time
	env                                  env
	out                                  *bytes.Buffer
}

// binarySeed holds the seeded football digest, agent tokens, owner sessions and initial draft instant.
type binarySeed struct {
	digest        string
	token         accounts.AgentToken
	checkToken    accounts.AgentToken
	commishBearer string
	owner04Bearer string
	draftAt       time.Time
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

// ledgerRevision reads the league's current revision over HTTPS via twr ledger.
func (b *binaryBox) ledgerRevision(t *testing.T) int64 {
	t.Helper()
	printed, err := b.twr(t, "", "ledger")
	if err != nil {
		t.Fatalf("twr ledger: %v", err)
	}
	var ledger commissioner.Ledger
	if err := json.Unmarshal([]byte(printed), &ledger); err != nil || ledger.Revision == 0 {
		t.Fatalf("league not served: %v %s", err, printed)
	}
	return ledger.Revision
}

// read is `twr read <command>` (with optional scope on stdin), decoded into out.
func (b *binaryBox) read(t *testing.T, command string, scope map[string]int, out any) {
	t.Helper()
	var (
		printed string
		err     error
	)
	if scope == nil {
		printed, err = b.twr(t, "", "read", command)
	} else {
		body, _ := json.Marshal(scope)
		printed, err = b.twr(t, string(body), "read", command, "-")
	}
	if err != nil {
		t.Fatalf("twr read %s: %v", command, err)
	}
	if err := json.Unmarshal([]byte(printed), out); err != nil {
		t.Fatalf("twr read %s printed %q: %v", command, printed, err)
	}
}

// ownerPost sends one owner-door HTTPS request (PickPath or LineupPath) with the owner's session token.
func (b *binaryBox) ownerPost(t *testing.T, path, bearer string, payload serverhttp.RawNativeRequest) ([]byte, error) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, b.address+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearer)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: b.env.roots, MinVersion: tls.VersionTLS12}}}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	reply, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		return reply, fmt.Errorf("%s: %d %s", path, response.StatusCode, bytes.TrimSpace(reply))
	}
	return reply, nil
}

// ownerRevision reads the league version visible to bearer, the Expected token its own writes require.
func (b *binaryBox) ownerRevision(t *testing.T, bearer string) int64 {
	t.Helper()
	reply, err := b.ownerPost(t, serverhttp.LedgerPath, bearer, serverhttp.RawNativeRequest{Protocol: 1, League: "native.hosted"})
	if err != nil {
		t.Fatalf("owner ledger read: %v", err)
	}
	var ledger commissioner.Ledger
	if err := json.Unmarshal(reply, &ledger); err != nil || ledger.Revision == 0 {
		t.Fatalf("owner ledger unreadable: %v %s", err, reply)
	}
	return ledger.Revision
}

// quickDraft runs STEP 2 against the binary: franchise.01 and franchise.04 each draft a rookie through
// the owner door (PickPath, owner session token), then twr read confirms both rookies on their rosters.
func (b *binaryBox) quickDraft(t *testing.T) {
	t.Helper()
	picks := []struct {
		franchise, player, request, bearer string
	}{
		{"franchise.01", "5001", "box.pick.01", b.commishBearer},
		{"franchise.04", "5002", "box.pick.04", b.owner04Bearer},
	}
	for _, p := range picks {
		b.setClock(t, b.at.Add(time.Minute))
		input := commissioner.DraftPickInput{
			League: "native.hosted", Season: 2027, Franchise: p.franchise,
			Player: p.player, Expected: b.ownerRevision(t, p.bearer), Request: p.request,
		}
		if _, err := b.ownerPost(t, serverhttp.PickPath, p.bearer, serverhttp.RawNativeRequest{Protocol: 1, Pick: &input}); err != nil {
			t.Fatalf("STEP 2 owner pick %s (%s) refused: %v", p.franchise, p.player, err)
		}
	}
	var assets commissioner.NativeAssets
	b.read(t, "native.assets.read", nil, &assets)
	for _, p := range picks {
		if !rostered(assets, p.player, p.franchise) {
			t.Fatalf("STEP 2 twr read native.assets.read: rookie %s missing from %s", p.player, p.franchise)
		}
	}
	t.Log("STEP 2 quick draft (owner door PickPath): franchise.01 drafted 5001, franchise.04 drafted 5002; verified via twr read native.assets.read")
}

// setOpponentLineup records franchise.04's week-1 lineup (its drafted rookie 5002) through the owner door.
func (b *binaryBox) setOpponentLineup(t *testing.T) {
	t.Helper()
	cmd := commissioner.RawCommand{
		League: "native.hosted", Command: commissioner.LineupSet, Request: "box.lineup.04",
		Effective: "now", Visibility: "private", Expected: b.ownerRevision(t, b.owner04Bearer),
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: 2026, Franchise: "franchise.04", Week: 1, Starters: []string{"5002"},
		}},
	}
	if _, err := b.ownerPost(t, serverhttp.LineupPath, b.owner04Bearer, serverhttp.RawNativeRequest{Protocol: 1, Command: &cmd}); err != nil {
		t.Fatalf("STEP 3 franchise.04 owner lineup refused: %v", err)
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

// awaitNFLFinal polls nfl.games.read via twr read until the replayed week-1 game reports final ("post").
func (b *binaryBox) awaitNFLFinal(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		var games leagueservice.NFLGames
		b.read(t, "nfl.games.read", nil, &games)
		for _, g := range games.Board.Games {
			if g.Season == 2026 && g.Week == 1 && g.Status == "post" {
				return
			}
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(b.dir, "server.log"))
			t.Fatalf("nfl.games.read never reported a final week-1 game; server log:\n%s", log)
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

// runReportCard runs twr-check (read mode) against the binary and asserts all four Tuesday checks pass.
func (b *binaryBox) runReportCard(t *testing.T) string {
	t.Helper()
	detailPath := filepath.Join(b.dir, "reportcard.md")
	cmd := exec.CommandContext(t.Context(), b.checkBin, //nolint:gosec // G204: fixed test binary and args.
		"-server", b.address, "-token-file", b.checkTokenFile,
		"-league", "native.hosted", "-season", "2026", "-week", "1", "-out", detailPath)
	cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+b.certPath)
	out, err := cmd.CombinedOutput()
	summary := strings.TrimSpace(string(out))
	if err != nil {
		t.Fatalf("STEP 9 twr-check failed (%v):\n%s", err, summary)
	}
	for _, want := range []string{
		"Scores add up: 2 team scores add up",
		"GREEN   Official stats final:",
		"Standings match games:",
		"GREEN   Lineups locked at kickoff:",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("STEP 9 twr-check missing %q:\n%s", want, summary)
		}
	}
	if strings.Contains(summary, "RED") {
		t.Fatalf("STEP 9 twr-check reported RED:\n%s", summary)
	}
	detail, err := os.ReadFile(detailPath)
	if err != nil || !strings.Contains(string(detail), "Practice team 1") || !strings.Contains(string(detail), "Practice team 4") {
		t.Fatalf("STEP 9 report card detail missing teams: err=%v\n%s", err, detail)
	}
	return summary
}

// binaryBoxExtra runs playoff-specific setup inside seedBinaryBoxWith before auth tokens are issued.
type binaryBoxExtra func(t *testing.T, store *history.Store, credentials *accountstore.Store, principal governance.Principal, draftAt time.Time)

// seedBinaryBoxWith builds the server database offline, as tools/twr-practice does: the workflow league, a
// second seated owner on franchise.04, pick-2 draft order, franchise.01's lineup, optional extra setup,
// owner sessions, and agent tokens for twr and twr-check.
func seedBinaryBoxWith(t *testing.T, dir string, extra binaryBoxExtra) binarySeed {
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
	commishRaw := accounts.RawCredentials{Username: "commissioner", Password: rand.Text()}
	principal, err := hostedfixture.Bootstrap(ctx, credentials, commishRaw)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := hostedfixture.CreateWorkflowLeague(ctx, store, principal)
	if err != nil {
		t.Fatal(err)
	}
	ownerRaw := accounts.RawCredentials{Username: "season-box-owner", Password: rand.Text()}
	if err := hostedfixture.AddWorkflowOwner(ctx, store, credentials, principal, "franchise.04", ownerRaw); err != nil {
		t.Fatalf("seat franchise.04: %v", err)
	}
	draftAt := time.Now().UTC().Add(5 * time.Second)
	seedBinaryBoxLeague(t, store, digest, principal, draftAt)
	if extra != nil {
		extra(t, store, credentials, principal, draftAt)
	}
	return seedBinaryBoxAuth(t, store, credentials, digest, commishRaw, ownerRaw, draftAt)
}

// seedBinaryBoxLeague pins football, moves franchise.04 to pick 2, records franchise.01's lineup and links QB IDs.
func seedBinaryBoxLeague(t *testing.T, store *history.Store, digest string, principal governance.Principal, draftAt time.Time) {
	t.Helper()
	ctx := t.Context()
	store.SetClock(func() time.Time { return draftAt })
	football, err := store.LoadFootball(ctx, digest)
	if err != nil {
		t.Fatal(err)
	}
	store.SetPinnedFootball(football)
	order := []string{"franchise.01", "franchise.04"}
	for n := 2; n <= 32; n++ {
		if n != 4 {
			order = append(order, fmt.Sprintf("franchise.%02d", n))
		}
	}
	assets, err := store.NativeAssets(ctx, "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	reorder := commissioner.RawCommand{
		League: "native.hosted", Command: commissioner.DraftOrderSet, Expected: assets.Revision,
		Request: "box.draft.order", Effective: "now", Visibility: "league",
		Reason: "Season box: franchise.04 has a seated owner and picks second",
		Args:   commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2027, Order: order, Running: true}},
	}
	receipt, err := store.ExecuteCommissioner(ctx, reorder, string(principal), nil)
	if err != nil {
		t.Fatalf("move franchise.04 to pick two: %v", err)
	}
	if _, err := store.ExecuteCommissioner(ctx, boxLineup(receipt.Revision, "box.lineup", "0099"), string(principal), nil); err != nil {
		t.Fatalf("record franchise.01 lineup: %v", err)
	}
	if _, err := store.LinkPlayerIDs(ctx, "dynastyprocess", []history.PlayerIDLink{
		{IDType: "espn", IDValue: "4360310", PlayerID: "0099"},
		{IDType: "gsis", IDValue: "00-0000099", PlayerID: "0099"},
	}); err != nil {
		t.Fatalf("link replay quarterback ids: %v", err)
	}
}

// seedBinaryBoxAuth signs both owners in and issues the twr helper token and the read-only twr-check token.
func seedBinaryBoxAuth(
	t *testing.T, store *history.Store, credentials *accountstore.Store,
	digest string, commishRaw, ownerRaw accounts.RawCredentials, draftAt time.Time,
) binarySeed {
	t.Helper()
	ctx := t.Context()
	sessions := accounts.NewSessions(accounts.New(credentials), credentials)
	commish, err := sessions.Login(ctx, commishRaw, "season box")
	if err != nil {
		t.Fatal(err)
	}
	owner04, err := sessions.Login(ctx, ownerRaw, "season box")
	if err != nil {
		t.Fatal(err)
	}
	tokens := accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store})
	twrScope := accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01", Modes: []string{"read", "preview"},
		Commands: []string{"results.read", "standings.read", "commish.ledger.read", "native.assets.read",
			"native.directory.read", "nfl.games.read", "bracket.read", commissioner.BracketBuild, commissioner.BracketSeed, commissioner.BracketAdvance, commissioner.LineupSet},
	}
	token, err := tokens.Issue(ctx, commish.RawSession, twrScope)
	if err != nil {
		t.Fatal(err)
	}
	checkScope := accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01", Modes: []string{"read"},
		Commands: []string{"native.directory.read", "results.read", "standings.read", "nfl.games.read", "lineup.read"},
	}
	checkToken, err := tokens.Issue(ctx, commish.RawSession, checkScope)
	if err != nil {
		t.Fatal(err)
	}
	return binarySeed{
		digest: digest, token: token, checkToken: checkToken,
		commishBearer: commish.Bearer, owner04Bearer: owner04.Bearer, draftAt: draftAt,
	}
}

// boxLineup is franchise.01's week-1 lineup command.
func boxLineup(expected int64, request string, starters ...string) commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet, Request: request, Effective: "now",
		Visibility: "private", Expected: expected,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: starters}}}
}

// startBinaryBox builds twr-server (practice tag) and twr-check, seeds the DB, starts the server and connects twr.
func startBinaryBox(t *testing.T) *binaryBox {
	t.Helper()
	return startBinaryBoxWith(t, nil)
}

func startBinaryBoxWith(t *testing.T, extra binaryBoxExtra) *binaryBox {
	t.Helper()
	dir := t.TempDir()
	b := &binaryBox{
		dir: dir, clockFile: filepath.Join(dir, "clock"), liveDir: filepath.Join(dir, "live"),
		officialDir: filepath.Join(dir, "official"), certPath: filepath.Join(dir, "server.pem"),
		checkBin: filepath.Join(dir, "twr-check"), checkTokenFile: filepath.Join(dir, "check.token"),
		out: &bytes.Buffer{},
	}
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
	server, key := filepath.Join(dir, "twr-server"), filepath.Join(dir, "server.key")
	for _, args := range [][]string{
		{"go", "build", "-tags", "practice", "-o", server, "../twr-server"},
		{"go", "build", "-o", b.checkBin, "../twr-check"},
		{"openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes", "-keyout", key, "-out", b.certPath,
			"-days", "1", "-subj", "/CN=Season box", "-addext", "subjectAltName=IP:127.0.0.1"},
	} {
		if output, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec // G204: fixed test commands.
			t.Fatalf("%s: %v\n%s", args[0], err, output)
		}
	}
	if err := os.Chmod(key, 0o600); err != nil {
		t.Fatal(err)
	}
	seed := seedBinaryBoxWith(t, dir, extra)
	b.commishBearer, b.owner04Bearer = seed.commishBearer, seed.owner04Bearer
	b.setClock(t, seed.draftAt) // past draft.start (+2s); football kickoffs are at seed time + 24h
	registry, err := os.ReadFile("../../docs/ui/endpoint-registry.csv")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{
		"registry.csv": registry, "authenticator.key": bytes.Repeat([]byte{23}, 32),
		"check.token": []byte(seed.checkToken.Bearer + "\n"),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	b.launchServer(t, server, key, seed)
	return b
}

// launchServer upgrades the seeded DB, starts twr-server hosted, waits for its origin and wires twr.
func (b *binaryBox) launchServer(t *testing.T, server, key string, seed binarySeed) {
	t.Helper()
	dbPath := filepath.Join(b.dir, "server.db")
	if output, err := exec.CommandContext(t.Context(), server, "upgrade", "-db", dbPath, "-save", filepath.Join(b.dir, "pre-upgrade")).CombinedOutput(); err != nil {
		t.Fatalf("upgrade: %v\n%s", err, output)
	}
	logFile, err := os.Create(filepath.Join(b.dir, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	child := exec.CommandContext(ctx, server, "hosted", "-football", seed.digest, "-db", dbPath, "-listen", "127.0.0.1:0", //nolint:gosec // G204: fixed test binary and args.
		"-tls-cert", b.certPath, "-tls-key", key, "-totp-key-file", filepath.Join(b.dir, "authenticator.key"),
		"-preview-registry", filepath.Join(b.dir, "registry.csv"), "-official-stats")
	child.Env = append(os.Environ(), "TWR_PRACTICE_CLOCK_FILE="+b.clockFile, "TWR_PRACTICE_LIVE_DIR="+b.liveDir, "TWR_PRACTICE_OFFICIAL_DIR="+b.officialDir)
	child.Stdout, child.Stderr = logFile, logFile
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); _ = child.Wait(); _ = logFile.Close() })
	b.address = awaitOrigin(t, filepath.Join(b.dir, "server.log"))
	certificate, err := os.ReadFile(b.certPath)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certificate)
	memory := &memoryStore{tokens: map[string]string{b.address + " native.hosted": seed.token.Bearer}}
	b.env = env{configDir: t.TempDir(), roots: roots, out: b.out,
		keychain: func(origin, league string) tokenStore { return memoryKey{store: memory, key: origin + " " + league} }}
	if err := b.env.save(connection{Address: b.address, League: "native.hosted", Helper: seed.token.ID}); err != nil {
		t.Fatal(err)
	}
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
// and temp data, a fake clock it obeys, owner-door rookie draft picks, live frames replayed from
// testdata/live-week9.tar.gz by the server's practice live job, official statistics replayed from
// testdata/official by the server's own official job, and the Tuesday report card (twr-check) in read mode.
func TestSeasonInABoxBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the twr-server binary")
	}
	b := startBinaryBox(t)

	// STEP 1 league created: twr ledger over HTTPS from the binary.
	rev := b.ledgerRevision(t)
	t.Log("STEP 1 binary serves the league; twr ledger revision", rev)

	// STEP 2 quick draft: franchise.01 and franchise.04 each draft a rookie through the owner door (PickPath).
	b.quickDraft(t)

	// STEP 3 lineups: franchise.04 sets its lineup via LineupPath; twr preview admits franchise.01 before lock.
	b.setOpponentLineup(t)
	early := boxLineup(b.ledgerRevision(t), "box.lineup.early", "0099")
	body, _ := json.Marshal(early)
	if printed, err := b.twr(t, string(body), "preview", "-"); err != nil || !strings.Contains(printed, `"blockers": []`) {
		t.Fatalf("twr preview before kickoff refused: %v %s", err, printed)
	}
	t.Log("STEP 3 franchise.04 lineup set via owner door (5002); twr preview of franchise.01 admitted before kickoff")

	// STEP 4 lineup lock: the fake clock passes kickoff; the server refuses the same preview.
	b.setClock(t, b.at.Add(25*time.Hour))
	late := boxLineup(b.ledgerRevision(t), "box.lineup.late", "0099", "0101")
	body, _ = json.Marshal(late)
	if printed, err := b.twr(t, string(body), "preview", "-"); err != nil || !strings.Contains(printed, "locked at kickoff") {
		t.Fatalf("a lineup previewed after kickoff was admitted: %s", printed)
	}
	t.Log("STEP 4 lineup lock held by the binary after the fake clock passed kickoff: the preview names \"locked at kickoff\"")

	// STEP 5 live frames: recorded ESPN summaries replayed from disk; twr read sees provisional points move.
	frames := replayFrames(t)
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-01-progress.json")
	first := b.awaitProvisional(t, func(p int64) bool { return p > 0 })
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-02-progress.json")
	second := b.awaitProvisional(t, func(p int64) bool { return p > first })
	b.useLiveFrame(t, frames, "espn-401873037-raw-final.json")
	b.awaitNFLFinal(t)
	t.Logf("STEP 5 live scores move (twr read): franchise.01 %d -> %d (provisional, frames 01 -> 02 -> final)", first, second)

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

	// STEP 8 standings: 1 decided game (franchise.01 beat franchise.04); 15 other games remain pending.
	var standings commissioner.NativeStandings
	b.read(t, "standings.read", map[string]int{"season": 2026}, &standings)
	if standings.Status == "Needs a ruling" || standings.Decided != 1 {
		regressed(t, 8, fmt.Sprintf("standings unexpected: status %q (%s), %d decided, through week %d",
			standings.Status, standings.Words, standings.Decided, standings.Through))
	}
	t.Logf("STEP 8 standings.read: status %q, %d decided, %d pending, through week %d", standings.Status, standings.Decided, standings.Pending, standings.Through)

	// STEP 9 Tuesday report card: twr-check in read mode against the binary.
	summary := b.runReportCard(t)
	t.Logf("STEP 9 report card (twr-check, read mode) passed:\n%s", summary)
	notReached(t, 9, "recap: no hosted recap text read exists (blocker B2 recap)")
}
