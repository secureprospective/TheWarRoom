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

// RawNativeSnapshot is captured by Go from held inputs, never supplied by an IPC caller.
type RawNativeSnapshot struct {
	StandingsOrder []string          `json:"standingsOrder,omitempty"`
	Season         int               `json:"season"`
	RosterAt       time.Time         `json:"rosterAt"`
	DraftAt        time.Time         `json:"draftAt"`
	FutureAt       time.Time         `json:"futureAt"`
	Franchises     []NativeFranchise `json:"franchises"`
	Players        []NativePlayer    `json:"players"`
	Picks          []NativePick      `json:"picks"`
	Adjustments    []NativeCharge    `json:"adjustments"`
}

type NativeFranchise struct {
	ID           string `json:"id"`
	HeldID       string `json:"heldId"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Division     string `json:"division"`
}

type NativePlayer struct {
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
	Season   int    `json:"season"`
	Round    int    `json:"round"`
	Slot     int    `json:"slot"`
	Original string `json:"original"`
	Owner    string `json:"owner"`
	Player   string `json:"player"`
	Skipped  bool   `json:"skipped"`
}

type NativeCharge struct {
	Season      int          `json:"season,omitempty"`
	ID          string       `json:"id"`
	Franchise   string       `json:"franchise"`
	Amount      domain.Money `json:"amountCents"`
	Description string       `json:"description"`
}

func (r RawNativeSnapshot) Validate() error {
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
	if err := r.validatePicks(franchises); err != nil {
		return err
	}
	return r.validateCharges(franchises)
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
		if c.ID == "" || charges[c.ID] || !franchises[c.Franchise] || c.Season < 0 || c.Season > 9999 {
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
			p.ExpirationYear < 0 || p.ExpirationYear > 9999 {
			return fmt.Errorf("a held player has repeated or invalid contract details")
		}
		if !slices.Contains([]domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi}, p.Status) {
			return fmt.Errorf("a held player has an unknown roster status")
		}
		players[p.ID] = true
	}
	return nil
}

func (r RawNativeSnapshot) validatePicks(franchises map[string]bool) error {
	picks := map[string]bool{}
	for _, p := range r.Picks {
		if err := p.validate(r.Season, franchises); err != nil {
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
