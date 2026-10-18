package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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
	assets, err := NativeAssetsReading(in.League, recordsThrough(records, at))
	if err != nil {
		return RawCommand{}, err
	}
	preview, err := rookiedraft.PreviewSelection(basis.rules, basis.draft, in.Franchise, candidateFrom(facts, in),
		pickBasis(assets, basis, in, facts, local))
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
	local *time.Location) rookiedraft.SelectionBasis {
	out := rookiedraft.SelectionBasis{Season: in.Season, Facts: facts, Local: local}
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

// applyDraftPick gives a drafted rookie his rookie contract, and marks the pick he was taken with.
func applyDraftPick(seed *RawNativeSnapshot, record Record) {
	d := record.Command.Args.Draft
	p := d.Pick
	signRookie(seed, d.Season, p.Player, p.Franchise, p.Round, p.Slot, p.Terms)
	markPick(seed, d.Season, p.Round, p.Original, p.Slot, p.Player)
}

// signRookie puts a drafted rookie on a roster under the contract his slot fixed.
func signRookie(seed *RawNativeSnapshot, season int, player, franchise string, round, slot int, terms rookiedraft.Terms) {
	status := domain.CStatusUFA
	if terms.RFAAfter {
		status = domain.CStatusRFA
	}
	seed.Players = append(seed.Players, NativePlayer{ID: player, Franchise: franchise, RookieRound: round,
		Status: domain.RosterActive, Salary: terms.Salary, ExpirationYear: season + terms.Years - 1,
		ContractStatus: status, ContractInfo: fmt.Sprintf("Rookie contract, round %d pick %d of the %d draft",
			round, slot, season)})
}

// markPick records which rookie a scheduled pick was used on, and where it fell.
func markPick(seed *RawNativeSnapshot, season, round int, original string, slot int, player string) {
	for i, pick := range seed.Picks {
		if pick.Season == season && pick.Round == round && pick.Original == original {
			seed.Picks[i].Slot, seed.Picks[i].Player = slot, player
		}
	}
}

// applyDraftTray gives the rookie a commissioner chose, or added, his contract; a chosen pick is marked used.
func applyDraftTray(seed *RawNativeSnapshot, record Record) {
	d, o := record.Command.Args.Draft, record.Command.Args.DraftOutcome
	if o == nil || o.Terms == nil {
		return
	}
	signRookie(seed, d.Season, d.Act.Player, o.Franchise, d.Act.Round, o.Slot, *o.Terms)
	if record.Command.Command == DraftAssign && d.Act.Extra == 0 {
		markPick(seed, d.Season, d.Act.Round, d.Act.Original, o.Slot, d.Act.Player)
	}
}

// applyDraftChange swaps the rookie a pick holds: the new one takes the contract, and the old one leaves the
// roster to be undrafted.
func applyDraftChange(seed *RawNativeSnapshot, record Record) {
	d := record.Command.Args.Draft
	for i := range seed.Players {
		if seed.Players[i].ID == d.Act.Replaces {
			seed.Players[i].ID = d.Act.Player
		}
	}
	for i := range seed.Picks {
		if seed.Picks[i].Player == d.Act.Replaces {
			seed.Picks[i].Player = d.Act.Player
		}
	}
}

// RetriedDraftPick returns the receipt of a pick already recorded under the same request, so a retried
// request lands once. A request key reused for anything else is refused.
func RetriedDraftPick(in DraftPickInput, principal string, records []Record) (Receipt, bool, error) {
	for _, r := range records {
		if r.Command.Request != in.Request {
			continue
		}
		d := r.Command.Args.Draft
		if r.Command.Command != DraftPick || r.Principal != principal || d.Season != in.Season ||
			d.Pick.Franchise != in.Franchise || d.Pick.Player != in.Player {
			return Receipt{}, false, Refused(fmt.Errorf("this request was already used for a different change"))
		}
		return r.Receipt, true, nil
	}
	return Receipt{}, false, nil
}
