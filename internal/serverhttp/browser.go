package serverhttp

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp/landing"
)

const (
	AppPath        = "/app/"
	AppSessionPath = "/app/session"
)

const appSessionCookie = "__Host-twr_session"

// browserRoute names the existing public pages and their unauthenticated JSON doors.
func browserRoute(path string) bool {
	switch path {
	case JoinPagePath, JoinFranchisesPath, JoinClaimPath, PasswordPagePath, PasswordChangePath:
		return true
	}
	return landing.Has(path)
}

func appRoute(path string) bool { return path == "/app" || strings.HasPrefix(path, AppPath) }

// browserJSONRoute lists command routes that may use the owner-session cookie. The changes stream,
// account administration and enrollment remain outside this browser exception.
func browserJSONRoute(path string) bool {
	switch path {
	case PeoplePath, DispatchPath, LedgerPath, PreviewPath, ExecutePath, LineupPath, PickPath, ConfirmPath,
		WorkflowPath, AccountPreferencePath:
		return true
	default:
		return false
	}
}

// validPublicHost accepts only a bare configured DNS hostname, never an origin, wildcard, port or IP.
func ValidatePublicHost(host string) error {
	if !validPublicHost(host) {
		return fmt.Errorf("public host must be a bare lowercase DNS hostname")
	}
	return nil
}

func validPublicHost(host string) bool {
	if host == "" || len(host) > 253 || net.ParseIP(host) != nil || strings.ToLower(host) != host || strings.HasSuffix(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if !validPublicHostLabel(label) {
			return false
		}
	}
	return true
}

func validPublicHostLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, c := range label {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

func (h Hosted) ownBrowserOrigin(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	return r.TLS != nil && validPublicHost(h.PublicHost) && len(origins) == 1 &&
		origins[0] == "https://"+h.PublicHost && r.Host == h.PublicHost
}

func (h Hosted) originAdmissible(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if browserRoute(r.URL.Path) && sameOrigin(r) {
		return true
	}
	return h.ownBrowserOrigin(r) && (appRoute(r.URL.Path) || browserJSONRoute(r.URL.Path))
}

func (h Hosted) sessionOriginRequired(r *http.Request) bool {
	return r.URL.Path == AppSessionPath || (browserJSONRoute(r.URL.Path) && hasBrowserSessionCookie(r))
}

// sameOrigin reports an absent Origin, or exactly one Origin equal to this server's own scheme and
// host. A browser sends Origin on every POST; a cross-site page cannot forge it.
func sameOrigin(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 || origins[0] == "" || origins[0] == "null" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return origins[0] == scheme+"://"+r.Host
}

// originAdmissible is the one browser exception: a same-origin request to a page route or its JSON
// door. Everything else with an Origin is refused before routing.
func originAdmissible(r *http.Request) bool {
	if len(r.Header.Values("Origin")) == 0 {
		return true
	}
	return browserRoute(r.URL.Path) && sameOrigin(r)
}
