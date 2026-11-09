package serverhttp_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

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
