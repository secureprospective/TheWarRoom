package serverhttp

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
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

func (t *authThrottle) allow(client netip.Addr, now time.Time) bool {
	if !client.IsValid() {
		return false
	}
	host := client.Unmap().String()
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

// clientAddress is the address a request is throttled as: its socket peer, or, only when that peer
// is the configured loopback tunnel hop (cloudflared), the one CF-Connecting-IP the hop sets. Every
// other forwarding header is refused from anyone else, so a client cannot choose its own bucket.
func clientAddress(r *http.Request, hop netip.Addr) (netip.Addr, bool) {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}, false
	}
	viaHop := hop.IsValid() && hop.IsLoopback() && peer.Addr().Unmap() == hop.Unmap()
	for key := range r.Header {
		key = strings.ToLower(key)
		if !viaHop && (key == "forwarded" || strings.HasPrefix(key, "x-forwarded-") || key == "x-real-ip" ||
			key == "cf-connecting-ip" || key == "true-client-ip") {
			return netip.Addr{}, false
		}
	}
	if !viaHop {
		return peer.Addr(), true
	}
	// The hop always names the client; a request without exactly one address is not from cloudflared.
	values := r.Header.Values("CF-Connecting-IP")
	if len(values) != 1 {
		return netip.Addr{}, false
	}
	client, err := netip.ParseAddr(values[0])
	if err != nil || client.Zone() != "" {
		return netip.Addr{}, false
	}
	return client, true
}

// resourceLimits bounds every hosted request. hop is the loopback cloudflared address whose
// CF-Connecting-IP is trusted; the zero address trusts no proxy.
// BusyMessage is what a person reads when the server, not their password or code, is the problem.
const BusyMessage = "The server is busy; try again in a few seconds"

const busyRetryAfter = "2"

func writeUnavailable(w http.ResponseWriter, message string) {
	w.Header().Set("Retry-After", busyRetryAfter)
	http.Error(w, message, http.StatusServiceUnavailable)
}

type busyLogState struct {
	started time.Time
	logger  *busyLogger
}

type busyLogKey struct{}

type busyLogger struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func (l *busyLogger) refusal(class string, elapsed time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if last := l.last[class]; now.Sub(last) < time.Minute {
		return
	}
	l.last[class] = now
	log.Printf("hosted refusal class=%s elapsed_ms=%d", class, elapsed.Milliseconds())
}

func logBusyRefusal(ctx context.Context, class string) {
	state, ok := ctx.Value(busyLogKey{}).(busyLogState)
	if ok && state.logger != nil {
		state.logger.refusal(class, time.Since(state.started))
	}
}

// passwordBusy answers a full password budget, or password work that ran out of request time while
// queued, as the server being busy. Reporting it as a wrong password or phone clock sends people
// hunting for a mistake they did not make.
func passwordBusy(w http.ResponseWriter, r *http.Request, err error) bool {
	if !errors.Is(err, accounts.ErrPasswordBusy) && !errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	logBusyRefusal(r.Context(), "password-limit")
	writeUnavailable(w, BusyMessage)
	return true
}

func resourceLimits(next http.Handler, hop netip.Addr) http.Handler {
	throttle := &authThrottle{peers: make(map[string]authWindow)}
	inFlight := make(chan struct{}, 64)
	authInFlight := make(chan struct{}, 6)
	logger := &busyLogger{last: make(map[string]time.Time)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestContext := context.WithValue(r.Context(), busyLogKey{}, busyLogState{started: started, logger: logger})
		r = r.WithContext(requestContext)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !originAdmissible(r) {
			http.Error(w, "browser requests refused", http.StatusBadRequest)
			return
		}
		// Clients never select their throttle identity or assert that an insecure hop was HTTPS.
		client, ok := clientAddress(r, hop)
		if !ok {
			http.Error(w, "forwarding headers refused", http.StatusBadRequest)
			return
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
				logBusyRefusal(requestContext, "auth-limit")
				writeUnavailable(w, BusyMessage)
				return
			}
		}
		select {
		case inFlight <- struct{}{}:
			defer func() { <-inFlight }()
		default:
			logBusyRefusal(requestContext, "request-limit")
			writeUnavailable(w, BusyMessage)
			return
		}
		budget := 5 * time.Second
		if strings.HasPrefix(r.URL.Path, "/v1/auth/") {
			budget = 10 * time.Second
			if !throttle.allow(client, time.Now()) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "too many authentication attempts", http.StatusTooManyRequests)
				return
			}
		}
		ctx, cancel := context.WithTimeout(requestContext, budget)
		defer cancel()
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
