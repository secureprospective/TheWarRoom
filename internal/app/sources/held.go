package sources

import (
	"slices"
	"sync"
)

// Held owns this feature's launch reports. It is never bound to a transport.
type Held struct {
	crosswalkMu sync.Mutex
	crosswalk   CrosswalkReport
	signalsMu   sync.Mutex
	signals     SignalsReport
}

func (h *Held) PublishCrosswalk(value CrosswalkReport) {
	h.crosswalkMu.Lock()
	defer h.crosswalkMu.Unlock()
	h.crosswalk = cloneCrosswalk(value)
}

func (h *Held) SnapshotCrosswalk() CrosswalkReport {
	h.crosswalkMu.Lock()
	defer h.crosswalkMu.Unlock()
	return cloneCrosswalk(h.crosswalk)
}

func (h *Held) PublishSignals(value SignalsReport) {
	h.signalsMu.Lock()
	defer h.signalsMu.Unlock()
	h.signals = cloneSignals(value)
}

func (h *Held) SnapshotSignals() SignalsReport {
	h.signalsMu.Lock()
	defer h.signalsMu.Unlock()
	return cloneSignals(h.signals)
}

func cloneCrosswalk(value CrosswalkReport) CrosswalkReport {
	value.Rates = slices.Clone(value.Rates)
	value.Unmatched = slices.Clone(value.Unmatched)
	return value
}

func cloneSignals(value SignalsReport) SignalsReport {
	value.Loads = slices.Clone(value.Loads)
	for i := range value.Loads {
		value.Loads[i].Missing = slices.Clone(value.Loads[i].Missing)
	}
	return value
}
