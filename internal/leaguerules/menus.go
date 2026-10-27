package leaguerules

type Position string
type LineupGroup string
type StatID string
type Action string
type Format string
type InitialRosters string
type CarryPolicy string
type Penalty string
type Criterion string
type CapMode string
type NFLStatus string
type ExperienceBasis string
type WaiverTrigger string
type WaiverMethod string
type BidMode string
type ReviewRoute string
type SchedulePattern string
type SeedPolicy string
type BoundarySide string
type DraftMode string
type ExpiryAction string
type OrderSource string
type OrderPattern string
type TieRule string

const (
	QB Position = "QB"
	RB Position = "RB"
	WR Position = "WR"
	TE Position = "TE"
	PK Position = "PK"
	DT Position = "DT"
	DE Position = "DE"
	LB Position = "LB"
	CB Position = "CB"
	S  Position = "S"
	// OL is a team's offensive line as one card (DEFAULT-32 spec §3), never a person.
	OL                   Position        = "OL"
	Offense              LineupGroup     = "offense"
	Defense              LineupGroup     = "defense"
	Total                LineupGroup     = "total"
	HeadToHead           Format          = "head_to_head"
	ByDraft              InitialRosters  = "draft"
	CarryAll             CarryPolicy     = "all"
	KeepSome             CarryPolicy     = "keepers"
	ZeroTopStarter       Penalty         = "zero_top_starter"
	ZeroSlot             Penalty         = "zero_slot"
	NoPenalty            Penalty         = "none"
	WinPercentage        Criterion       = "win_percentage"
	HeadToHeadRecord     Criterion       = "head_to_head"
	DivisionPercentage   Criterion       = "division_percentage"
	ConferencePercentage Criterion       = "conference_percentage"
	AllPlayPercentage    Criterion       = "all_play_percentage"
	Points               Criterion       = "points"
	PowerRank            Criterion       = "power_rank"
	HardCap              CapMode         = "hard"
	SoftCap              CapMode         = "soft"
	InjuredReserve       NFLStatus       = "IR"
	PUP                  NFLStatus       = "PUP"
	NFI                  NFLStatus       = "NFI"
	Out                  NFLStatus       = "Out"
	Suspended            NFLStatus       = "suspended"
	CalendarYears        ExperienceBasis = "calendar_years"
	CareerGames          ExperienceBasis = "career_games"
	// NFLSeasons counts only seasons the player spent in the NFL: no other league, no year out of it.
	NFLSeasons           ExperienceBasis = "nfl_seasons"
	OnCut                WaiverTrigger   = "cut"
	ReverseStandings     WaiverMethod    = "reverse_standings"
	Rolling              WaiverMethod    = "rolling"
	Blind                BidMode         = "blind"
	Open                 BidMode         = "open"
	DOTReview            ReviewRoute     = "dot"
	CommissionerReview   ReviewRoute     = "commissioner"
	NoReview             ReviewRoute     = "none"
	DivisionWeighted     SchedulePattern = "division_weighted"
	DivisionWinnersFirst SeedPolicy      = "division_winners_first"
	StartOfDay           BoundarySide    = "start_of_day"
	EndOfDay             BoundarySide    = "end_of_day"
	SlowDraft            DraftMode       = "slow"
	LiveDraft            DraftMode       = "live"
	SkipPick             ExpiryAction    = "skip"
	AutoPick             ExpiryAction    = "autopick"
	WorstStandingsFirst  OrderSource     = "standings"
	Linear               OrderPattern    = "linear"
	Snake                OrderPattern    = "snake"
)

type Bounds struct {
	Min Value[int]
	Max Value[int]
}

func bounds(minimum, maximum int, citation string) Bounds {
	return Bounds{Min: cited(minimum, citation), Max: cited(maximum, citation)}
}
func legacyLineupBounds() map[Position]Bounds {
	return map[Position]Bounds{
		QB: bounds(1, 1, "Census CM-004"), RB: bounds(1, 3, "Census CM-004"),
		WR: bounds(2, 5, "Census CM-004"), TE: bounds(1, 3, "Census CM-004"),
		PK: bounds(1, 1, "Census CM-004"), DT: bounds(2, 4, "Census CM-004"),
		DE: bounds(2, 4, "Census CM-004"), LB: bounds(2, 4, "Census CM-004"),
		CB: bounds(2, 4, "Census CM-004"), S: bounds(2, 4, "Census CM-004"),
	}
}
func legacyGroups() map[LineupGroup]Bounds {
	return map[LineupGroup]Bounds{
		Offense: {Min: Needs[int]("Census CM-004: only maximum stated"), Max: cited(8, "Census CM-004")},
		Defense: bounds(12, 12, "Census CM-004"), Total: bounds(21, 21, "Census CM-004"),
	}
}
