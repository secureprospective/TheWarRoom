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

// Stream budget: each league holds at most streamsPerSeat per seated franchise plus streamHeadroom (the
// commissioner's and an unseated league's devices), and one device holds at most streamsPerDevice.
const (
	streamsPerSeat   = 3
	streamHeadroom   = 8
	streamsPerDevice = 2
)

// leagueStreamBudget scales with the league so every seated owner's phone, desktop and tablet fit.
func leagueStreamBudget(seated int) int { return streamsPerSeat*max(seated, 0) + streamHeadroom }

// streamBudget counts open streams per league and per device.
type streamBudget struct {
	mu      sync.Mutex
	devices map[string]int
	active  map[string]int
}

// acquire takes one stream slot for league and device; release must follow a true result.
func (b *streamBudget) acquire(league, device string, seated int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active[league] >= leagueStreamBudget(seated) || b.devices[device] >= streamsPerDevice {
		return false
	}
	b.active[league]++
	b.devices[device]++
	return true
}

func (b *streamBudget) release(league, device string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active[league]--; b.active[league] == 0 {
		delete(b.active, league)
	}
	if b.devices[device]--; b.devices[device] == 0 {
		delete(b.devices, device)
	}
}

// FUTURE-SEAM(stream): live push for new features rides this "changed for you" signal; it never
// carries event contents. See docs/architecture/future-seams.md.
// Each Handler owns its connection budget. No event history, ledger cursor or bearer
// is published. A slow consumer retains only its last observed recipient view.
func (h Hosted) invalidationHandler(next http.Handler) http.Handler {
	budget := &streamBudget{devices: make(map[string]int), active: make(map[string]int)}
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
		seated, err := leagueservice.JoinSeats{Store: h.Store}.Seated(r.Context(), league)
		if err != nil {
			http.Error(w, "native read unavailable", http.StatusServiceUnavailable)
			return
		}
		if !budget.acquire(league, device, seated) {
			http.Error(w, "too many streams", http.StatusTooManyRequests)
			return
		}
		defer budget.release(league, device)
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
	poll := invalidationPoll{head: head, revision: revision,
		committedHead: func(ctx context.Context) (int64, error) {
			return h.Store.CommittedHead(ctx, league, commissioner.Stream)
		},
		deadline: func(context.Context) (time.Time, time.Time, bool) {
			//nolint:contextcheck // StoredHost owns the request context.
			_, next, ok, err := service.ReadingEpoch(league, h.Football.Kickoffs())
			return next, h.Store.Now(), ok && err == nil
		},
		read: func(context.Context) (commissioner.Ledger, error) {
			//nolint:contextcheck // StoredHost owns the request context.
			return service.TargetCommissionerLedger(league)
		},
	}
	// The deadline is taken before the first wake, so a deadline right after it still wakes the device.
	if next, _, ok := poll.deadline(r.Context()); ok {
		poll.next = next
	}
	if emitInvalidation(controller, w) != nil {
		return
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
// A deadline is an event too: when the clock passes the league's next public deadline
// (commissioner.ReadingEpoch) its readings change with no new entry, so every recipient is woken. Only
// public deadlines and kickoffs wake anyone: a private entry's time never does.
type invalidationPoll struct {
	head, revision int64
	committedHead  func(context.Context) (int64, error)
	read           func(context.Context) (commissioner.Ledger, error)
	deadline       func(context.Context) (next, now time.Time, ok bool)
	next           time.Time
}

func (p *invalidationPoll) changed(ctx context.Context) (bool, error) {
	head, err := p.committedHead(ctx)
	if err != nil {
		return false, fmt.Errorf("poll committed head: %w", err)
	}
	next, epochPassed := p.next, false
	if p.deadline != nil {
		upcoming, now, ok := p.deadline(ctx)
		epochPassed = !p.next.IsZero() && !now.Before(p.next)
		if ok {
			next = upcoming
		} else if epochPassed {
			next = time.Time{}
		}
	}
	if head == p.head {
		p.next = next
		return epochPassed, nil
	}
	reading, err := p.read(ctx)
	if err != nil {
		return false, fmt.Errorf("poll recipient view: %w", err)
	}
	changed := reading.Revision != p.revision || epochPassed
	p.head, p.revision = head, reading.Revision
	p.next = next
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
		h.sessionFailure(w, err)
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
