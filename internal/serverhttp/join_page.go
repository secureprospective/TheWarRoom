package serverhttp

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// JoinPagePath is the browser join page. The league and code travel in the fragment, so neither is
// sent to the server in a URL or written to an access log. The page itself calls the JSON join doors.
const JoinPagePath = "/join"

//go:embed join_page.html
var joinPageHTML string

// JoinPageLink builds a same-origin link to the join page for one live code, optionally opening on
// one franchise. The code is a private secret and belongs only in the fragment.
func JoinPageLink(league, code, franchise string) string {
	values := url.Values{"league": {league}, "code": {code}}
	if franchise != "" {
		values.Set("franchise", franchise)
	}
	return JoinPagePath + "#" + values.Encode()
}

func (h Hosted) joinPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	nonce, err := pageNonce()
	if err != nil {
		http.Error(w, "page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", pageCSP(nonce))
	body := strings.ReplaceAll(joinPageHTML, pageNoncePlaceholder, nonce)
	if _, err := w.Write([]byte(body)); err != nil {
		return
	}
}

const pageNoncePlaceholder = "__CSP_NONCE__"

func pageNonce() (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("page nonce: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(nonce[:]), nil
}

// pageCSP keeps the page to its own embedded script and style and its own JSON doors: no external
// asset, frame, form navigation or injected script is allowed.
func pageCSP(nonce string) string {
	return "default-src 'none'; script-src 'nonce-" + nonce + "'; style-src 'nonce-" + nonce +
		"'; font-src data:; connect-src 'self'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"
}
