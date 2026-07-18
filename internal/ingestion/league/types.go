package league

import "github.com/secureprospective/TheWarRoom/internal/ingestion"

// RawConfig is the league's rulebook EXACTLY as MFL returns it, assembled from the
// two MFL exports that carry league configuration: the `league` export (settings,
// roster limits, starter requirements, salary-cap amount) and the `rules` export
// (the position-additive scoring config). Every value is a raw string — MFL encodes
// all numbers and rule values as strings, and a Layer-1 fetcher transforms nothing
// (WF 1B). The `points` values keep their MFL form verbatim ("*0.5", "*.1", "3"):
// the leading "*" means "per unit of the stat", an absent "*" means a flat award.
// Interpreting that is the engine's job (Approach A); the rulebook only stores it.
type RawConfig struct {
	Source                string          // provenance, e.g. "mfl:2026"
	SalaryCapAmount       string          // league cap AMOUNT (AD-21), e.g. "120"
	RosterSize            string          // total roster slots, e.g. "80"
	TaxiSquad             string          // taxi-squad size, e.g. "8"
	InjuredReserve        string          // IR slots, e.g. "12"
	KeeperType            string          // "dynasty"
	UsesSalaries          string          // "1" when the league runs a salary cap
	UsesContractYear      string          // "1" when contracts carry a final year
	StartWeek             string          // first scoring week
	EndWeek               string          // last scoring week
	LastRegularSeasonWeek string          // last week before playoffs
	RosterLimits          []PositionLimit // per-position roster min-max ("0-0" = unlimited)
	Starters              Starters        // starter requirements
	ScoringRules          []PositionRuleSet
	Franchises            []Franchise // the league's franchise directory (id -> display name)
}

// Franchise is one entry in the league's franchise directory: its stable MFL id
// ("0001"–"0032", matching the rosters feed's franchise id) and its commissioner-set
// display name. It is DIRECTORY data (who the teams are), not a scoring rule — the
// rulebook stores it in the versioned payload and GetFranchises serves the name;
// an absent/empty name is tolerated (the UI falls back to the id).
type Franchise struct {
	ID   string
	Name string
}

// PositionLimit is one position's min-max bound ("QB" -> "1", "RB" -> "2-4"),
// raw from MFL's rosterLimits or starters position lists.
type PositionLimit struct {
	Name  string
	Limit string
}

// Starters holds the league's starter requirements: the total starter count plus
// the offense/IDP split and the per-position bounds.
type Starters struct {
	Count       string
	IOPStarters string // offense+kicker starter count
	IDPStarters string // defensive starter count
	Positions   []PositionLimit
}

// PositionRuleSet is one MFL `positionRules` block: a pipe-delimited position group
// ("CB|S", "DT|DE|LB|CB|S") and the rules that apply additively to it. MFL stacks
// these — a CB inherits the universal TK rule AND the "CB|S" TK overlay — so a
// position's true scoring is the union of every block whose Positions contains it.
type PositionRuleSet struct {
	Positions string // raw pipe-delimited group, e.g. "CB|S"
	Rules     []ScoringRule
}

// ScoringRule is one scoring event within a position group, raw from MFL. Event is
// the stat code (e.g. "PY", "TK", "FG", "MG"); Points is the raw award string
// ("*.05", "-3"); Range is the qualifying band ("0-39", "-50-999").
type ScoringRule struct {
	Event  string
	Points string
	Range  string
}

// --- MFL decode envelopes (internal) ---------------------------------------

// leagueEnvelope mirrors the MFL `league` export. Unknown fields are tolerated
// (external API, no versioning — the internal/schema unknown-field policy);
// correctness comes from validating the fields we depend on, not rejecting extras.
type leagueEnvelope struct {
	League struct {
		SalaryCapAmount       string `json:"salaryCapAmount"`
		RosterSize            string `json:"rosterSize"`
		TaxiSquad             string `json:"taxiSquad"`
		InjuredReserve        string `json:"injuredReserve"`
		KeeperType            string `json:"keeperType"`
		UsesSalaries          string `json:"usesSalaries"`
		UsesContractYear      string `json:"usesContractYear"`
		StartWeek             string `json:"startWeek"`
		EndWeek               string `json:"endWeek"`
		LastRegularSeasonWeek string `json:"lastRegularSeasonWeek"`
		RosterLimits          struct {
			Position ingestion.MFLList[posLimit] `json:"position"`
		} `json:"rosterLimits"`
		Starters struct {
			Count       string                      `json:"count"`
			IOPStarters string                      `json:"iop_starters"`
			IDPStarters string                      `json:"idp_starters"`
			Position    ingestion.MFLList[posLimit] `json:"position"`
		} `json:"starters"`
		Franchises struct {
			// MFL collapses a single-element array to a bare object; a 32-team league
			// returns an array, but MFLList tolerates both (same device as rosters).
			Franchise ingestion.MFLList[franchiseEntry] `json:"franchise"`
		} `json:"franchises"`
	} `json:"league"`
}

type posLimit struct {
	Name  string `json:"name"`
	Limit string `json:"limit"`
}

// franchiseEntry is one franchise row in the MFL `league` export's franchises block.
// It carries many fields (owner, division, contact); we decode only the id and the
// display name — the rest is not directory data the app needs.
type franchiseEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// rulesEnvelope mirrors the MFL `rules` export. positionRules and each block's rule
// list both use MFLList: MFL collapses a single-element array to a bare object, and
// a position group with exactly one rule is observed live (e.g. the "QB|WR|TE|DT|RB"
// #P block), so the collapse-tolerant decode is load-bearing here, not defensive.
type rulesEnvelope struct {
	Rules struct {
		PositionRules ingestion.MFLList[posRuleBlock] `json:"positionRules"`
	} `json:"rules"`
}

type posRuleBlock struct {
	Positions string                       `json:"positions"`
	Rule      ingestion.MFLList[ruleEntry] `json:"rule"`
}

// ruleEntry is one MFL rule. MFL's XML->JSON converter wraps each leaf value in a
// {"$t": "..."} object, so event/points/range each decode through ttext.
type ruleEntry struct {
	Event  ttext `json:"event"`
	Points ttext `json:"points"`
	Range  ttext `json:"range"`
}

// ttext unwraps MFL's {"$t": "value"} XML->JSON leaf wrapper to its inner string.
type ttext struct {
	T string `json:"$t"`
}
