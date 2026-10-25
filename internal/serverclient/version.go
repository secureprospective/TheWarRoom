package serverclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

// ErrUpdateRequired matches every UpdateRequired, for callers that only need to know.
var ErrUpdateRequired = errors.New("update required")

// UpdateRequired is the server's refusal of this build's protocol or command version. Update
// is serverversion.App when this app must update and serverversion.Server when the league's
// server must; Message is the sentence to show the person. It is the signal the updater keys on.
type UpdateRequired struct {
	Update  serverversion.Side
	Message string
}

func (u *UpdateRequired) Error() string        { return u.Message }
func (u *UpdateRequired) Is(target error) bool { return target == ErrUpdateRequired }

// versionedTransport stamps every request with this build's versions and turns the server's 426
// refusal into an UpdateRequired error, whichever call or stream made the request.
type versionedTransport struct{ next http.RoundTripper }

func (t versionedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set(serverhttp.ProtocolHeader, serverversion.ClientProtocol())
	request.Header.Set(serverhttp.CommandsHeader, serverversion.ClientCommands())
	response, err := t.next.RoundTrip(request)
	if err != nil || response.StatusCode != http.StatusUpgradeRequired {
		return response, err //nolint:wrapcheck // http.Client wraps transport errors with the request URL.
	}
	defer func() { _ = response.Body.Close() }()
	return nil, readUpdateRequired(response)
}

func readUpdateRequired(response *http.Response) error {
	unreadable := fmt.Errorf("the league server refused this version of The War Room but its reason was unreadable; install the latest update")
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<10+1))
	if err != nil || len(body) > 4<<10 || response.Header.Get("Content-Type") != "application/json" {
		return unreadable
	}
	var raw serverhttp.RawUpdateRequiredResponse
	if err := json.Unmarshal(body, &raw); err != nil || raw.Validate() != nil || raw.Update != response.Header.Get(serverhttp.UpdateHeader) {
		return unreadable
	}
	return &UpdateRequired{Update: serverversion.Side(raw.Update), Message: raw.Message}
}

// updateRequired returns the UpdateRequired inside a failed request's error, if there is one.
func updateRequired(err error) (*UpdateRequired, bool) {
	var update *UpdateRequired
	ok := errors.As(err, &update)
	return update, ok
}

// newHTTPClient is the bounded, redirect-refusing, version-declaring client every league connection uses.
func newHTTPClient(roots *x509.CertPool, timeout time.Duration, redirect string) *http.Client {
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	return &http.Client{Timeout: timeout, Transport: versionedTransport{next: transport},
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New(redirect) }}
}
