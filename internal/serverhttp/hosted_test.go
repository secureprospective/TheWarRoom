package serverhttp_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Independent RFC 6238 test code.
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type hostedFixture struct {
	pools      *db.Pools
	server     *httptest.Server
	history    *history.Store
	accounts   *accountstore.Store
	sessions   *accounts.SessionService
	enrollment *accounts.EnrollmentService
	tokens     *accounts.TokenService
	totp       *accounts.TOTPService
	principal  string
	now        time.Time
}

func newHostedFixture(t *testing.T) *hostedFixture {
	t.Helper()
	store, pools := fixtureStore(t, filepath.Join(t.TempDir(), "server.db"))
	credentials := accountstore.New(pools)
	if err := credentials.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	passwords := accounts.New(credentials)
	account, err := passwords.LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	create := commissioner.RawCommand{League: "native.hosted", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Hosted test", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Create disposable league", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissioner(t.Context(), create, string(principal), nil); err != nil {
		t.Fatal(err)
	}
	policy, err := governance.Apply(governance.State{}, governance.LeagueCreated{Creator: principal, Franchises: []governance.FranchiseID{"franchise.01", "franchise.02"}})
	if err != nil {
		t.Fatal(err)
	}
	totp, err := accounts.NewTOTP(passwords, credentials, bytes.Repeat([]byte{19}, 32))
	if err != nil {
		t.Fatal(err)
	}
	f := &hostedFixture{history: store, pools: pools, accounts: credentials, principal: string(principal), now: time.Now().UTC()}
	credentials.SetClock(func() time.Time { return f.now })
	f.sessions, f.totp = accounts.NewSessions(passwords, credentials), totp
	f.enrollment = accounts.NewEnrollment(credentials, enrollmentAuthority{policy: policy, principal: principal})
	worker := leagueservice.NewDueWorker(store, func() nativecheck.Facts { return nil }, nil)
	f.tokens = accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: store})
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		t.Fatal(err)
	}
	f.server = httptest.NewTLSServer(serverhttp.Hosted{Tokens: f.tokens, Confirmations: confirmations, Store: store, Sessions: f.sessions, TOTP: totp, Enrollment: f.enrollment, Worker: worker,
		Recovery: accounts.NewRecovery(passwords, credentials, [32]byte{}),
		Registry: "id,name,gm_question,governs\nCM-053,Note,What changed?,\n"}.Handler())
	t.Cleanup(f.server.Close)
	return f
}

func (f *hostedFixture) post(t *testing.T, path, body, bearer, proof string, status int) []byte {
	t.Helper()
	// A busy reply is transient: a loaded test machine retries it, as a desktop would.
	for attempt := 1; ; attempt++ {
		got, result := f.send(t, path, body, bearer, proof)
		if got == http.StatusServiceUnavailable && status != got && attempt < 5 &&
			strings.TrimSpace(string(result)) == serverhttp.BusyMessage {
			continue
		}
		if got != status {
			t.Fatalf("%s: status=%d want=%d body=%s", path, got, status, result)
		}
		return result
	}
}

func (f *hostedFixture) send(t *testing.T, path, body, bearer, proof string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	request.Header.Set("X-TWR-Recent-Auth", proof)
	request.Header.Set("X-Principal", f.principal) // An attacker-controlled header never selects identity.
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("%s: cacheable reply %d", path, response.StatusCode)
	}
	return response.StatusCode, result
}

// directCommissioner signs in and steps up through the account services, outside the 10-second
// HTTP password budget, for tests whose subject is not sign-in itself. The HTTPS sign-in flows
// keep their own coverage (TestHostedHTTPSEnrollmentLoginWriteAndAdmission).
func (f *hostedFixture) directCommissioner(t *testing.T) (bearer, proof string) {
	t.Helper()
	credentials := accounts.RawCredentials{Username: "commissioner", Password: "private-password"}
	session, err := f.sessions.Login(t.Context(), credentials, "test device")
	if err != nil {
		t.Fatal(err)
	}
	setup, err := f.totp.Begin(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.totp.Confirm(t.Context(), credentials, accounts.RawTOTP{Code: testTOTP(t, setup.Seed, f.now)}); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	recent, err := f.totp.StepUp(t.Context(), credentials, accounts.RawTOTP{Code: testTOTP(t, setup.Seed, f.now)})
	if err != nil {
		t.Fatal(err)
	}
	return session.Bearer, recent.Token
}

func (f *hostedFixture) login(t *testing.T, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"protocol":1,"username":%q,"password":%q,"device":"test device"}`, username, password)
	result := f.post(t, serverhttp.LoginPath, body, "", "", 200)
	var session serverhttp.LoginResponse
	if err := json.Unmarshal(result, &session); err != nil || session.Bearer == "" {
		t.Fatalf("login response: %v", err)
	}
	return session.Bearer
}
func (f *hostedFixture) revision(t *testing.T, want int) {
	t.Helper()
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil || len(events) != want {
		t.Fatalf("history events=%d want=%d: %v", len(events), want, err)
	}
}
func testTOTP(t *testing.T, seed string, at time.Time) string {
	t.Helper()
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(seed)
	if err != nil {
		t.Fatal(err)
	}
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(at.Unix()/30)) //nolint:gosec // Positive test time.
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1000000)
}
func (f *hostedFixture) privilegedProof(t *testing.T) string {
	t.Helper()
	body := `{"protocol":1,"username":"commissioner","password":"private-password"}`
	setupBody := f.post(t, serverhttp.TOTPBeginPath, body, "", "", 200)
	var setup serverhttp.AuthenticatorResponse
	if err := json.Unmarshal(setupBody, &setup); err != nil {
		t.Fatal(err)
	}
	withCode := func() string {
		return strings.TrimSuffix(body, "}") + `,"code":"` + testTOTP(t, setup.Seed, f.now) + `"}`
	}
	f.post(t, serverhttp.TOTPConfirmPath, withCode(), "", "", 200)
	f.post(t, serverhttp.TOTPStepUpPath, withCode(), "", "", 403)
	f.now = f.now.Add(30 * time.Second)
	proofBody := f.post(t, serverhttp.TOTPStepUpPath, withCode(), "", "", 200)
	var proof serverhttp.AuthenticatorResponse
	if err := json.Unmarshal(proofBody, &proof); err != nil || proof.RecentAuth == "" {
		t.Fatalf("proof: %v", err)
	}
	return proof.RecentAuth
}

func TestHostedHTTPSEnrollmentLoginWriteAndAdmission(t *testing.T) {
	f := newHostedFixture(t)
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "new.owner")
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.EnrollmentPath, `{"protocol":1,"enrollment":{"code":"`+code.Code+`","username":"new.owner","password":"owner-password"}}`, "", "", 201)
	visitor := f.login(t, "new.owner", "owner-password")
	commissionerSession := f.login(t, "commissioner", "private-password")
	read := `{"protocol":1,"league":"native.hosted"}`
	change := `{"protocol":1,"command":{"league":"native.hosted","command":"commish.whatsnew.post","args":{"note":{"text":"Hello owners"}},"reason":"Publish a league note","expected":1,"request":"https-note","effective":"now","visibility":"league"}}`
	for _, path := range []string{serverhttp.LedgerPath, serverhttp.PreviewPath, serverhttp.ExecutePath} {
		body := change
		if path == serverhttp.LedgerPath {
			body = read
		}
		f.post(t, path, body, "", "", 401)
		f.post(t, path, body, "bad-bearer", "", 401)
		status := 403
		if path == serverhttp.ExecutePath {
			status = 409
		}
		f.post(t, path, body, visitor, "", status)
		f.post(t, path, body, commissionerSession, "", status)
	}
	f.revision(t, 1)
	proof := f.privilegedProof(t)
	// A forged principal header plus the real commissioner's proof cannot upgrade a visitor.
	f.post(t, serverhttp.ExecutePath, change, visitor, proof, 409)
	f.post(t, serverhttp.ExecutePath, strings.Replace(change, `"protocol":1`, `"protocol":1,"principal":"`+f.principal+`"`, 1), commissionerSession, proof, 400)
	initialBody := f.post(t, serverhttp.LedgerPath, read, commissionerSession, proof, 200)
	var initial commissioner.Ledger
	if err := json.Unmarshal(initialBody, &initial); err != nil {
		t.Fatal(err)
	}
	change = strings.Replace(change, `"expected":1`, fmt.Sprintf(`"expected":%d`, initial.Revision), 1)
	preview := f.post(t, serverhttp.PreviewPath, change, commissionerSession, proof, 200)
	if !bytes.Contains(preview, []byte("Hello owners")) {
		t.Fatalf("missing preview: %s", preview)
	}
	f.revision(t, 1)
	receipt := f.post(t, serverhttp.ExecutePath, change, commissionerSession, proof, 200)
	retry := f.post(t, serverhttp.ExecutePath, change, commissionerSession, proof, 200)
	if !bytes.Equal(receipt, retry) {
		t.Fatal("retry changed receipt")
	}
	f.revision(t, 2)
	var recorded commissioner.Receipt
	if err := json.Unmarshal(receipt, &recorded); err != nil || recorded.Revision == initial.Revision || recorded.Revision < 2 {
		t.Fatalf("receipt: %v", err)
	}
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil || events[1].Payload.Commissioner.Principal != f.principal {
		t.Fatalf("server trusted spoofed principal: %v", err)
	}
	// Proof expiry affects reads, previews, new writes and already committed retries.
	f.now = f.now.Add(accounts.RecentAuthLifetime)
	f.post(t, serverhttp.LedgerPath, read, commissionerSession, proof, 403)
	f.post(t, serverhttp.PreviewPath, change, commissionerSession, proof, 403)
	f.post(t, serverhttp.ExecutePath, change, commissionerSession, proof, 409)
	f.revision(t, 2)
	if err := f.sessions.Revoke(t.Context(), accounts.RawSession{Bearer: commissionerSession}, false); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{serverhttp.LedgerPath, serverhttp.PreviewPath, serverhttp.ExecutePath} {
		f.post(t, path, change, commissionerSession, proof, 401)
	}
	f.post(t, serverhttp.FixtureChangePath, change, visitor, "", 404)
	f.revision(t, 2)
}

func TestHostedNilDependenciesFailClosed(t *testing.T) {
	server := httptest.NewTLSServer((serverhttp.Hosted{}).Handler())
	defer server.Close()
	for _, path := range []string{serverhttp.EnrollmentPath, serverhttp.LoginPath, serverhttp.LedgerPath, serverhttp.PreviewPath, serverhttp.ExecutePath, serverhttp.TOTPBeginPath, serverhttp.TOTPConfirmPath, serverhttp.TOTPStepUpPath} {
		response, err := server.Client().Post(server.URL+path, "application/json", strings.NewReader(`{"protocol":1}`))
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != 503 {
			t.Fatalf("nil configuration %s: %d", path, response.StatusCode)
		}
	}
}

func TestHostedHTTPSQueuedRevocation(t *testing.T) {
	f := newHostedFixture(t)
	bearer, proof := f.directCommissioner(t)
	f.now = f.now.Add(time.Nanosecond)
	digest := sha256.Sum256([]byte(bearer))
	tokenHash := hex.EncodeToString(digest[:])
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close(); _ = writer.Close() }()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.ExecutePath, reader)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("X-TWR-Recent-Auth", proof)
	result := make(chan int, 1)
	go func() {
		response, err := f.server.Client().Do(request)
		if err != nil {
			result <- 0
			return
		}
		defer func() { _ = response.Body.Close() }()
		_, _ = io.Copy(io.Discard, response.Body)
		result <- response.StatusCode
	}()
	// Hold back the body until request authentication has touched this real session.
	if _, err := io.WriteString(writer, " "); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		var used int64
		err := f.pools.Read().QueryRowContext(t.Context(), `SELECT last_used_at FROM account_sessions WHERE token_hash = ?`, tokenHash).Scan(&used)
		return err == nil && used == f.now.UnixNano()
	})
	tx, err := f.pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(t.Context(), `UPDATE account_sessions SET revoked = 1 WHERE token_hash = ?`, tokenHash); err != nil {
		t.Fatal(err)
	}
	waits := f.pools.Write().Stats().WaitCount
	body := `{"protocol":1,"command":{"league":"native.hosted","command":"commish.whatsnew.post","args":{"note":{"text":"Must not append"}},"reason":"Test queued revocation","expected":1,"request":"queued-note","effective":"now","visibility":"league"}}`
	if _, err := io.WriteString(writer, body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return f.pools.Write().Stats().WaitCount > waits })
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != 409 {
			t.Fatalf("queued revoked request status=%d", status)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("queued request did not complete")
	}
	f.revision(t, 1)
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("request did not reach the expected admission boundary")
		}
		time.Sleep(time.Millisecond)
	}
}
