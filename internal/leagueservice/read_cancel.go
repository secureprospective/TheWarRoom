package leagueservice

import (
	"context"
	"fmt"
	"sync"
)

// maxWithdrawn bounds the withdrawals kept for reads that have not started yet.
const maxWithdrawn = 256

// readCancels lets a client withdraw a read it named, so a read stops when the screen that asked for it goes
// away. A withdrawal that arrives before its read starts is kept, so that read starts already cancelled.
type readCancels struct {
	mu        sync.Mutex
	running   map[string]context.CancelFunc
	withdrawn map[string]bool
}

// watch derives a context the named request can cancel. An empty request cannot be withdrawn.
func (r *readCancels) watch(ctx context.Context, request string) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	if request == "" {
		return ctx, cancel
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.withdrawn[request] {
		delete(r.withdrawn, request)
		cancel()
		return ctx, cancel
	}
	if r.running == nil {
		r.running = map[string]context.CancelFunc{}
	}
	r.running[request] = cancel
	return ctx, func() {
		r.mu.Lock()
		delete(r.running, request)
		r.mu.Unlock()
		cancel()
	}
}

func (r *readCancels) withdraw(request string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cancel, ok := r.running[request]; ok {
		delete(r.running, request)
		cancel()
		return
	}
	if r.withdrawn == nil || len(r.withdrawn) >= maxWithdrawn {
		r.withdrawn = map[string]bool{}
	}
	r.withdrawn[request] = true
}

// validRequest admits a client-chosen request name: empty, or up to 64 letters, digits and dashes.
func validRequest(request string) error {
	if len(request) > 64 {
		return fmt.Errorf("read request %q: longer than 64 characters", request)
	}
	for _, c := range request {
		if (c < 0x30 || c > 0x39) && (c < 0x41 || c > 0x5a) && (c < 0x61 || c > 0x7a) && c != 0x2d {
			return fmt.Errorf("read request %q: only letters, digits and dashes", request)
		}
	}
	return nil
}

// requestKey scopes a hosted request to its principal, so one person cannot withdraw another person's read.
func (a *Service) requestKey(request string) string {
	if request == "" || !a.hosted {
		return request
	}
	principal, _ := a.host.CommissionerPrincipal()
	return principal + "\x00" + request
}

// TargetCancelRead withdraws the read the client named request. Joined readers of the same reading keep it
// running; it stops only when none waits. An unknown or finished request is not an error.
func (a *Service) TargetCancelRead(request string) error {
	if err := validRequest(request); err != nil {
		return err
	}
	if request != "" {
		a.cancels.withdraw(a.requestKey(request))
	}
	return nil
}
