package envelope

import (
	"fmt"
	"sync"
)

type Filter struct{ LeagueID, FranchiseID, CorrelationID string }
type MemoryLog struct {
	mu       sync.Mutex
	receipts []Receipt
}

func NewMemoryLog() *MemoryLog { return &MemoryLog{receipts: []Receipt{}} }
func (m *MemoryLog) Append(e Envelope) error {
	if e.id == "" {
		return fmt.Errorf("envelope: cannot append unconstructed envelope")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receipts = append(m.receipts, e.Receipt())
	return nil
}
func (m *MemoryLog) List(f Filter) []Receipt {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Receipt{}
	for _, r := range m.receipts {
		if f.LeagueID != "" && f.LeagueID != r.Spec.LeagueID {
			continue
		}
		if f.FranchiseID != "" && f.FranchiseID != r.Spec.FranchiseID {
			continue
		}
		if f.CorrelationID != "" && f.CorrelationID != r.CorrelationID {
			continue
		}
		r.Spec = cloneSpec(r.Spec)
		r.Audit = append([]AuditEntry{}, r.Audit...)
		out = append(out, r)
	}
	return out
}
