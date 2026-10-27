package nativecheck

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Standing is where a player on a reserve squad stands under the league's rules.
type Standing string

const (
	Stays     Standing = "stays"
	MustLeave Standing = "must_leave"
	Unclear   Standing = "unclear"
)

// ReserveVerdict is one IR or taxi player's plain verdict.
type ReserveVerdict struct {
	Player   string   `json:"player"`
	Name     string   `json:"name"`
	Squad    string   `json:"squad"`
	Standing Standing `json:"standing"`
	Words    string   `json:"words"`
}

// SquadUse is how full one reserve squad is.
type SquadUse struct {
	Used   int    `json:"used"`
	Slots  int    `json:"slots"`
	Judged bool   `json:"judged"`
	Words  string `json:"words"`
}

// ReserveCheck is one franchise's IR and taxi squads against the league's rules.
type ReserveCheck struct {
	IR      SquadUse         `json:"ir"`
	Taxi    SquadUse         `json:"taxi"`
	Players []ReserveVerdict `json:"players"`
}

// Entry says whether a move onto a reserve squad is allowed under this league's rules. It is a native
// answer only; it never says what MFL would accept.
type Entry struct {
	Allowed bool   `json:"allowed"`
	Known   bool   `json:"known"`
	Words   string `json:"words"`
}

// CheckReserves judges every IR and taxi player of a franchise, and how full each squad is.
func CheckReserves(ir leaguerules.IRRules, taxi leaguerules.TaxiRules, season int, players []RosterPlayer, facts Facts) ReserveCheck {
	check := ReserveCheck{Players: []ReserveVerdict{}}
	var onIR, onTaxi int
	for _, p := range players {
		f, _ := facts.Of(p.ID)
		switch p.Status {
		case domain.RosterIR:
			onIR++
			check.Players = append(check.Players, irStanding(ir, p.ID, f, p.Exceptions...))
		case domain.RosterTaxi:
			onTaxi++
			check.Players = append(check.Players, taxiStanding(taxi, season, p.ID, f, p.Exceptions...))
		case domain.RosterActive:
		}
	}
	check.IR = squadUse("IR", onIR, ir.Slots)
	check.Taxi = squadUse("Taxi", onTaxi, taxi.Slots)
	return check
}

func squadUse(name string, used int, slots leaguerules.Value[int]) SquadUse {
	u := SquadUse{Used: used}
	if slots.State != leaguerules.IsKnown {
		u.Words = "Needs a ruling: " + name + " spots"
		return u
	}
	u.Slots, u.Judged = slots.Value, true
	u.Words = fmt.Sprintf("%s %d of %d", name, used, slots.Value)
	if used > slots.Value {
		u.Words += fmt.Sprintf(": %d over", used-slots.Value)
	}
	return u
}

// qualification is whether a player meets a squad's entry test, or why that cannot be told.
type qualification int

const (
	qualifies qualification = iota
	disqualified
	undetermined
)

// irQualification tests the NFL's listing of a player against the designations the league accepts.
func irQualification(rules leaguerules.IRRules, f PlayerFacts, exceptions ...DecisionException) (qualification, string) {
	if why, allowed := ExceptionReason(exceptions, "ir.eligible"); allowed {
		return qualifies, why
	}
	switch {
	case rules.Eligible.State != leaguerules.IsKnown:
		return undetermined, "Needs a ruling: which NFL listings qualify for IR"
	case !f.DesignationHeld:
		return undetermined, "the NFL's listing of him is not held"
	case f.Designation == "":
		return disqualified, "the NFL no longer lists him as injured"
	case slices.Contains(rules.Eligible.Value, f.Designation):
		return qualifies, "the NFL lists him " + designationWords(f.Designation)
	}
	return disqualified, "the NFL lists him " + designationWords(f.Designation) + ", which this league does not accept for IR"
}

func irStanding(rules leaguerules.IRRules, id string, f PlayerFacts, exceptions ...DecisionException) ReserveVerdict {
	v := ReserveVerdict{Player: id, Name: f.Name, Squad: "IR"}
	q, why := irQualification(rules, f, exceptions...)
	switch {
	case q == undetermined:
		v.Standing, v.Words = Unclear, "Can't tell: "+why
	case q == qualifies:
		v.Standing, v.Words = Stays, "Stays on IR: "+why
	case rules.MustLeaveHealthy.State != leaguerules.IsKnown:
		v.Standing, v.Words = Unclear, "Needs a ruling: whether a player must leave IR when he no longer qualifies"
	case rules.MustLeaveHealthy.Value:
		v.Standing, v.Words = MustLeave, "Must come off IR: "+why
	default:
		v.Standing, v.Words = Stays, "May stay on IR: "+why+", and this league lets him stay"
	}
	return v
}

// CanEnterIR says whether a player may be placed on IR now, with used IR spots already taken.
func CanEnterIR(rules leaguerules.IRRules, f PlayerFacts, used int, exceptions ...DecisionException) Entry {
	q, why := irQualification(rules, f, exceptions...)
	switch {
	case q == undetermined:
		return Entry{Words: "Can't tell: " + why}
	case q == disqualified:
		return Entry{Known: true, Words: "Cannot go on IR: " + why}
	case rules.Slots.State != leaguerules.IsKnown:
		return Entry{Words: "Needs a ruling: IR spots"}
	case used >= rules.Slots.Value:
		return Entry{Known: true, Words: fmt.Sprintf("Cannot go on IR: all %d IR spots are taken", rules.Slots.Value)}
	}
	return Entry{Allowed: true, Known: true, Words: "May go on IR: " + why}
}
