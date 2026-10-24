package serverhttp

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// authThrottle stores only socket peer IPs, never usernames, headers or credentials.
// When full it refuses new peers rather than evicting an attacker's spent allowance.
type authThrottle struct {
	mu    sync.Mutex
	peers map[string]authWindow
}

type authWindow struct {
	start    time.Time
	attempts int
}

func (t *authThrottle) allow(peer string, now time.Time) bool {
	host, _, err := net.SplitHostPort(peer)
	if err != nil || net.ParseIP(host) == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, exists := t.peers[host]
	if !exists && len(t.peers) >= 4096 {
		for key, value := range t.peers {
			if now.Sub(value.start) >= time.Minute {
				delete(t.peers, key)
			}
		}
		if len(t.peers) >= 4096 {
			return false
		}
	}
	if now.Sub(entry.start) >= time.Minute {
		entry = authWindow{start: now}
	}
	if entry.attempts >= 10 {
		return false
	}
	entry.attempts++
	t.peers[host] = entry
	return true
}

func resourceLimits(next http.Handler) http.Handler {
	throttle := &authThrottle{peers: make(map[string]authWindow)}
	inFlight := make(chan struct{}, 64)
	authInFlight := make(chan struct{}, 6)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if len(r.Header.Values("Origin")) != 0 {
			http.Error(w, "browser requests refused", http.StatusBadRequest)
			return
		}
		// No proxy is trusted yet. Deployment must strip these headers, not let clients
		// select their throttle identity or assert that an insecure hop was HTTPS.
		for key := range r.Header {
			key = strings.ToLower(key)
			if key == "forwarded" || strings.HasPrefix(key, "x-forwarded-") || key == "x-real-ip" || key == "cf-connecting-ip" || key == "true-client-ip" {
				http.Error(w, "forwarding headers refused", http.StatusBadRequest)
				return
			}
		}
		// Streams own their connection budget and per-write deadlines.
		if r.URL.Path == InvalidationPath {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/auth/") {
			select {
			case authInFlight <- struct{}{}:
				defer func() { <-authInFlight }()
			default:
				w.Header().Set("Retry-After", "1")
				http.Error(w, "authentication service busy", http.StatusTooManyRequests)
				return
			}
		}
		select {
		case inFlight <- struct{}{}:
			defer func() { <-inFlight }()
		default:
			http.Error(w, "service busy", http.StatusServiceUnavailable)
			return
		}
		budget := 5 * time.Second
		if strings.HasPrefix(r.URL.Path, "/v1/auth/") {
			budget = 10 * time.Second
			if !throttle.allow(r.RemoteAddr, time.Now()) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "too many authentication attempts", http.StatusTooManyRequests)
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
