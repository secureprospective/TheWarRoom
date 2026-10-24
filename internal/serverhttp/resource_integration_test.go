package serverhttp_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedLoginFloodKeepsCommandsAvailable(t *testing.T) {
	f := newHostedFixture(t)
	bearer := f.login(t, "commissioner", "private-password")
	proof := f.privilegedProof(t)
	var logs bytes.Buffer
	f.server.Config.ErrorLog = log.New(&logs, "", 0)
	start := make(chan struct{})
	var wg sync.WaitGroup
	statuses := make(chan int, 100)
	for range 100 {
		wg.Go(func() {
			<-start
			response, err := f.server.Client().Post(f.server.URL+serverhttp.LoginPath, "application/json", strings.NewReader(`{"protocol":1,"username":"commissioner","password":"sentinel-password-never-log","device":"flood"}`))
			if err != nil {
				t.Error(err)
				return
			}
			body, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				t.Error(err)
			}
			if bytes.Contains(body, []byte("sentinel-password-never-log")) {
				t.Error("password leaked in reply")
			}
			statuses <- response.StatusCode
		})
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	close(start)
	latencies := make([]time.Duration, 0, 20)
	for range 20 {
		began := time.Now()
		f.post(t, serverhttp.PreviewPath, `{"protocol":1,"command":{"league":"native.hosted","command":"commish.whatsnew.post","args":{"note":{"text":"Flood preview"}},"reason":"Exercise command during authentication flood","expected":1,"request":"flood-preview","effective":"now","visibility":"league"}}`, bearer, proof, 200)
		latencies = append(latencies, time.Since(began))
	}
	wg.Wait()
	close(statuses)
	runtime.ReadMemStats(&after)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	for status := range counts {
		if status != http.StatusUnauthorized && status != http.StatusTooManyRequests && status != http.StatusServiceUnavailable {
			t.Errorf("unexpected flood response %d", status)
		}
	}
	if counts[429] == 0 {
		t.Fatal("flood was not throttled")
	}
	slices.Sort(latencies)
	t.Logf("100 login flood: statuses=%v command p95=%s heap before=%d after=%d (local process, not target CT)", counts, latencies[18], before.HeapAlloc, after.HeapAlloc)
	f.revision(t, 1)
	if strings.Contains(logs.String(), "sentinel-password-never-log") || strings.Contains(logs.String(), bearer) || strings.Contains(logs.String(), proof) {
		t.Fatal("secret leaked in HTTP log")
	}
}

func TestHostedOversizeAndForwardingPreserveState(t *testing.T) {
	f := newHostedFixture(t)
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Proto", "CF-Connecting-IP"} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.LoginPath, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set(header, "sentinel-private-value")
		response, err := f.server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != 400 {
			t.Fatalf("%s accepted: %d", header, response.StatusCode)
		}
	}
	f.post(t, serverhttp.LoginPath, fmt.Sprintf(`{"protocol":1,"username":"commissioner","password":%q,"device":"test"}`, strings.Repeat("x", 256<<10)), "", "", 400)
	f.revision(t, 1)
}

func TestHTTPSecretFormatting(t *testing.T) {
	const sentinel = "sentinel-private-password-code-seed-bearer-proof"
	login := serverhttp.RawLogin{Protocol: 1, Username: "owner", Password: sentinel, Device: "test"}
	authenticator := serverhttp.RawAuthenticator{Protocol: 1, Username: "owner", Password: sentinel, Code: sentinel}
	response := serverhttp.LoginResponse{Protocol: 1, Bearer: sentinel}
	proof := serverhttp.AuthenticatorResponse{Protocol: 1, Seed: sentinel, RecentAuth: sentinel}
	for _, formatted := range []string{
		fmt.Sprintf("%v %+v %#v", login, login, login),
		fmt.Sprintf("%v %+v %#v", authenticator, authenticator, authenticator),
		fmt.Sprintf("%v %+v %#v", response, response, response),
		fmt.Sprintf("%v %+v %#v", proof, proof, proof),
	} {
		if strings.Contains(formatted, sentinel) {
			t.Fatal("HTTP credential formatting leaked secret")
		}
	}
}
