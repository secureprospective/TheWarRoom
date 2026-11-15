package serverhttp

import (
	"sync"
	"time"
)

type RequestCounts struct {
	Calls      uint64 `json:"calls"`
	Refused    uint64 `json:"refused"`
	Computed   uint64 `json:"computed"`
	Cached     uint64 `json:"cached"`
	Unchanged  uint64 `json:"unchanged"`
	ReplyBytes uint64 `json:"replyBytes"`
}
type tallyKey struct{ command, mode string }
type RequestTally struct {
	mu     sync.Mutex
	since  time.Time
	counts map[tallyKey]RequestCounts
}

func NewRequestTally() *RequestTally {
	return &RequestTally{since: time.Now(), counts: make(map[tallyKey]RequestCounts)}
}
func (t *RequestTally) add(command, mode string, kind int, bytes uint64) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	k := tallyKey{command, mode}
	if _, ok := t.counts[k]; !ok && len(t.counts) >= 255 {
		k = tallyKey{command: "other"}
	}
	c := t.counts[k]
	switch kind {
	case 0:
		c.Calls++
	case 1:
		c.Refused++
	case 2:
		c.Computed++
	case 3:
		c.Cached++
	case 4:
		c.Unchanged++
	case 5:
		c.ReplyBytes += bytes
	}
	t.counts[k] = c
}
func (t *RequestTally) Unchanged(command, mode string) { t.add(command, mode, 4, 0) }
func (t *RequestTally) Snapshot() (time.Time, map[string]RequestCounts) {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]RequestCounts, len(t.counts))
	for k, v := range t.counts {
		key := k.command
		if k.mode != "" {
			key += " " + k.mode
		}
		out[key] = v
	}
	return t.since, out
}
func (t *RequestTally) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.since = time.Now()
	t.counts = make(map[tallyKey]RequestCounts)
}
