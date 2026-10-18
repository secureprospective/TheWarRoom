package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const NativeSeed = "commish.league.seed_held"

// NativeChampion is the franchise that won a finished season.
type NativeChampion struct {
	Season    int    `json:"season"`
	Franchise string `json:"franchise"`
}

// RawNativeSnapshot is captured by Go from held inputs, never supplied by an IPC caller.
type RawNativeSnapshot struct {
	StandingsOrder []string `json:"standingsOrder,omitempty"`
	Season         int      `json:"season"`
	// Champion is the franchise recorded as the champion of Season, once decided; the rollover archives it.
	Champion string `json:"champion,omitempty"`
	// Champions archives the winner of each finished season.
	Champions []NativeChampion `json:"champions,omitempty"`
	// Conferences names the conference each division sits in, as the held league sets them out.
	Conferences map[string]string `json:"conferences,omitempty"`
	RosterAt    time.Time         `json:"rosterAt"`
	DraftAt     time.Time         `json:"draftAt"`
	FutureAt    time.Time         `json:"futureAt"`
	Franchises  []NativeFranchise `json:"franchises"`
	Players     []NativePlayer    `json:"players"`
	Picks       []NativePick      `json:"picks"`
	Adjustments []NativeCharge    `json:"adjustments"`
}

type NativeFranchise struct {
	ID           string `json:"id"`
	HeldID       string `json:"heldId"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Division     string `json:"division"`
}

type NativePlayer struct {
	AcquiredVia string `json:"acquiredVia,omitempty"`
	Correction  string `json:"correction,omitempty"`
	// Name is the name the app held for the player when the snapshot was captured.
	Name string `json:"name,omitempty"`
	// RookieRound is the draft round of the player's rookie contract, where it is recorded; 0 is unknown.
	RookieRound int `json:"rookieRound,omitempty"`
	// Pending marks a player whose contract has ended and whose decision is still to be made.
	Pending        bool                  `json:"pending,omitempty"`
	Restructured   bool                  `json:"restructured,omitempty"`
	ID             string                `json:"id"`
	Franchise      string                `json:"franchise"`
	Status         domain.RosterStatus   `json:"status"`
	Salary         domain.Money          `json:"salaryCents"`
	ExpirationYear int                   `json:"expirationYear"`
	ContractStatus domain.ContractStatus `json:"contractStatus"`
	ContractInfo   string                `json:"contractInfo"`
}

type NativePick struct {
	Compensatory bool   `json:"compensatory,omitempty"`
	Correction   string `json:"correction,omitempty"`
	Season       int    `json:"season"`
	Round        int    `json:"round"`
	Slot         int    `json:"slot"`
	Original     string `json:"original"`
	Owner        string `json:"owner"`
	Player       string `json:"player"`
	Skipped      bool   `json:"skipped"`
}

type NativeCharge struct {
	// Kind is "dead_money" for dead money, which carries into the next season; empty otherwise.
	Kind        string       `json:"kind,omitempty"`
	Season      int          `json:"season,omitempty"`
	ID          string       `json:"id"`
	Franchise   string       `json:"franchise"`
	Amount      domain.Money `json:"amountCents"`
	Description string       `json:"description"`
}

// Validate checks a snapshot captured from held data, as it is recorded.
func (r RawNativeSnapshot) Validate() error { return r.validate(false) }

// validate checks a snapshot; rolled is true for assets a season rollover has carried past the season they
// were captured in, whose picks keep their history and whose new season's picks wait for the draft.
func (r RawNativeSnapshot) validate(rolled bool) error {
	if r.Season < 1 || r.Season > 9999 || r.RosterAt.IsZero() || r.DraftAt.IsZero() || r.FutureAt.IsZero() ||
		len(r.Franchises) != legacyTeams || len(r.Players) == 0 {
		return fmt.Errorf("hold the Legacy NFL rosters, franchises and both draft reports before seeding")
	}
	franchises, err := r.franchiseIndex()
	if err != nil {
		return err
	}
	if len(r.StandingsOrder) != 0 {
		if err := completeWaiverOrder(r.StandingsOrder, &r); err != nil {
			return err
		}
	}
	if err := r.validatePlayers(franchises); err != nil {
		return err
	}
	if err := r.validatePicks(franchises, rolled); err != nil {
		return err
	}
	if err := r.validateArchive(franchises); err != nil {
		return err
	}
	return r.validateCharges(franchises)
}

// validateArchive checks what the snapshot carries of the league's past and its conference layout.
func (r RawNativeSnapshot) validateArchive(franchises map[string]bool) error {
	if err := r.validateChampions(franchises); err != nil {
		return err
	}
	divisions := map[string]bool{}
	for _, f := range r.Franchises {
		divisions[f.Division] = true
	}
	for division, conference := range r.Conferences {
		if !divisions[division] || strings.TrimSpace(conference) == "" {
			return fmt.Errorf("the held conferences name a division no franchise sits in, or no conference")
		}
	}
	return nil
}

func (r RawNativeSnapshot) validateChampions(franchises map[string]bool) error {
	if r.Champion != "" && !franchises[r.Champion] {
		return fmt.Errorf("the champion must be a recorded franchise")
	}
	seen := map[int]bool{}
	for _, champion := range r.Champions {
		if champion.Season < 1 || champion.Season >= r.Season || seen[champion.Season] ||
			!franchises[champion.Franchise] {
			return fmt.Errorf("the champion archive has invalid seasons or franchises")
		}
		seen[champion.Season] = true
	}
	return nil
}

func (r RawNativeSnapshot) franchiseIndex() (map[string]bool, error) {
	franchises, held := map[string]bool{}, map[string]bool{}
	for _, f := range r.Franchises {
		if strings.TrimSpace(f.HeldID) == "" || franchises[f.ID] || held[f.HeldID] || strings.TrimSpace(f.Name) == "" {
			return nil, fmt.Errorf("the held franchise directory is incomplete or repeated")
		}
		franchises[f.ID], held[f.HeldID] = true, true
	}
	return franchises, nil
}

func (r RawNativeSnapshot) validateCharges(franchises map[string]bool) error {
	charges := map[string]bool{}
	for _, c := range r.Adjustments {
		if (c.Kind != "" && c.Kind != deadMoney) || c.ID == "" || charges[c.ID] || !franchises[c.Franchise] ||
			c.Season < 0 || c.Season > lastSeason {
			return fmt.Errorf("a held salary adjustment has no unique identity or franchise")
		}
		charges[c.ID] = true
	}
	return nil
}

func (r RawNativeSnapshot) validatePlayers(franchises map[string]bool) error {
	players := map[string]bool{}
	for _, p := range r.Players {
		id, err := playerid.New(p.ID)
		if err != nil {
			return fmt.Errorf("held roster player: %w", err)
		}
		if id.String() != p.ID || players[p.ID] || !franchises[p.Franchise] || p.Salary < 0 ||
			p.RookieRound < 0 || p.RookieRound > 99 || p.ExpirationYear < 0 || p.ExpirationYear > 9999 {
			return fmt.Errorf("a held player has repeated or invalid contract details")
		}
		if !slices.Contains([]domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi}, p.Status) {
			return fmt.Errorf("a held player has an unknown roster status")
		}
		players[p.ID] = true
	}
	return nil
}

func (r RawNativeSnapshot) validatePicks(franchises map[string]bool, rolled bool) error {
	picks := map[string]bool{}
	for _, p := range r.Picks {
		check := func() error { return p.validate(r.Season, franchises) }
		if rolled {
			check = func() error { return p.validateRolled(franchises) }
		}
		if err := check(); err != nil {
			return err
		}
		identity := fmt.Sprintf("%d/%d/%d/%s", p.Season, p.Round, p.Slot, p.Original)
		if picks[identity] {
			return fmt.Errorf("a held draft pick is listed twice")
		}
		if p.Player != "" {
			id, err := playerid.New(p.Player)
			if err != nil || id.String() != p.Player {
				return fmt.Errorf("a held pick has an invalid selected player")
			}
		}
		picks[identity] = true
	}
	return nil
}

func (p NativePick) validate(season int, franchises map[string]bool) error {
	if p.Season < season || p.Season > 9999 || p.Round < 1 || p.Round > 100 || !franchises[p.Owner] {
		return fmt.Errorf("a held pick has invalid ownership, season or round")
	}
	if p.Season == season {
		return p.validateSelection()
	}
	if p.Slot != 0 || !franchises[p.Original] || p.Player != "" || p.Skipped {
		return fmt.Errorf("a held future pick has invalid original ownership")
	}
	return nil
}

// validateRolled accepts a pick of any season: a finished draft's picks stay as history, and the picks of
// the season that has begun still show their original owner until the draft orders them.
func (p NativePick) validateRolled(franchises map[string]bool) error {
	if p.Season < 1 || p.Season > lastSeason || p.Round < 1 || p.Round > 100 || p.Slot < 0 || p.Slot > 1000 ||
		!franchises[p.Owner] || (p.Original != "" && !franchises[p.Original]) || (p.Skipped && p.Player != "") {
		return fmt.Errorf("a recorded pick has invalid ownership, season or round")
	}
	return nil
}

func (p NativePick) validateSelection() error {
	if p.Slot < 1 || p.Slot > 1000 || p.Original != "" || (p.Skipped && p.Player != "") {
		return fmt.Errorf("a held current pick has invalid selection details")
	}
	return nil
}

func (a RawArgs) validateNativeSeed() error {
	if a.Native == nil || a.Name != "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("seed from the held league snapshot only")
	}
	return a.Native.Validate()
}

// CheckNativeAppend is run under the ledger append lock; a second request can never reseed.
func CheckNativeAppend(raw RawCommand, records []Record) error {
	if raw.Command == Revert {
		for _, r := range records {
			if r.Receipt.Entry == raw.Args.Entry && r.Command.Command == NativeSeed {
				return Refused(fmt.Errorf("the starting snapshot is permanent; correct individual assets instead"))
			}
		}
	}
	if raw.Command != NativeSeed {
		return nil
	}
	if len(records) == 0 || records[0].Command.Args.Preset != PresetLegacy {
		return Refused(fmt.Errorf("only a Legacy NFL league can be seeded from held data"))
	}
	if season := records[0].Command.Args.Season; season != 0 && season != raw.Args.Native.Season {
		return Refused(fmt.Errorf("the starting snapshot must be for season %d, the season this league starts in",
			season))
	}
	if moved, _ := explicitRolloverInstants(records); len(moved) > 0 {
		return Refused(fmt.Errorf("record the starting assets before moving the league to a new season"))
	}
	for _, r := range records {
		if r.Command.Command == NativeSeed {
			return Refused(fmt.Errorf("this league already has its starting snapshot"))
		}
	}
	ids := records[0].Command.Args.Franchises
	for _, f := range raw.Args.Native.Franchises {
		if !slices.Contains(ids, f.ID) {
			return Refused(fmt.Errorf("the snapshot must use this app league's franchises"))
		}
	}
	return nil
}
