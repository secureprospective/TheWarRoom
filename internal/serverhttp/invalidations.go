package serverhttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const InvalidationPath = "/v1/native/invalidations"

// Each Handler owns its connection budget. No event history, ledger cursor or bearer
// is published. A slow consumer retains only its last observed recipient view.
func (h Hosted) invalidationHandler(next http.Handler) http.Handler {
	var mu sync.Mutex
	devices := make(map[string]int)
	active := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != InvalidationPath {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		league := r.Header.Get("X-TWR-League")
		if !validInvalidationRequest(r) {
			http.Error(w, "invalid invalidation request", http.StatusBadRequest)
			return
		}
		if h.Store == nil || h.Sessions == nil || h.TOTP == nil || h.Worker == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		service, device := h.invalidationService(w, r)
		if service == nil {
			return
		}
		//nolint:contextcheck // StoredHost owns the request context used by the reading API.
		reading, err := service.TargetCommissionerLedger(league)
		if err != nil {
			http.Error(w, "native read refused", http.StatusForbidden)
			return
		}
		mu.Lock()
		allowed := active < 64 && devices[device] < 2
		if allowed {
			active++
			devices[device]++
		}
		mu.Unlock()
		if !allowed {
			http.Error(w, "too many streams", http.StatusTooManyRequests)
			return
		}
		defer func() {
			mu.Lock()
			active--
			devices[device]--
			if devices[device] == 0 {
				delete(devices, device)
			}
			mu.Unlock()
		}()
		w.Header().Set("Content-Type", "text/event-stream")
		controller := http.NewResponseController(w)
		if emitInvalidation(controller, w) != nil {
			return
		}
		streamInvalidations(r.Context(), controller, w, service, league, reading.Revision)
	})
}

func streamInvalidations(ctx context.Context, controller *http.ResponseController, w http.ResponseWriter, service *leagueservice.Service, league string, revision int64) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Reads recheck session, current grants and TOTP. Only committed, recipient-visible
			// state affects the token; hidden inserts and failed transactions cannot wake a peer.
			//nolint:contextcheck // The service's StoredHost carries ctx through the read.
			reading, err := service.TargetCommissionerLedger(league)
			if err != nil {
				return
			}
			if reading.Revision != revision {
				if emitInvalidation(controller, w) != nil {
					return
				}
				revision = reading.Revision
			}
		case <-heartbeat.C:
			if err := controller.SetWriteDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
				return
			}
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			if err := controller.Flush(); err != nil {
				return
			}
		}
	}
}

func emitInvalidation(controller *http.ResponseController, w http.ResponseWriter) error {
	if err := controller.SetWriteDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return fmt.Errorf("stream deadline: %w", err)
	}
	if _, err := fmt.Fprint(w, "event: invalidate\ndata: {\"protocol\":1}\n\n"); err != nil {
		return fmt.Errorf("stream invalidation: %w", err)
	}
	if err := controller.Flush(); err != nil {
		return fmt.Errorf("flush invalidation: %w", err)
	}
	return nil
}

func (h Hosted) invalidationService(w http.ResponseWriter, r *http.Request) (*leagueservice.Service, string) {
	authorization := r.Header.Get("Authorization")
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(authorization, "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return nil, ""
	}
	session := accounts.RawSession{Bearer: strings.TrimPrefix(authorization, "Bearer ")}
	account, err := h.Sessions.Authenticate(r.Context(), session)
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return nil, ""
	}
	principal, err := account.Principal()
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return nil, ""
	}
	host := leagueservice.StoredHost{Store: h.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(session), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	service := leagueservice.NewHosted(h.Worker.Host(host), gate)
	return service, session.Bearer
}

func validInvalidationRequest(r *http.Request) bool {
	raw := RawNativeRequest{Protocol: 1, League: r.Header.Get("X-TWR-League")}
	return r.TLS != nil && r.Method == http.MethodGet && r.ContentLength == 0 && r.Header.Get("Origin") == "" && r.URL.RawQuery == "" && raw.Validate() == nil
}
