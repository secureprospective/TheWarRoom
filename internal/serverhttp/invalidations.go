package serverhttp

import (
	"context"
	"fmt"
	"math/rand/v2"
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
// Each Handler owns its connection budget and one head watcher per league with open streams. No event
// history, ledger cursor or bearer is published. A slow consumer retains only its last observed view.
func (h Hosted) invalidationHandler(next http.Handler) http.Handler {
	budget := &streamBudget{devices: make(map[string]int), active: make(map[string]int)}
	watch := newLeagueWatch(h.leagueHead, headPollEvery)
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
		if h.Store == nil || h.Leagues == nil || h.Sessions == nil || h.TOTP == nil || h.Worker == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		service, gate, device := h.invalidationService(w, r, league)
		if service == nil {
			return
		}
		// Read the head before the view so a concurrent commit cannot be skipped.
		head, err := h.leagueHead(r.Context(), league)
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
		pin := h.pin(r.Context(), league)
		if pin.Err != nil {
			http.Error(w, "native read refused", http.StatusForbidden)
			return
		}
		seated, err := leagueservice.JoinSeats{Store: pin.Store}.Seated(r.Context(), league)
		if err != nil {
			http.Error(w, "native read unavailable", http.StatusServiceUnavailable)
			return
		}
		if !budget.acquire(league, device, seated) {
			http.Error(w, "too many streams", http.StatusTooManyRequests)
			return
		}
		defer budget.release(league, device)
		// Subscribing after the head read is safe: the watcher hands a new subscriber the newest head it
		// has seen, and any later commit reaches it on the next poll.
		news, unsubscribe := watch.subscribe(league) //nolint:contextcheck // the watcher outlives any one request

		defer unsubscribe()
		h.streamRecipientInvalidations(w, r, service, league, device, head, reading.Revision, news, proofLapse(lifetime, limited))
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

func (h Hosted) streamRecipientInvalidations(w http.ResponseWriter, r *http.Request, service *leagueservice.Service, league, device string, head, revision int64, news <-chan headNews, lapse <-chan time.Time) {
	w.Header().Set("Content-Type", "text/event-stream")
	controller := http.NewResponseController(w)
	poll := invalidationPoll{head: head, revision: revision,
		deadline: func(ctx context.Context) (time.Time, time.Time, bool) {
			//nolint:contextcheck // StoredHost owns the request context.
			_, next, ok, err := service.ReadingEpoch(league, h.pin(ctx, league).Football.Kickoffs())
			return next, h.Store.Now(), ok && err == nil
		},
		now: h.Store.Now,
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
	streamInvalidations(r.Context(), controller, w, streamWakes{news: news, lapse: lapse, session: sessionCheckEvery}, &poll, checkSession)
}

// The global head is only a work gate; the authorized recipient token alone wakes peers.
// A deadline is an event too: when the clock passes the league's next public deadline
// (commissioner.ReadingEpoch) its readings change with no new entry, so every recipient is woken. Only
// public deadlines and kickoffs wake anyone: a private entry's time never does.
type invalidationPoll struct {
	head, revision int64
	read           func(context.Context) (commissioner.Ledger, error)
	deadline       func(context.Context) (next, now time.Time, ok bool)
	now            func() time.Time
	next           time.Time
}

// changed reports whether this recipient's view moved, given the league head the watcher last saw.
func (p *invalidationPoll) changed(ctx context.Context, head int64) (bool, error) {
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

// untilDeadline is how long until the next public deadline; ok is false for an untimed league.
func (p *invalidationPoll) untilDeadline() (time.Duration, bool) {
	if p.next.IsZero() || p.now == nil {
		return 0, false
	}
	return max(p.next.Sub(p.now()), 0), true
}

// arm sets timer to fire at the next public deadline, or not at all for an untimed league.
func (p *invalidationPoll) arm(timer *time.Timer) {
	timer.Stop()
	if wait, ok := p.untilDeadline(); ok {
		timer.Reset(wait)
	}
}

// wake re-reads this recipient's view at head, emits if it moved and re-arms the deadline timer; false
// ends the stream.
func (p *invalidationPoll) wake(ctx context.Context, controller *http.ResponseController, w http.ResponseWriter, deadline *time.Timer, head int64) bool {
	changed, err := p.changed(ctx, head)
	p.arm(deadline)
	return err == nil && (!changed || emitInvalidation(controller, w) == nil)
}

// headPollEvery is the league watcher's cadence; one read per league, whatever the device count.
const headPollEvery = 250 * time.Millisecond

// sessionCheckEvery spaces one stream's session checks 2-4 s apart, jittered so a league's devices do
// not read in step. A revoked session therefore ends its stream within 4 s plus one check.
func sessionCheckEvery() time.Duration { return 2*time.Second + rand.N(2*time.Second) } //nolint:gosec // jitter, not a secret

// streamWakes are the per-stream signals besides the heartbeat: the league watcher's head news, the
// recent-auth lapse and the session check cadence.
type streamWakes struct {
	news    <-chan headNews
	lapse   <-chan time.Time
	session func() time.Duration
}

func streamInvalidations(ctx context.Context, controller *http.ResponseController, w http.ResponseWriter, wakes streamWakes, poll *invalidationPoll, checkSession func(context.Context) error) {
	session := time.NewTimer(wakes.session())
	defer session.Stop()
	// The heartbeat also refreshes the deadline, so a kickoff change with no commit is seen within 15 s.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	deadline := time.NewTimer(time.Hour)
	defer deadline.Stop()
	poll.arm(deadline)
	wake := func(head int64) bool { return poll.wake(ctx, controller, w, deadline, head) }
	for {
		select {
		case <-ctx.Done():
			return
		case <-wakes.lapse:
			return
		case news := <-wakes.news:
			if !headWake(ctx, news, poll.head, checkSession) || !wake(news.head) {
				return
			}
		case <-deadline.C:
			if !wake(poll.head) {
				return
			}
		case <-session.C:
			if checkSession(ctx) != nil {
				return
			}
			session.Reset(wakes.session())
		case <-heartbeat.C:
			if !wake(poll.head) || keepalive(controller, w) != nil {
				return
			}
		}
	}
}

// headWake reports whether a watcher reading lets the stream go on: a read error ends it, and a moved
// head checks the session at once.
func headWake(ctx context.Context, news headNews, head int64, checkSession func(context.Context) error) bool {
	return news.err == nil && (news.head == head || checkSession(ctx) == nil)
}

func keepalive(controller *http.ResponseController, w http.ResponseWriter) error {
	if err := controller.SetWriteDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return fmt.Errorf("stream deadline: %w", err)
	}
	if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
		return fmt.Errorf("stream keepalive: %w", err)
	}
	if err := controller.Flush(); err != nil {
		return fmt.Errorf("flush keepalive: %w", err)
	}
	return nil
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

func (h Hosted) invalidationService(w http.ResponseWriter, r *http.Request, league string) (*leagueservice.Service, accountAdmission, string) {
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
	pin := h.pin(r.Context(), league)
	if pin.Err != nil {
		http.Error(w, "native read refused", http.StatusForbidden)
		return nil, accountAdmission{}, ""
	}
	host := leagueservice.StoredHost{Store: pin.Store, Context: r.Context, Principal: string(principal), Registry: h.Registry}
	gate := accountAdmission{host: host, sessions: h.Sessions.Admission(session), totp: h.TOTP, proof: accounts.RecentAuth{Token: r.Header.Get("X-TWR-Recent-Auth")}}
	service := leagueservice.NewHosted(h.Worker.Host(host), gate)
	return service, gate, session.Bearer
}

func validInvalidationRequest(r *http.Request) bool {
	raw := RawNativeRequest{Protocol: 1, League: r.Header.Get("X-TWR-League")}
	return r.TLS != nil && r.Method == http.MethodGet && r.ContentLength == 0 && r.Header.Get("Origin") == "" && r.URL.RawQuery == "" && raw.Validate() == nil
}
