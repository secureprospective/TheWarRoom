package nativecheck

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// PlayerFacts are the football facts a check needs about one player. A fact the app does not hold is
// marked as not held, and the check says so; it is never guessed.
type PlayerFacts struct {
	Name     string
	Position leaguerules.Position // empty when not held
	// Team is the NFL team code the schedule uses; empty or FA when the player has no NFL team.
	Team string
	// Designation is the NFL's injury listing; empty means the NFL lists the player with none.
	Designation     leaguerules.NFLStatus
	DesignationHeld bool
	DraftYear       int
	DraftYearHeld   bool
	CareerGames     int
	CareerGamesHeld bool
	// NFLSeasons counts the seasons the player has spent in the NFL, this one included; no feed holds it yet.
	NFLSeasons     int
	NFLSeasonsHeld bool
}

// Facts holds PlayerFacts by canonical player id.
type Facts map[string]PlayerFacts

// positionRank orders positions the way a lineup is read: offense, kicker, O-Line, defense, then any other.
func positionRank(p leaguerules.Position) int {
	order := []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE,
		leaguerules.PK, leaguerules.OL, leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S}
	for i, o := range order {
		if o == p {
			return i
		}
	}
	return len(order)
}

// ordinal writes 1st, 2nd, 3rd, 4th, 11th, 21st.
func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// designationWords says how the NFL lists a player, for a sentence that starts "the NFL lists him ...".
func designationWords(d leaguerules.NFLStatus) string {
	switch d {
	case leaguerules.InjuredReserve:
		return "on injured reserve"
	case leaguerules.PUP:
		return "as physically unable to perform"
	case leaguerules.NFI:
		return "as non-football injured"
	case leaguerules.Out:
		return "as out"
	case leaguerules.Suspended:
		return "as suspended"
	}
	return "as " + string(d)
}
