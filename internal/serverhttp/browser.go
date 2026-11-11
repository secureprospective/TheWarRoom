package serverhttp

import (
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp/landing"
)

// browserRoute names the pages served to a browser and the JSON doors those pages call. Every other
// route keeps refusing any request that carries an Origin header.
func browserRoute(path string) bool {
	switch path {
	case JoinPagePath, JoinFranchisesPath, JoinClaimPath, PasswordPagePath, PasswordChangePath:
		return true
	}
	return landing.Has(path)
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
