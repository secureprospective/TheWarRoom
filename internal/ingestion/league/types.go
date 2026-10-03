package league

import "github.com/secureprospective/TheWarRoom/internal/ingestion"

// RawConfig is the league's rulebook exactly as MFL returns it, every value a raw string.
// Points keep MFL's form ("*0.5", "3"): a leading "*" means per unit of the stat, none means
// a flat award. The engine interprets them.
type RawConfig struct {
	Source                string // provenance, e.g. "mfl:2026"
	SalaryCapAmount       string // cap amount, e.g. "120"
	RosterSize            string // total roster slots, e.g. "80"
	TaxiSquad             string // taxi-squad size, e.g. "8"
	InjuredReserve        string // IR slots, e.g. "12"
	KeeperType            string // "dynasty"
	UsesSalaries          string // "1" when the league runs a salary cap
	UsesContractYear      string // "1" when contracts carry a final year
	StartWeek             string // first scoring week
	EndWeek               string // last scoring week
	LastRegularSeasonWeek string // last week before playoffs
	// The percent ("100") of a taxi or IR player's salary that counts against the cap.
	IncludeTaxiWithSalary       string
	IncludeIRWithSalary         string
	IncludeTaxiWithContractYear string          // taxi years count toward contract length: "0"/"100"
	RosterLimits                []PositionLimit // per-position roster min-max ("0-0" = unlimited)
	Starters                    Starters        // starter requirements
	ScoringRules                []PositionRuleSet
	Franchises                  []Franchise // the league's franchise directory (id -> display name)
	// CurrentSeason is the newest year MFL's league history lists under this league's id. It is
	// the app's one source for the season.
	CurrentSeason int
}

// Franchise is a directory entry: the MFL id ("0001"–"0032") and the display name, which may
// be blank.
type Franchise struct {
	ID   string
	Name string
}

// PositionLimit is one position's raw min-max ("QB" -> "1", "RB" -> "2-4").
type PositionLimit struct {
	Name  string
	Limit string
}

// Starters is the starter count, its offense/IDP split and the per-position bounds.
type Starters struct {
	Count       string
	IOPStarters string // offense+kicker starter count
	IDPStarters string // defensive starter count
	Positions   []PositionLimit
}

// PositionRuleSet is one MFL positionRules block. Blocks stack: a CB gets the universal TK rule
// and the "CB|S" overlay, so a position's scoring is every block that names it.
type PositionRuleSet struct {
	Positions string // raw pipe-delimited group, e.g. "CB|S"
	Rules     []ScoringRule
}

// ScoringRule is one raw scoring event: Event "PY", Points "*.05", Range "0-39".
type ScoringRule struct {
	Event  string
	Points string
	Range  string
}

// leagueEnvelope mirrors the MFL `league` export. Unknown fields are tolerated; validation
// covers the fields used.
type leagueEnvelope struct {
	League struct {
		ID                          string `json:"id"`
		SalaryCapAmount             string `json:"salaryCapAmount"`
		RosterSize                  string `json:"rosterSize"`
		TaxiSquad                   string `json:"taxiSquad"`
		InjuredReserve              string `json:"injuredReserve"`
		KeeperType                  string `json:"keeperType"`
		UsesSalaries                string `json:"usesSalaries"`
		UsesContractYear            string `json:"usesContractYear"`
		StartWeek                   string `json:"startWeek"`
		EndWeek                     string `json:"endWeek"`
		LastRegularSeasonWeek       string `json:"lastRegularSeasonWeek"`
		IncludeTaxiWithSalary       string `json:"includeTaxiWithSalary"`
		IncludeIRWithSalary         string `json:"includeIRWithSalary"`
		IncludeTaxiWithContractYear string `json:"includeTaxiWithContractYear"`
		RosterLimits                struct {
			Position ingestion.MFLList[posLimit] `json:"position"`
		} `json:"rosterLimits"`
		Starters struct {
			Count       string                      `json:"count"`
			IOPStarters string                      `json:"iop_starters"`
			IDPStarters string                      `json:"idp_starters"`
			Position    ingestion.MFLList[posLimit] `json:"position"`
		} `json:"starters"`
		Franchises struct {
			Franchise ingestion.MFLList[franchiseEntry] `json:"franchise"`
		} `json:"franchises"`
		History struct {
			League ingestion.MFLList[historyEntry] `json:"league"`
		} `json:"history"`
	} `json:"league"`
}

// historyEntry is one season of the league's history: its year and home page URL, which ends
// in that season's league id.
type historyEntry struct {
	Year string `json:"year"`
	URL  string `json:"url"`
}

type posLimit struct {
	Name  string `json:"name"`
	Limit string `json:"limit"`
}

// franchiseEntry decodes only the id and name of a franchise row.
type franchiseEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// rulesEnvelope mirrors the MFL `rules` export. MFLList matters here: a group with exactly one
// rule is live (the "QB|WR|TE|DT|RB" #P block) and arrives as a bare object.
type rulesEnvelope struct {
	Rules struct {
		PositionRules ingestion.MFLList[posRuleBlock] `json:"positionRules"`
	} `json:"rules"`
}

type posRuleBlock struct {
	Positions string                       `json:"positions"`
	Rule      ingestion.MFLList[ruleEntry] `json:"rule"`
}

// ruleEntry is one MFL rule; each leaf arrives wrapped as {"$t": "..."}.
type ruleEntry struct {
	Event  ttext `json:"event"`
	Points ttext `json:"points"`
	Range  ttext `json:"range"`
}

// ttext unwraps MFL's {"$t": "value"} XML->JSON leaf wrapper to its inner string.
type ttext struct {
	T string `json:"$t"`
}
