// Package landing serves the public landing page and its own assets from an embedded, fixed allowlist.
package landing

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strconv"
)

//go:embed site
var site embed.FS

// CSP keeps the page to its own script, style, fonts and images; the only inline allowance is
// style attributes, which the page's script sets.
const CSP = "default-src 'none'; script-src 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"

type asset struct {
	file        string
	contentType string
	cache       string
}

// find maps a URL path to its embedded file. Nothing outside this fixed list is ever read over HTTP.
func find(route string) (asset, bool) {
	const (
		code   = "public, max-age=300"
		stable = "public, max-age=604800"
	)
	switch route {
	case "/":
		return asset{"site/index.html", "text/html; charset=utf-8", "no-store"}, true
	case "/site.css":
		return asset{"site/site.css", "text/css; charset=utf-8", code}, true
	case "/site.js", "/steel.js":
		return asset{"site" + route, "text/javascript; charset=utf-8", code}, true
	case "/og.png":
		return asset{"site/og.png", "image/png", stable}, true
	}
	if path.Dir(route) == "/fonts" && path.Ext(route) == ".woff2" {
		if _, err := fs.Stat(site, "site"+route); err == nil {
			return asset{"site" + route, "font/woff2", stable}, true
		}
	}
	return asset{}, false
}

// Has reports whether route is one of the landing page's URLs.
func Has(route string) bool {
	_, ok := find(route)
	return ok
}

// Serve answers one landing URL with GET or HEAD. The query string is ignored: shared links carry
// tracking parameters.
func Serve(w http.ResponseWriter, r *http.Request) {
	file, ok := find(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	body, err := site.ReadFile(file.file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	header := w.Header()
	header.Set("Content-Security-Policy", CSP)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		header.Set("Allow", "GET, HEAD")
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	header.Set("Content-Type", file.contentType)
	header.Set("Cache-Control", file.cache)
	header.Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}
