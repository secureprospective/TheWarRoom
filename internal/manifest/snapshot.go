package manifest

import "time"

// Attempt is one compile over a manifest: a good one has a ResultSHA, a failed one an Error.
type Attempt struct {
	ManifestSHA string
	Compiler    string
	RulesHash   string
	ResultSHA   string
	Error       string
	BuiltAt     time.Time
}

// State is a compiled snapshot's standing.
type State string

const (
	StateNone    State = "none"    // nothing has compiled yet
	StateCurrent State = "current" // the last good snapshot matches the inputs, compiler and rules
	StateStale   State = "stale"   // the last good snapshot is kept, and Reason says why it is behind
)

// Status is the snapshot a reader gets: the last good attempt, if any, and its standing.
type Status struct {
	State  State
	Good   Attempt
	Reason string
}

// Want is what a current snapshot must have been compiled from.
type Want struct {
	ManifestSHA string
	Compiler    string
	RulesHash   string
}

// StatusOf judges attempts, newest first, against want. A failed build never replaces the last
// good snapshot; it marks it stale.
func StatusOf(attempts []Attempt, want Want) Status {
	good := -1
	for i, a := range attempts {
		if a.ResultSHA != "" {
			good = i
			break
		}
	}
	if good < 0 {
		if len(attempts) > 0 {
			return Status{State: StateNone, Reason: "Every build so far failed; the latest: " + attempts[0].Error}
		}
		return Status{State: StateNone, Reason: "Nothing has been compiled from these inputs yet."}
	}
	s := Status{State: StateStale, Good: attempts[good]}
	switch {
	case good > 0:
		s.Reason = "The latest build failed, so the last good copy is kept: " + attempts[0].Error
	case s.Good.ManifestSHA != want.ManifestSHA:
		s.Reason = "The inputs changed after it was built."
	case s.Good.Compiler != want.Compiler:
		s.Reason = "The compiler changed after it was built."
	case s.Good.RulesHash != want.RulesHash:
		s.Reason = "The rules changed after it was built."
	default:
		s.State = StateCurrent
	}
	return s
}
