package nativecheck

import (
	"cmp"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// AutoLineupCandidate is one active-roster player considered for an automatic lineup.
type AutoLineupCandidate struct {
	ID        string
	Position  leaguerules.Position
	Median    float64
	Ceiling   float64
	Available bool
	Locked    bool
	Eligible  bool
}

// IneligibleDesignation reports whether an NFL injury or status listing rules a player out of an
// automatic lineup. Questionable and empty designations remain eligible.
func IneligibleDesignation(status leaguerules.NFLStatus) bool {
	switch status {
	case leaguerules.Out, leaguerules.InjuredReserve, leaguerules.PUP,
		leaguerules.NFI, leaguerules.Suspended:
		return true
	}
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "out", "doubtful", "ir", "ir-r", "injured reserve", "reserve",
		"pup", "ir-pup", "nfi", "ir-nfi", "suspended":
		return true
	default:
		return false
	}
}

// SelectAutoLineup picks the highest-projected legal starter set under rules. Locked starters stay
// unchanged; unlocked slots are filled by position minimum, then group minimum, then remaining total
// slots while respecting all position and group maxima. If no legal lineup exists, it returns the
// best partial set.
func SelectAutoLineup(rules leaguerules.LineupRules, candidates []AutoLineupCandidate) []string {
	state := newAutoLineupState(rules, candidates)
	state.fillPositionMinimums()
	state.fillGroupMinimums()
	state.fillRemaining()
	slices.SortFunc(state.selected, func(a, b AutoLineupCandidate) int {
		if c := cmp.Compare(positionRank(a.Position), positionRank(b.Position)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	out := make([]string, len(state.selected))
	for i, c := range state.selected {
		out[i] = c.ID
	}
	return out
}

type autoLineupState struct {
	rules      leaguerules.LineupRules
	pool       []AutoLineupCandidate
	used       []bool
	reserve    []AutoLineupCandidate // allowed but benched (bye, injury): only to meet a position minimum
	reserveUse []bool
	selected   []AutoLineupCandidate
	posCount   map[leaguerules.Position]int
	groupCount map[leaguerules.LineupGroup]int
}

func newAutoLineupState(rules leaguerules.LineupRules, candidates []AutoLineupCandidate) *autoLineupState {
	s := &autoLineupState{
		rules:      rules,
		posCount:   map[leaguerules.Position]int{},
		groupCount: map[leaguerules.LineupGroup]int{},
	}
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.ID == "" || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		if c.Locked {
			s.addSelected(c)
			continue
		}
		if !s.positionAllowed(c.Position) {
			continue
		}
		if !c.Eligible {
			s.reserve = append(s.reserve, c)
			continue
		}
		s.pool = append(s.pool, c)
	}
	slices.SortFunc(s.pool, compareAutoCandidate)
	slices.SortFunc(s.reserve, compareAutoCandidate)
	s.used = make([]bool, len(s.pool))
	s.reserveUse = make([]bool, len(s.reserve))
	return s
}

func compareAutoCandidate(a, b AutoLineupCandidate) int {
	aMed, aCeil := effectiveProjection(a)
	bMed, bCeil := effectiveProjection(b)
	if c := cmp.Compare(bMed, aMed); c != 0 {
		return c
	}
	if c := cmp.Compare(bCeil, aCeil); c != 0 {
		return c
	}
	return cmp.Compare(a.ID, b.ID)
}

func effectiveProjection(c AutoLineupCandidate) (float64, float64) {
	if !c.Available {
		return 0, 0
	}
	return c.Median, c.Ceiling
}

func (s *autoLineupState) positionAllowed(pos leaguerules.Position) bool {
	if s.rules.Positions.State != leaguerules.IsKnown {
		return pos != ""
	}
	_, ok := s.rules.Positions.Value[pos]
	return ok
}

func (s *autoLineupState) addSelected(c AutoLineupCandidate) {
	s.selected = append(s.selected, c)
	s.posCount[c.Position]++
	s.groupCount[leaguerules.Total]++
	if side := sideOf(c.Position); side != leaguerules.Total {
		s.groupCount[side]++
	}
}

func (s *autoLineupState) canAdd(pos leaguerules.Position) bool {
	if s.rules.Positions.State == leaguerules.IsKnown {
		b, ok := s.rules.Positions.Value[pos]
		if !ok || (b.Max.State == leaguerules.IsKnown && s.posCount[pos] >= b.Max.Value) {
			return false
		}
	}
	if s.rules.Groups.State == leaguerules.IsKnown {
		if side := sideOf(pos); side != leaguerules.Total {
			if b, ok := s.rules.Groups.Value[side]; ok &&
				b.Max.State == leaguerules.IsKnown && s.groupCount[side] >= b.Max.Value {
				return false
			}
		}
		if b, ok := s.rules.Groups.Value[leaguerules.Total]; ok &&
			b.Max.State == leaguerules.IsKnown && s.groupCount[leaguerules.Total] >= b.Max.Value {
			return false
		}
	}
	return true
}

func (s *autoLineupState) fillPositionMinimums() {
	if s.rules.Positions.State != leaguerules.IsKnown {
		return
	}
	for _, pos := range sortedPositions(s.rules.Positions.Value) {
		b := s.rules.Positions.Value[pos]
		if b.Min.State != leaguerules.IsKnown {
			continue
		}
		atPos := func(c AutoLineupCandidate) bool { return c.Position == pos }
		for s.posCount[pos] < b.Min.Value {
			if !s.pickNext(atPos) && !s.pickReserve(atPos) {
				break
			}
		}
	}
}

func (s *autoLineupState) fillGroupMinimums() {
	if s.rules.Groups.State != leaguerules.IsKnown {
		return
	}
	for _, side := range []leaguerules.LineupGroup{leaguerules.Offense, leaguerules.Defense} {
		b, ok := s.rules.Groups.Value[side]
		if !ok || b.Min.State != leaguerules.IsKnown {
			continue
		}
		for s.groupCount[side] < b.Min.Value {
			if !s.pickNext(func(c AutoLineupCandidate) bool { return sideOf(c.Position) == side }) {
				break
			}
		}
	}
}

func (s *autoLineupState) fillRemaining() {
	target := s.totalTarget()
	for i, c := range s.pool {
		if target > 0 && s.groupCount[leaguerules.Total] >= target {
			return
		}
		if s.used[i] || !s.canAdd(c.Position) {
			continue
		}
		s.used[i] = true
		s.addSelected(c)
	}
}

func (s *autoLineupState) totalTarget() int {
	if s.rules.Groups.State != leaguerules.IsKnown {
		return 0
	}
	b, ok := s.rules.Groups.Value[leaguerules.Total]
	if !ok {
		return 0
	}
	if b.Max.State == leaguerules.IsKnown {
		return b.Max.Value
	}
	if b.Min.State == leaguerules.IsKnown {
		return b.Min.Value
	}
	return 0
}

func (s *autoLineupState) pickNext(match func(AutoLineupCandidate) bool) bool {
	for i, c := range s.pool {
		if s.used[i] || !match(c) || !s.canAdd(c.Position) {
			continue
		}
		s.used[i] = true
		s.addSelected(c)
		return true
	}
	return false
}

// pickReserve starts a benched player (bye week, injury) only when no playing player can meet a
// position minimum: a legal lineup with a zero in one slot beats an invalid one the league penalises
// (a team's only O-Line card on its bye week).
func (s *autoLineupState) pickReserve(match func(AutoLineupCandidate) bool) bool {
	for i, c := range s.reserve {
		if s.reserveUse[i] || !match(c) || !s.canAdd(c.Position) {
			continue
		}
		s.reserveUse[i] = true
		s.addSelected(c)
		return true
	}
	return false
}
