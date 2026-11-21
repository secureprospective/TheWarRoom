package serverhttp

import (
	"embed"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// webassets is populated from frontend/dist-web, or frontend/dist, before the server release build.
//
//go:embed all:webassets
var webassets embed.FS

const browserAppCSP = "default-src 'self'; frame-ancestors 'none'"

func webAppHandler() http.Handler {
	assets, err := fs.Sub(webassets, "webassets")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "browser app unavailable", http.StatusServiceUnavailable)
		})
	}
	return webAssetsHandler(assets)
}

func webAssetsHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", browserAppCSP)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/app" {
			http.Redirect(w, r, AppPath, http.StatusPermanentRedirect)
			return
		}
		if !strings.HasPrefix(r.URL.Path, AppPath) {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, AppPath)
		if name == "" {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			http.NotFound(w, r)
			return
		}
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			if path.Ext(name) != "" {
				// A stale phone copy asks for files an older build named; the path says which build.
				slog.Warn("browser app: no file", slog.String("file", logSafePath(name)))
				if strings.HasPrefix(name, "assets/") {
					// Only a page from an older release asks for a build file that is gone. Drop that
					// device's app cache and worker so its next open loads the current release; the
					// sign-in cookie is not touched. (An S25 stayed on an old page after many releases.)
					w.Header().Set("Clear-Site-Data", `"cache", "storage"`)
				}
				http.NotFound(w, r)
				return
			}
			name = "index.html"
			body, err = fs.ReadFile(assets, name)
			if err != nil {
				http.Error(w, "browser app unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		if name == "index.html" {
			body = browserBasePaths(body)
		}
		contentType := webAppContentType(name)
		w.Header().Set("Content-Type", contentType)
		if name == "sw.js" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", stringLength(body))
			return
		}
		//nolint:gosec // Embedded build assets are trusted input; request paths select only their names.
		_, _ = w.Write(body)
	})
}

func browserBasePaths(index []byte) []byte {
	body := string(index)
	for _, quote := range []string{`"`, `'`} {
		body = strings.ReplaceAll(body, quote+"/assets/", quote+AppPath+"assets/")
		body = strings.ReplaceAll(body, quote+"/manifest.webmanifest", quote+AppPath+"manifest.webmanifest")
		body = strings.ReplaceAll(body, quote+"/sw.js", quote+AppPath+"sw.js")
	}
	return []byte(body)
}

func webAppContentType(name string) string {
	switch path.Ext(name) {
	case ".webmanifest":
		return "application/manifest+json"
	case ".js":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	}
	if value := mime.TypeByExtension(path.Ext(name)); value != "" {
		return value
	}
	return "application/octet-stream"
}

func stringLength(body []byte) string {
	return strconv.Itoa(len(body))
}

// logSafePath keeps only the characters an app file name uses, so a request cannot forge log lines.
func logSafePath(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r == '/' || r == '.' || r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}
		return '?'
	}, name)
	if len(clean) > 120 {
		clean = clean[:120]
	}
	return clean
}
