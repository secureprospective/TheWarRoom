package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// DraftPickInput is an owner's choice of rookie for the pick their franchise has on the clock.
type DraftPickInput struct {
	League    string `json:"league"`
	Season    int    `json:"season"`
	Franchise string `json:"franchise"`
	Player    string `json:"player"`
	Request   string `json:"request"`
	Expected  int64  `json:"expected"`
}

func (i DraftPickInput) Validate() error {
	if !strings.HasPrefix(i.League, "native.") || i.League == "native." || strings.TrimSpace(i.League) != i.League ||
		i.Season < 1 || i.Season > 9999 || blank(i.Franchise) || blank(i.Player) || blank(i.Request) ||
		i.Expected < 1 {
		return fmt.Errorf("choose an app league, draft season, your franchise, a rookie and a request")
	}
	return nil
}

// PlanDraftPick judges a pick against the draft as it stands at an instant: it must be this franchise's
// turn, the player an unowned rookie of the season, and the contract must fit the roster and the cap. It
// returns the command to record, carrying the slot and the contract fixed now. Every refusal is in words.
func PlanDraftPick(in DraftPickInput, records []Record, facts nativecheck.Facts, at time.Time,
	local *time.Location, standings DraftStandings) (RawCommand, error) {
	if err := in.Validate(); err != nil {
		return RawCommand{}, Refused(err)
	}
	if len(records) == 0 || !leagueFranchise(records, in.Franchise) {
		return RawCommand{}, Refused(fmt.Errorf("choose a franchise in this app league"))
	}
	id, err := playerid.New(in.Player)
	if err != nil || id.String() != in.Player {
		return RawCommand{}, Refused(fmt.Errorf("choose a rookie from this season's player pool"))
	}
	basis, err := readDraft(records, in.Season, at, standings)
	if err != nil {
		return RawCommand{}, err
	}
	if basis.words != "" {
		return RawCommand{}, Refused(fmt.Errorf("the draft is not ready: %s", strings.TrimSuffix(basis.words, ".")))
	}
	if basis.closed != nil {
		return RawCommand{}, Refused(fmt.Errorf("the %d rookie draft is closed", in.Season))
	}
	if basis.draft.OnClock != nil && basis.draft.OnClock.Paused {
		return RawCommand{}, Refused(fmt.Errorf("the draft clock is paused; wait for the commissioner to resume it"))
	}
	assets, err := NativeAssetsReading(in.League, RecordsThrough(records, at))
	if err != nil {
		return RawCommand{}, err
	}
	preview, err := rookiedraft.PreviewSelection(basis.rules, basis.draft, in.Franchise, candidateFrom(facts, in),
		pickBasis(assets, basis, in, facts, local, records, at))
	if err != nil {
		return RawCommand{}, Refused(err)
	}
	clock := basis.draft.OnClock
	selection := RawDraftSelection{Franchise: in.Franchise, Player: in.Player, EntrySeason: in.Season,
		Round: clock.Round, Slot: clock.Slot, Original: clock.Original, Terms: preview.Terms}
	return RawCommand{League: in.League, Command: DraftPick, Expected: in.Expected, Request: in.Request,
		Args: RawArgs{Draft: &RawDraftArgs{Season: in.Season, Pick: &selection}}, Effective: "now",
		Visibility: "league"}, nil
}

func candidateFrom(facts nativecheck.Facts, in DraftPickInput) rookiedraft.Candidate {
	candidate := rookiedraft.Candidate{ID: in.Player}
	if f, ok := facts[in.Player]; ok && f.DraftYearHeld {
		candidate.EntrySeason = f.DraftYear
	}
	return candidate
}

func pickBasis(assets NativeAssets, basis draftBasis, in DraftPickInput, facts nativecheck.Facts,
	local *time.Location, records []Record, at time.Time) rookiedraft.SelectionBasis {
	out := rookiedraft.SelectionBasis{Exceptions: ScopedDecisionExceptions(records, ExceptionScope{Franchise: in.Franchise, Player: in.Player}, at), Season: in.Season, Facts: facts, Local: local}
	if !assets.Seeded {
		return out
	}
	for _, p := range assets.Snapshot.Players {
		out.Players = append(out.Players, rookiedraft.PlayerContract{ID: p.ID, Franchise: p.Franchise,
			Salary: p.Salary, Status: p.Status})
	}
	for _, c := range assets.Snapshot.Adjustments {
		if c.Franchise == in.Franchise && (c.Season == 0 || c.Season == basis.rules.Season) {
			out.Charges = append(out.Charges, c.Amount)
		}
	}
	return out
}

// RetriedDraftPick returns the receipt of a pick already recorded under the same request, so a retried
// request lands once. A request key reused for anything else is refused.
func RetriedDraftPick(in DraftPickInput, principal string, records []Record) (Receipt, bool, error) {
	for _, r := range records {
		if r.Command.Request != in.Request {
			continue
		}
		d := r.Command.Args.Draft
		if r.Command.Command != DraftPick || !SamePrincipal(records, r.Principal, principal) || d.Season != in.Season ||
			d.Pick.Franchise != in.Franchise || d.Pick.Player != in.Player {
			return Receipt{}, false, Refused(fmt.Errorf("this request was already used for a different change"))
		}
		return r.Receipt, true, nil
	}
	return Receipt{}, false, nil
}
