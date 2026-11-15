package serverhttp

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

// gzipMinimum is the smallest reply worth compressing; below it the header costs more than it saves.
const gzipMinimum = 1024

// writeReading writes a league reading, gzipped when the client accepts it. Only readings use it:
// they carry league data and never a secret, so compression cannot leak one (BREACH). The fastest
// level keeps the server's CPU for the readings themselves.
func writeReading(w http.ResponseWriter, r *http.Request, encoded []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add("Vary", "Accept-Encoding")
	if len(encoded) < gzipMinimum || !acceptsGzip(r.Header.Values("Accept-Encoding")) {
		_, _ = w.Write(encoded)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	zw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return
	}
	if _, err := zw.Write(encoded); err != nil {
		return
	}
	_ = zw.Close()
}

// acceptsGzip reads Accept-Encoding (RFC 9110 §12.5.3): gzip or *, unless given q=0.
func acceptsGzip(values []string) bool {
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			coding, params, _ := strings.Cut(part, ";")
			coding = strings.ToLower(strings.TrimSpace(coding))
			if coding != "gzip" && coding != "x-gzip" && coding != "*" {
				continue
			}
			name, q, found := strings.Cut(strings.TrimSpace(params), "=")
			if !found || strings.TrimSpace(strings.ToLower(name)) != "q" {
				return true
			}
			weight, err := strconv.ParseFloat(strings.TrimSpace(q), 64)
			return err == nil && weight > 0
		}
	}
	return false
}
