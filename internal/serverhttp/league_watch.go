package serverhttp

import (
	"context"
	"sync"
	"time"
)

// headNews is one committed-head reading from a league watcher; err ends the streams that receive it.
type headNews struct {
	head int64
	err  error
}

// leagueWatch runs one head poll per league with open streams and hands each reading that differs from
// the last to every stream of that league. Idle cost is one read per league per tick, not per device.
type leagueWatch struct {
	head    func(context.Context, string) (int64, error)
	every   time.Duration
	mu      sync.Mutex
	leagues map[string]*leagueObserver
}

type leagueObserver struct {
	subs map[chan headNews]struct{}
	last *headNews
	stop context.CancelFunc
	done chan struct{}
}

func newLeagueWatch(head func(context.Context, string) (int64, error), every time.Duration) *leagueWatch {
	return &leagueWatch{head: head, every: every, leagues: make(map[string]*leagueObserver)}
}

// subscribe joins league's watcher, starting it if this is the first stream. The channel holds only the
// newest reading, and a new subscriber is handed the newest reading already seen. unsubscribe stops the
// watcher when the last stream leaves.
func (w *leagueWatch) subscribe(league string) (<-chan headNews, func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	observer := w.leagues[league]
	if observer == nil {
		ctx, stop := context.WithCancel(context.Background())
		observer = &leagueObserver{subs: make(map[chan headNews]struct{}), stop: stop, done: make(chan struct{})}
		w.leagues[league] = observer
		go w.run(ctx, league, observer)
	}
	news := make(chan headNews, 1)
	observer.subs[news] = struct{}{}
	if observer.last != nil {
		news <- *observer.last
	}
	return news, func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		delete(observer.subs, news)
		if len(observer.subs) == 0 && w.leagues[league] == observer {
			observer.stop()
			delete(w.leagues, league)
		}
	}
}

func (w *leagueWatch) run(ctx context.Context, league string, observer *leagueObserver) {
	defer close(observer.done)
	ticker := time.NewTicker(w.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		head, err := w.head(ctx, league)
		if ctx.Err() != nil {
			return
		}
		reading := headNews{head: head, err: err}
		w.mu.Lock()
		if observer.last == nil || *observer.last != reading {
			observer.last = &reading
			for news := range observer.subs {
				offer(news, reading)
			}
		}
		w.mu.Unlock()
	}
}

// offer replaces any unread reading with the newer one; the watcher is the only sender.
func offer(news chan headNews, reading headNews) {
	select {
	case <-news:
	default:
	}
	news <- reading
}
