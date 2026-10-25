package commissioner

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// CheckTokenFranchise checks argument references against the delegated franchise.
// Command and mode ceilings and current domain permission are checked separately.
func CheckTokenFranchise(access Access, franchise string) error {
	targets := append(previewTargets(access.Raw), access.Franchise)
	if access.DraftPick != nil {
		targets = append(targets, access.DraftPick.Franchise)
	}
	extra, err := tokenMarketTargets(access.Raw, access.Records)
	if err != nil {
		return err
	}
	targets = append(targets, extra...)
	extra, err = tokenDraftTargets(access.Raw, access.Records)
	if err != nil {
		return err
	}
	targets = append(targets, extra...)
	for _, target := range targets {
		if target != "" && target != franchise {
			return fmt.Errorf("outside the delegated franchise")
		}
	}
	if err := tokenReferencedTrade(access, franchise); err != nil {
		return err
	}
	if err := tokenReferencedTransaction(access, franchise); err != nil {
		return err
	}
	return checkTokenPlayers(access, franchise)
}

func checkTokenPlayers(access Access, franchise string) error {
	players, err := tokenPlayers(access.Raw.Args)
	if err != nil {
		return err
	}
	for _, player := range append(players, access.Player) {
		if player == "" {
			continue
		}
		if len(access.Records) == 0 {
			return fmt.Errorf("player scope is unavailable")
		}
		owner, err := playerFranchise(access.Records, player)
		if err != nil {
			return err
		}
		if owner != "" && owner != franchise {
			return fmt.Errorf("player is outside the delegated franchise")
		}
	}
	return nil
}

func tokenPlayers(a RawArgs) ([]string, error) {
	players := []string{}
	if a.Override != nil {
		players = append(players, a.Override.Player)
	}
	if a.Score != nil {
		players = append(players, a.Score.Player)
	}
	if a.Stat != nil {
		players = append(players, a.Stat.Player)
		if a.Stat.Item != "" {
			id, err := ParseStatItemID(a.Stat.Item)
			if err != nil {
				return nil, err
			}
			players = append(players, id.Player)
		}
	}
	if a.Correction != nil {
		players = append(players, a.Correction.Player)
		if a.Correction.Scope != nil {
			players = append(players, a.Correction.Scope.Player)
		}
	}
	if a.Lineup != nil {
		players = append(players, a.Lineup.Starters...)
	}
	return players, nil
}

func tokenMarketTargets(raw RawCommand, records []Record) ([]string, error) {
	a := raw.Args
	targets := []string{}
	if a.Bid != nil {
		targets = append(targets, a.Bid.Franchise)
		if a.Bid.Bid != "" {
			r, err := tokenReferencedRecord(records, a.Bid.Bid)
			if err != nil {
				return nil, err
			}
			if r.Command.Args.Bid == nil {
				return nil, fmt.Errorf("bid is unavailable")
			}
			targets = append(targets, r.Command.Args.Bid.Franchise)
		}
	}
	if a.Waiver != nil {
		targets = append(targets, a.Waiver.Franchise)
		if a.Waiver.Claim != "" {
			r, err := tokenReferencedRecord(records, a.Waiver.Claim)
			if err != nil {
				return nil, err
			}
			if r.Command.Args.Waiver == nil {
				return nil, fmt.Errorf("claim is unavailable")
			}
			targets = append(targets, r.Command.Args.Waiver.Franchise)
		}
	}
	if a.Trade != nil {
		targets = append(targets, a.Trade.Franchise)
	}
	if a.Bracket != nil {
		targets = append(targets, a.Bracket.Team)
	}
	if a.Link != nil {
		targets = append(targets, a.Link.Franchise)
	}
	if a.Schedule != nil {
		for _, game := range a.Schedule.Games {
			targets = append(targets, game.Home, game.Away)
		}
	}
	return targets, nil
}

func tokenReferencedRecord(records []Record, entry string) (Record, error) {
	for _, r := range records {
		if r.Receipt.Entry == entry {
			return r, nil
		}
	}
	return Record{}, fmt.Errorf("referenced league change is unavailable")
}

func tokenReferencedTransaction(access Access, franchise string) error {
	c := access.Raw.Args.Correction
	if c == nil {
		return nil
	}
	entry := c.Entry
	if c.Scope != nil && c.Scope.Transaction != "" {
		entry = c.Scope.Transaction
	}
	if entry == "" {
		return nil
	}
	for i, r := range access.Records {
		if r.Receipt.Entry != entry {
			continue
		}
		return CheckTokenFranchise(Access{Raw: r.Command, Records: access.Records[:i]}, franchise)
	}
	return fmt.Errorf("referenced transaction is unavailable")
}

func tokenDraftTargets(raw RawCommand, records []Record) ([]string, error) {
	d := raw.Args.Draft
	if c := raw.Args.Correction; c != nil && c.Pick != nil {
		return tokenPickOwner(raw.League, records, c.Pick.Season, c.Pick.Round, c.Pick.Original)
	}
	if d == nil {
		return nil, nil
	}
	if d.Pick != nil {
		return []string{d.Pick.Franchise}, nil
	}
	if d.Act == nil || d.Act.Original == "" {
		return nil, nil
	}
	targets, err := tokenPickOwner(raw.League, records, d.Season, d.Act.Round, d.Act.Original)
	return append(targets, d.Act.Franchise), err
}

func tokenPickOwner(league string, records []Record, season, round int, original string) ([]string, error) {
	assets, err := NativeAssetsReading(league, records)
	if err != nil {
		return nil, err
	}
	if assets.Snapshot != nil {
		for _, p := range assets.Snapshot.Picks {
			if p.Season == season && p.Round == round && p.Original == original {
				return []string{p.Owner}, nil
			}
		}
	}
	return nil, fmt.Errorf("referenced draft pick is unavailable")
}

func tokenReferencedTrade(access Access, franchise string) error {
	trade := access.Raw.Args.Trade
	if trade == nil || trade.ID == "" {
		return nil
	}
	_, book, err := nativeAssetsState(access.Raw.League, access.Records)
	if err != nil {
		return err
	}
	for _, t := range book.Trades {
		if t.ID == trade.ID && slices.Contains(t.Parties, franchise) {
			return nil
		}
	}
	return fmt.Errorf("trade is outside the delegated franchise")
}

type tokenAuditKey struct{}

// WithTokenAudit is supplied only by the authenticated transport, never request JSON.
func WithTokenAudit(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tokenAuditKey{}, id)
}

func TokenAuditID(ctx context.Context) string {
	id, _ := ctx.Value(tokenAuditKey{}).(string)
	return id
}

// TokenScopePermission requires a current named commissioner and a real franchise.
func TokenScopePermission(records []Record, principal, franchise string) error {
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	if !state.HasRole(governance.Principal(PrincipalFor(records, principal)), governance.Commissioner) {
		return fmt.Errorf("agent tokens act for the current commissioner only")
	}
	if _, ok := state.Franchise(governance.FranchiseID(franchise)); !ok {
		return fmt.Errorf("choose a franchise in this league")
	}
	return nil
}
