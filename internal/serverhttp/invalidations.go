package serverhttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
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
		service, gate, device := h.invalidationService(w, r)
		if service == nil {
			return
		}
		// Read the head before the view so a concurrent commit cannot be skipped.
		head, err := h.Store.CommittedHead(r.Context(), league, commissioner.Stream)
		if err != nil {
			http.Error(w, "native read unavailable", http.StatusServiceUnavailable)
			return
		}
		//nolint:contextcheck // StoredHost owns the request context used by the reading API.
		reading, err := service.TargetCommissionerLedger(league)
		if err != nil {
			http.Error(w, "native read refused", http.StatusForbidden)
			return
		}
		lifetime, limited, err := gate.proofRemaining(r.Context(), league)
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
		h.streamRecipientInvalidations(w, r, service, league, device, head, reading.Revision, proofLapse(lifetime, limited))
	})
}

// proofLapse fires when a privileged stream's recent-auth proof lapses; nil never fires. The timer is
// left to the runtime: an unreferenced timer is collected whether or not it fired.
func proofLapse(lifetime time.Duration, limited bool) <-chan time.Time {
	if !limited {
		return nil
	}
	return time.NewTimer(lifetime).C
}

func (h Hosted) streamRecipientInvalidations(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, league, device string, head, revision int64, lapse <-chan time.Time) {
	w.Header().Set("Content-Type", "text/event-stream")
	controller := http.NewResponseController(w)
	if emitInvalidation(controller, w) != nil {
		return
	}
	poll := invalidationPoll{head: head, revision: revision,
		committedHead: func(ctx context.Context) (int64, error) {
			return h.Store.CommittedHead(ctx, league, commissioner.Stream)
		},
		read: func(context.Context) (commissioner.Ledger, error) {
			//nolint:contextcheck // StoredHost owns the request context.
			return service.TargetCommissionerLedger(league)
		},
	}
	checkSession := func(ctx context.Context) error {
		_, err := h.Sessions.Check(ctx, accounts.RawSession{Bearer: device})
		if err != nil {
			return fmt.Errorf("stream session: %w", err)
		}
		return nil
	}
	streamInvalidations(r.Context(), controller, w, &poll, checkSession, lapse)
}

// The global head is only a work gate; the authorized recipient token alone wakes peers.
type invalidationPoll struct {
	head, revision int64
	committedHead  func(context.Context) (int64, error)
	read           func(context.Context) (commissioner.Ledger, error)
}

func (p *invalidationPoll) changed(ctx context.Context) (bool, error) {
	head, err := p.committedHead(ctx)
	if err != nil {
		return false, fmt.Errorf("poll committed head: %w", err)
	}
	if head == p.head {
		return false, nil
	}
	reading, err := p.read(ctx)
	if err != nil {
		return false, fmt.Errorf("poll recipient view: %w", err)
	}
	changed := reading.Revision != p.revision
	p.head, p.revision = head, reading.Revision
	return changed, nil
}

func streamInvalidations(ctx context.Context, controller *http.ResponseController, w http.ResponseWriter, poll *invalidationPoll, checkSession func(context.Context) error, lapse <-chan time.Time) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	session := time.NewTicker(500 * time.Millisecond)
	defer session.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-lapse:
			return
		case <-ticker.C:
			changed, err := poll.changed(ctx)
			if err != nil {
				return
			}
			if changed && emitInvalidation(controller, w) != nil {
				return
			}
		case <-session.C:
			if checkSession(ctx) != nil {
				return
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

func (h Hosted) invalidationService(w http.ResponseWriter, r *http.Request) (*leagueservice.Service, accountAdmission, string) {
	authorization := r.Header.Get("Authorization")
	if len(r.Header.Values("Authorization")) != 1 || !strings.HasPrefix(authorization, "Bearer ") {
		http.Error(w, "session required", http.StatusUnauthorized)
		return nil, accountAdmission{}, ""
	}
	session := accounts.RawSession{Bearer: strings.TrimPrefix(authorization, "Bearer ")}
	account, err := h.Sessions.Authenticate(r.Context(), session)
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return nil, accountAdmission{}, ""
	}
	principal, err := account.Principal()
	if err != nil {
		http.Error(w, "session refused", http.StatusUnauthorized)
		return nil, accountAdmission{}, ""
	}
	host := leagueservice.StoredHost{Store: h.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(session), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	service := leagueservice.NewHosted(h.Worker.Host(host), gate)
	return service, gate, session.Bearer
}

func validInvalidationRequest(r *http.Request) bool {
	raw := RawNativeRequest{Protocol: 1, League: r.Header.Get("X-TWR-League")}
	return r.TLS != nil && r.Method == http.MethodGet && r.ContentLength == 0 && r.Header.Get("Origin") == "" && r.URL.RawQuery == "" && raw.Validate() == nil
}
