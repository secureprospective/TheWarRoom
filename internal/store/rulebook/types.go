package rulebook

// ChangeKind classifies how a single rule value changed between two config versions.
type ChangeKind string

const (
	// KindAdded is a rule/value present in the new config but not the old.
	KindAdded ChangeKind = "added"
	// KindRemoved is a rule/value present in the old config but not the new.
	KindRemoved ChangeKind = "removed"
	// KindChanged is a rule/value present in both with a different value.
	KindChanged ChangeKind = "changed"
)

// RuleDelta is one detected difference between the active config and a freshly
// fetched candidate. Field identifies what changed (a scalar name like
// "salaryCapAmount", or a scoring key like "scoring:CB|S/TK[0-99]"); Old/New carry
// the raw MFL values (one side empty for an add/remove).
type RuleDelta struct {
	Field string
	Kind  ChangeKind
	Old   string
	New   string
}

// ChangeSet is every difference between the active config and a newly fetched candidate. A
// non-empty set means MFL config drifted, and a human must Promote before the new values go
// live; Reload never promotes.
type ChangeSet struct {
	FromVersion int // the active version the candidate was diffed against
	ToVersion   int // the newly stored candidate version
	Deltas      []RuleDelta
}

// Empty reports whether the candidate is identical to the active config.
func (c ChangeSet) Empty() bool { return len(c.Deltas) == 0 }

// Override is a commissioner value layered over the MFL default at read time, never written
// into it, so a Reload cannot clobber it. Only scalars (cap, settings) can be overridden;
// scoring changes go through Reload and Promote.
type Override struct {
	Scope     string
	Key       string
	Value     string
	Note      string
	CreatedAt string
}

// VersionMeta describes one stored config version for the gate/audit surface.
type VersionMeta struct {
	Version   int
	Source    string
	CreatedAt string
	Active    bool
}

// Override scopes. SetOverride rejects any other.
const (
	scopeCap     = "cap"     // the salary-cap amount
	scopeSetting = "setting" // a scalar league setting (roster size, weeks, …)
)

// capKey is the canonical override key under scopeCap.
const capKey = "salaryCapAmount"
