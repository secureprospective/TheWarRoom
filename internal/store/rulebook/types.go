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

// ChangeSet is the result of a Reload: every difference between the active config
// and the newly fetched candidate version, plus the version numbers either side. It
// is the SIGNAL for the commissioner gate — a non-empty ChangeSet means MFL config
// drifted (a vote, an exploit fix) and a human must confirm a Promote before the new
// values go live. Reload NEVER auto-promotes: the active config stays stable.
type ChangeSet struct {
	FromVersion int // the active version the candidate was diffed against
	ToVersion   int // the newly stored candidate version
	Deltas      []RuleDelta
}

// Empty reports whether the candidate is identical to the active config.
func (c ChangeSet) Empty() bool { return len(c.Deltas) == 0 }

// Override is a commissioner delta layered over the MFL-sourced default. It is a
// SEPARATE record, never an in-place mutation of a default — so a Reload can re-pull
// MFL defaults without clobbering an override. Scope+Key identify the target value;
// Value is the raw replacement. v1.0 supports scalar overrides (cap, settings);
// structured changes (scoring tables) flow through MFL Reload + Promote instead.
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

// Override scopes. A SetOverride outside this set is rejected (the data-integrity
// half of the commissioner gate).
const (
	scopeCap     = "cap"     // the salary-cap amount
	scopeSetting = "setting" // a scalar league setting (roster size, weeks, …)
)

// capKey is the canonical override key under scopeCap.
const capKey = "salaryCapAmount"
