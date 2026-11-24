package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha1" //nolint:gosec // Independent RFC 6238 test TOTP code.
	"crypto/x509"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

type canaryEnrollmentPolicy struct {
	state     governance.State
	principal governance.Principal
}

func (p canaryEnrollmentPolicy) EnrollmentPolicy(_ context.Context, _ string) (governance.State, governance.Principal, error) {
	return p.state, p.principal, nil
}

type canaryClock struct{}

func (canaryClock) Now() time.Time { return time.Now().UTC() }
func (canaryClock) Trusted() error { return nil }

type canaryBox struct {
	store         *history.Store
	totp          *accounts.TOTPService
	now           *atomic.Int64
	commishRaw    accounts.RawCredentials
	commishBearer string
	token         accounts.AgentToken
	server        *httptest.Server
	url           string
	roots         *x509.CertPool
	tokenFile     string
	allowFile     string
}

func startCanaryBox(t *testing.T, wrap func(http.Handler) http.Handler) *canaryBox {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	pools, err := db.Open(ctx, filepath.Join(dir, "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	now := &atomic.Int64{}
	now.Store(time.Now().UTC().UnixNano())
	credentials := accountstore.New(pools)
	credentials.SetClock(func() time.Time { return time.Unix(0, now.Load()).UTC() })
	commishRaw := accounts.RawCredentials{Username: "commissioner", Password: cryptorand.Text()}
	principal, err := hostedfixture.Bootstrap(ctx, credentials, commishRaw)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := hostedfixture.CreateWorkflowLeague(ctx, store, principal)
	if err != nil {
		t.Fatal(err)
	}
	ownerRaw := accounts.RawCredentials{Username: "canary-owner", Password: cryptorand.Text()}
	if err := hostedfixture.AddWorkflowOwner(ctx, store, credentials, principal, "franchise.04", ownerRaw); err != nil {
		t.Fatalf("seat owner: %v", err)
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
	lineupCmd := commissioner.RawCommand{
		League: "native.hosted", Command: commissioner.LineupSet, Request: "canary.seed.lineup",
		Effective: "now", Visibility: "private", Expected: assets.Revision,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"},
		}},
	}
	if _, err := store.ExecuteCommissioner(ctx, lineupCmd, string(principal), nil); err != nil {
		t.Fatalf("seed week-1 lineup: %v", err)
	}
	commishBearer, token, totp, hosted := buildCanaryHosted(t, pools, store, credentials, commishRaw, principal, digest, football)
	httpHandler := hosted.Handler()
	if wrap != nil {
		httpHandler = wrap(httpHandler)
	}
	srv := httptest.NewTLSServer(httpHandler)
	t.Cleanup(srv.Close)
	tokenFile := filepath.Join(dir, "canary.token")
	allowFile := filepath.Join(dir, "allow.txt")
	if err := os.WriteFile(tokenFile, []byte(token.Bearer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(allowFile, []byte("native.hosted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return &canaryBox{
		store:         store,
		totp:          totp,
		now:           now,
		commishRaw:    commishRaw,
		commishBearer: commishBearer,
		token:         token,
		server:        srv,
		url:           srv.URL,
		roots:         srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs,
		tokenFile:     tokenFile,
		allowFile:     allowFile,
	}
}

func buildCanaryHosted(
	t *testing.T, pools *db.Pools, store *history.Store, credentials *accountstore.Store,
	commishRaw accounts.RawCredentials, principal governance.Principal, digest string, football *history.PinnedFootball,
) (string, accounts.AgentToken, *accounts.TOTPService, serverhttp.Hosted) {
	t.Helper()
	ctx := t.Context()
	passwords := accounts.New(credentials)
	sessions := accounts.NewSessions(passwords, credentials)
	commish, err := sessions.Login(ctx, commishRaw, "canary test")
	if err != nil {
		t.Fatal(err)
	}
	totp, err := accounts.NewTOTP(passwords, credentials, bytes.Repeat([]byte{19}, 32))
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.Events(ctx, "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records := make([]commissioner.Record, len(events))
	for i, event := range events {
		records[i] = *event.Payload.Commissioner
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	tokens := accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store})
	scope := accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01",
		Modes: []string{"read", "preview", "execute"},
		Commands: []string{
			"native.directory.read", "native.assets.read", "commish.ledger.read",
			"week.read", "lineup.read", "nfl.games.read", "waivers.read",
			"results.read", "standings.read", "bracket.read", "league.activity.read",
			"commish.rules.read", "draft.read",
			commissioner.LineupSet, commissioner.WaiverClaim,
		},
	}
	token, err := tokens.Issue(ctx, commish.RawSession, scope)
	if err != nil {
		t.Fatalf("issue canary token: %v", err)
	}
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		t.Fatal(err)
	}
	pinned := serverhttp.LeaguePin{Store: store, Personal: personal.NewLeague(pools), Football: football, Digest: digest}
	hosted := serverhttp.Hosted{
		Store: store, Leagues: history.NewOneFile(store), Sessions: sessions, Tokens: tokens, TOTP: totp,
		Enrollment: accounts.NewEnrollment(credentials, canaryEnrollmentPolicy{state, principal}),
		Worker:     leagueservice.NewDueWorker(history.NewOneFile(store), football.PlayerFacts, canaryClock{}),
		LiveGames:  leagueservice.NewLiveGames(), Confirmations: confirmations,
		Football: football, FootballDigest: digest,
		Pins:     func(context.Context, string) serverhttp.LeaguePin { return pinned },
		Registry: "id,name,gm_question,governs\n",
	}
	return commish.Bearer, token, totp, hosted
}

func testTOTPCode(t *testing.T, seed string, at time.Time) string {
	t.Helper()
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(seed)
	if err != nil {
		t.Fatal(err)
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(at.Unix()/30)) //nolint:gosec // Positive Unix timestamp in test.
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(counter[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1_000_000
	return fmt.Sprintf("%06d", code)
}

func (b *canaryBox) confirmCommand(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	ctx := t.Context()
	setup, err := b.totp.Begin(ctx, b.commishRaw)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(0, b.now.Load()).UTC()
	if err := b.totp.Confirm(ctx, b.commishRaw, accounts.RawTOTP{Code: testTOTPCode(t, setup.Seed, at)}); err != nil {
		t.Fatal(err)
	}
	b.now.Add(int64(30 * time.Second))
	at = time.Unix(0, b.now.Load()).UTC()
	recent, err := b.totp.StepUp(ctx, b.commishRaw, accounts.RawTOTP{Code: testTOTPCode(t, setup.Seed, at)})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, TokenID: b.token.ID, Command: &raw})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.url+serverhttp.ConfirmPath, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+b.commishBearer)
	req.Header.Set("X-TWR-Recent-Auth", recent.Token)
	resp, err := b.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("confirm status=%d err=%v body=%s", resp.StatusCode, err, respBody)
	}
	var out serverhttp.ConfirmationResponse
	if err := json.Unmarshal(respBody, &out); err != nil || out.Confirmation == "" {
		t.Fatalf("decode confirmation: %v %q", err, respBody)
	}
	return out.Confirmation
}

func (b *canaryBox) args(seed, outPath string, extra ...string) []string {
	args := []string{
		"-target", "lab",
		"-server", b.url,
		"-token-file", b.tokenFile,
		"-league", "native.hosted",
		"-franchise", "franchise.01",
		"-season", "2026",
		"-week", "1",
		"-allow-file", b.allowFile,
		"-seed", seed,
	}
	if outPath != "" {
		args = append(args, "-out", outPath)
	}
	return append(args, extra...)
}

func journeySignature(output string) []string {
	var sig []string
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if strings.HasPrefix(line, "seed=") {
			sig = append(sig, line)
			continue
		}
		if idx := strings.Index(line, "]"); idx > 0 {
			sig = append(sig, line[:idx+1])
		}
	}
	return sig
}

func TestHostedCanaryEndToEndAndReplay(t *testing.T) {
	const seed = "202648-9f3a"
	box := startCanaryBox(t, nil)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	var out1, err1 bytes.Buffer
	if code := run(box.args(seed, reportPath), &out1, &err1, box.roots); code != 0 {
		t.Fatalf("run 1 exit=%d stderr=%q\nstdout:\n%s", code, err1.String(), out1.String())
	}
	t.Logf("green canary run:\n%s", out1.String())
	if strings.Contains(out1.String(), "\nFAIL ") {
		t.Fatalf("unexpected FAIL line in green run:\n%s", out1.String())
	}
	for _, mustPass := range []string{
		"PASS connect [",
		"PASS week and lineup [",
		"PASS set lineup [",
		"PASS live score [",
		"PASS waiver claim [",
		"PASS standings [",
		"PASS rule read-back [",
	} {
		if !strings.Contains(out1.String(), mustPass) {
			t.Fatalf("expected %q in output:\n%s", mustPass, out1.String())
		}
	}
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil || !strings.Contains(string(reportBytes), "seed="+seed) {
		t.Fatalf("report file missing seed: err=%v body=%q", err, string(reportBytes))
	}

	var out2, err2 bytes.Buffer
	if code := run(box.args(seed, ""), &out2, &err2, box.roots); code != 0 {
		t.Fatalf("run 2 exit=%d stderr=%q\nstdout:\n%s", code, err2.String(), out2.String())
	}
	sig1, sig2 := journeySignature(out1.String()), journeySignature(out2.String())
	if !slices.Equal(sig1, sig2) {
		t.Fatalf("same seed replayed different subset or order:\nrun1=%v\nrun2=%v", sig1, sig2)
	}
}

func TestHostedCanaryLineupExecuteWithConfirmation(t *testing.T) {
	const seed = "202648-9f3a"
	box := startCanaryBox(t, nil)
	before, err := box.store.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := serverclient.NewAgent(box.url, "native.hosted", box.token.Bearer, box.roots)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := agent.Ledger(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{
		League: "native.hosted", Command: commissioner.LineupSet, Request: "canary." + seed + ".lineup",
		Effective: "now", Visibility: "private", Expected: ledger.Revision,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"},
		}},
	}
	code := box.confirmCommand(t, raw)
	var out, stderr bytes.Buffer
	if exit := run(box.args(seed, "", "-confirmation", code), &out, &stderr, box.roots); exit != 0 {
		t.Fatalf("confirmed execute exit=%d stderr=%q\nstdout:\n%s", exit, stderr.String(), out.String())
	}
	if !strings.Contains(out.String(), "PASS set lineup [lineup.read,commish.lineup.set]") ||
		!strings.Contains(out.String(), "previewed and recorded lineup") {
		t.Fatalf("expected recorded lineup in output:\n%s", out.String())
	}
	after, err := box.store.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("ledger events before=%d after=%d, want +1 from lineup execute (waiver must stay preview-only)", len(before), len(after))
	}
}

func TestHostedCanaryPlantedStandingsRed(t *testing.T) {
	var mutatedCount atomic.Int64
	wrap := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != serverhttp.WorkflowPath {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var req serverhttp.RawNativeRequest
			if json.Unmarshal(body, &req) != nil || req.Workflow == nil || req.Workflow.Command != "standings.read" {
				next.ServeHTTP(w, r)
				return
			}
			r.Header.Del("Accept-Encoding")
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			var resp serverhttp.WorkflowResponse
			if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &resp) != nil {
				t.Errorf("inner standings.read failed: status=%d body=%q", rec.Code, rec.Body.String())
				http.Error(w, "inner standings failed", http.StatusInternalServerError)
				return
			}
			var st commissioner.NativeStandings
			if err := json.Unmarshal(resp.Data, &st); err != nil || len(st.Computed) == 0 {
				t.Errorf("unexpected standings payload: %v", err)
				return
			}
			st.Computed[0].Wins += 3
			mutated, err := json.Marshal(st)
			if err != nil {
				t.Errorf("marshal mutated standings: %v", err)
				return
			}
			resp.Data = mutated
			mutatedCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		})
	}
	box := startCanaryBox(t, wrap)
	var out, stderr bytes.Buffer
	code := run(box.args("202648-9f3a", ""), &out, &stderr, box.roots)
	if mutatedCount.Load() == 0 {
		t.Fatalf("standings.read wrapper never mutated a response")
	}
	if code != 1 {
		t.Fatalf("planted standings corruption exit=%d, want 1\nstdout:\n%s\nstderr:\n%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "FAIL standings [standings.read]") {
		t.Fatalf("expected journey 13 FAIL in output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "FAIL set lineup") {
		t.Fatalf("only standings should fail in planted red test:\n%s", out.String())
	}
}

func TestHostedCanaryWaiverClaimPreview5xxFails(t *testing.T) {
	wrap := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == serverhttp.WorkflowPath {
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(body))
				var req serverhttp.RawNativeRequest
				if json.Unmarshal(body, &req) == nil && req.Workflow != nil && req.Workflow.Command == commissioner.WaiverClaim {
					http.Error(w, "injected server error", http.StatusInternalServerError)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
	box := startCanaryBox(t, wrap)
	var out, stderr bytes.Buffer
	code := run(box.args("202648-9f3a", ""), &out, &stderr, box.roots)
	if code != 1 || !strings.Contains(out.String(), "FAIL waiver claim [") {
		t.Fatalf("expected 5xx waiver preview to fail: code=%d\nstdout:\n%s\nstderr:\n%s", code, out.String(), stderr.String())
	}
}

func TestHostedCanaryMFLRefusalZeroRequests(t *testing.T) {
	var hits atomic.Int64
	wrap := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			next.ServeHTTP(w, r)
		})
	}
	box := startCanaryBox(t, wrap)
	allow := filepath.Join(t.TempDir(), "allow.txt")
	if err := os.WriteFile(allow, []byte("native.hosted\nmfl.14432\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := run([]string{
		"-target", "lab",
		"-server", box.url,
		"-token-file", box.tokenFile,
		"-league", "mfl.14432",
		"-franchise", "franchise.01",
		"-season", "2026",
		"-week", "1",
		"-allow-file", allow,
	}, &out, &stderr, box.roots)
	if code != 2 {
		t.Fatalf("MFL refusal exit=%d, want 2 (stderr=%q)", code, stderr.String())
	}
	if got := hits.Load(); got != 0 {
		t.Fatalf("server saw %d requests for mfl.14432, want 0", got)
	}
}
