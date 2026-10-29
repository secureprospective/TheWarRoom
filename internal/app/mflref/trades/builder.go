package trades

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (r Input) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("trade builder scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "preview", "save", "reject", "revoke", "handoff", "remove"}, r.Action) {
		return fmt.Errorf("trade builder: franchise and known action required")
	}
	if slices.Contains([]string{"reject", "revoke", "handoff", "remove"}, r.Action) && r.ID == "" {
		return fmt.Errorf("trade builder: offer or plan required")
	}
	if r.Action != "preview" && r.Action != "save" {
		return nil
	}
	if err := (snapshot.RawContractScope{Mine: r.Other, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("trade builder other franchise: %w", err)
	}
	if r.Other == "" || r.Other == r.Mine || len(r.Give) == 0 || len(r.Get) == 0 || len(r.Give)+len(r.Get) > 256 {
		return fmt.Errorf("trade builder: distinct franchises and assets on both sides required")
	}
	return validateTradeAssets(r.Give, r.Get)
}

func validateTradeAssets(give, get []string) error {

	seen := map[string]bool{}
	for _, token := range append(slices.Clone(give), get...) {
		assets, err := leaguefeed.ParseAssets(token)
		if err != nil {
			return fmt.Errorf("trade builder asset: %w", err)
		}
		if len(assets) != 1 || assets[0].BlindBidCents != nil || assets[0].CurrentPick != nil {
			return fmt.Errorf("trade builder: player or held future pick required")
		}
		key := leaguefeed.AssetToken(assets[0])
		if seen[key] {
			return fmt.Errorf("trade builder: duplicate asset")
		}
		seen[key] = true
	}
	return nil
}
func PlayerAsset(p state.MirrorPlayer, player snapshot.Player, name string, value Value, season int) Asset {
	if value.ID == "" {
		value.Now = Cell{Text: "No score", Sub: "This season's points per game"}
		value.Dynasty = Cell{Text: "Not valued"}
	}
	return Asset{Token: p.MFLID, Name: name, Position: player.Position, Team: player.Team,
		Contract: fmt.Sprintf("%s · %d years", snapshot.ContractAmountOf(p.Salary).Text, max(p.ContractYear-season+1, 0)),
		Now:      &value.Now, Dynasty: &value.Dynasty}
}
func RichRoster(roster []Asset, catalog []Team) []Asset {
	values := map[string]Asset{}
	for _, team := range catalog {
		for _, asset := range team.Assets {
			values[asset.Token] = asset
		}
	}
	for i, player := range roster {
		if asset, ok := values[player.Token]; ok {
			asset.Name = player.Name
			roster[i] = asset
		}
	}
	return roster
}

func Teams(snap snapshot.Snapshot, m state.MirrorSnapshot, picks []draftreports.FuturePick,
	values map[string]Value, pickValue func(int, int, string) *Cell) []Team {
	players := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	out := []Team{}
	for _, f := range snap.Franchises.Value {
		team := Team{ID: f.ID, Name: f.Name, Assets: []Asset{}}
		for _, p := range m.Players {
			if p.FranchiseID == f.ID {
				name := players[p.MFLID].Name
				if name == "" {
					name = "Name unavailable"
				}
				team.Assets = append(team.Assets, PlayerAsset(p, players[p.MFLID], name, values[p.MFLID], m.Season))
			}
		}
		for _, pick := range picks {
			if pick.Owner != f.ID || pick.Year > m.Season+2 {
				continue
			}
			team.Assets = append(team.Assets, Asset{Token: fmt.Sprintf("FP_%s_%d_%d", pick.Original, pick.Year, pick.Round),
				Name:    FuturePickName(pick.Year, pick.Round, pick.Original, snap),
				Dynasty: pickValue(pick.Year, pick.Round, pick.Original)})
		}
		slices.SortFunc(team.Assets, func(a, b Asset) int { return strings.Compare(a.Name, b.Name) })
		out = append(out, team)
	}
	return out
}

func PickOwnershipReason(give, get []string, read func() (time.Time, string)) string {
	if !slices.ContainsFunc(append(slices.Clone(give), get...), func(token string) bool { return strings.HasPrefix(token, "FP_") }) {
		return ""
	}
	fetchedAt, refreshError := read()
	if fetchedAt.IsZero() {
		return "Draft-pick ownership unavailable; refresh before handing off."
	}
	if refreshError != "" {
		return "Draft-pick ownership unavailable: " + refreshError
	}
	return ""
}

func ClockReasons(read func() (snapshot.Sourced[leagueclock.Reading], error), now func() time.Time, zone *time.Location) []string {
	clock, err := read()
	if err != nil {
		return []string{"Trade deadline cannot be checked: " + err.Error()}
	}
	return DeadlineReasons(now(), clock.Value.Windows, zone)
}

func ReadState(franchiseID string, selection func(string) (snapshot.Snapshot, error), pending func() snapshot.Sourced[[]leaguefeed.PendingTrade]) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error) {
	snap, err := selection(franchiseID)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Sourced[[]leaguefeed.PendingTrade]{}, fmt.Errorf("trade state: %w", err)
	}
	return snap, pending(), nil
}

func BuilderTeams(snap snapshot.Snapshot, mirror state.MirrorSnapshot, picks []draftreports.FuturePick, board []Value, boardErr error, pickValue func(int, int, string) *Cell) []TradeBuilderTeam {
	values := map[string]Value{}
	if boardErr == nil {
		for _, v := range board {
			values[v.ID] = v
		}
	}
	return WireTeams(Teams(snap, mirror, picks, values, pickValue))
}
