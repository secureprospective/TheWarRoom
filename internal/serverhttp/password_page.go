package serverhttp

import (
	_ "embed"
	"net/http"
	"strings"
)

// PasswordPagePath is the small browser form for changing a password with the old one.
const PasswordPagePath = "/password"

const PasswordChangePath = "/v1/auth/password/change" //nolint:gosec // A route path, not a credential.

//go:embed password_page.html
var passwordPageHTML string

func (h Hosted) passwordPage(w http.ResponseWriter, r *http.Request) {
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
	body := strings.ReplaceAll(passwordPageHTML, pageNoncePlaceholder, nonce)
	if _, err := w.Write([]byte(body)); err != nil {
		return
	}
}
