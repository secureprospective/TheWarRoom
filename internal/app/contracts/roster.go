package contracts

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Acquisition is the twin of the root RawAcquisition: the same fields in the same order.
type Acquisition struct {
	Mine   string `json:"mine"`
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Cut    string `json:"cut"`
}

func (r Acquisition) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("acquisition scope: %w", err)
	}
	if r.Mine == "" || (r.Player != "" && r.Bid != "") {
		return fmt.Errorf("acquisition: choose franchise and one source")
	}
	for _, id := range []string{r.Player, r.Cut} {
		if id == "" {
			continue
		}
		if _, err := playerid.New(id); err != nil {
			return fmt.Errorf("acquisition player: %w", err)
		}
	}
	return nil
}

// Name is a player's name in the held directory.
func Name(snap snapshot.Snapshot, id string) string {
	for _, p := range snap.Players.Value {
		if p.ID.String() == id && p.Name != "" {
			return p.Name
		}
	}
	return "Name unavailable"
}

// Selection is the starters an owner picked for a lineup, as raw ids.
type Selection struct{ Starters []string }

// Validate reads each starter's id, refusing a malformed or repeated one.
func (r Selection) Validate() ([]playerid.PlayerID, error) {
	ids := make([]playerid.PlayerID, 0, len(r.Starters))
	for _, raw := range r.Starters {
		id, err := playerid.New(raw)
		if err != nil {
			return nil, fmt.Errorf("lineup selection: player: %w", err)
		}
		if slices.Contains(ids, id) {
			return nil, fmt.Errorf("lineup selection: duplicate player %s", id)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// UnreadableRules is a lineup check that could not run because the league's rules could not be read.
func UnreadableRules(err error) lineup.Result {
	return lineup.Result{Problems: []lineup.Problem{{
		Subject: "Rules", Kind: lineup.KindUnknown,
		Message: "The league's lineup rules could not be read: " + err.Error(),
	}}}
}

// Competing says whether two plans change the same thing: one week's lineup, or one trade.
func Competing(left, right envelope.Spec) bool {
	if left.Intent != right.Intent {
		return false
	}
	switch left.Intent {
	case "lineup.set":
		return left.Expected.Lineup.Week == right.Expected.Lineup.Week
	case "trade.accept", "trade.reject", "trade.revoke":
		return left.Expected.Trade.TradeID == right.Expected.Trade.TradeID
	default:
		return false
	}
}

// LineupPlayer is a player in the lineup; it is main.LineupPlayer.
type LineupPlayer struct {
	ID       string          `json:"id"`
	Position domain.Position `json:"position,omitempty" ts_type:"string"`
}

// Layout is a franchise's saved starters and the rest of its active roster as the bench, each in the
// rules' position order, then by name and id. found is false when MFL holds no saved lineup for it.
func Layout(snap snapshot.Snapshot, saved leaguefeed.Lineups, franchise string, rules lineup.Rules) (starters, bench []LineupPlayer, found bool) {
	starters, bench = []LineupPlayer{}, []LineupPlayer{}
	players := make(map[string]snapshot.Player, len(snap.Players.Value))
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	starting := make(map[string]bool)
	for _, held := range saved.Franchises {
		if held.Franchise != franchise {
			continue
		}
		found = true
		for _, id := range held.Starters {
			starting[id.String()] = true
			starters = append(starters, LineupPlayer{ID: id.String(), Position: players[id.String()].Position})
		}
		break
	}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != franchise {
			continue
		}
		for _, p := range roster.Players {
			id := p.ID.String()
			if p.RosterStatus == domain.RosterActive && !starting[id] {
				bench = append(bench, LineupPlayer{ID: id, Position: players[id].Position})
			}
		}
	}
	// MFL lists starters in a different order on every fetch, so its order is never used: the
	// same lineup must lay out the same way each time.
	byPosition := func(a, b LineupPlayer) int {
		if order := cmp.Compare(rules.Order(a.Position), rules.Order(b.Position)); order != 0 {
			return order
		}
		if name := cmp.Compare(players[a.ID].Name, players[b.ID].Name); name != 0 {
			return name
		}
		return cmp.Compare(a.ID, b.ID)
	}
	slices.SortFunc(starters, byPosition)
	slices.SortFunc(bench, byPosition)
	return starters, bench, found
}

func ParseScorePeriod(body []byte, period string) ([]ingestion.RawPlayerScore, error) {
	rows, err := ingestion.ParsePlayerScores(body)
	if err != nil {
		return nil, fmt.Errorf("player scores: parse: %w", err)
	}
	out := []ingestion.RawPlayerScore{}
	for _, r := range rows {
		if r.Week == period {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("player scores: period %s not held", period)
	}
	return out, nil
}

func RulesFreshness(checked int64) domain.Freshness {
	if checked == 0 {
		return domain.Freshness{State: domain.FreshStale, Note: "stored league settings; not yet checked with MFL this run"}
	}
	return report.LiveFreshness(time.Unix(checked, 0))
}
