package snapshot

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type ContractAmount struct {
	Cents domain.Money `json:"cents"`
	Text  string       `json:"text"`
}

type ContractYear struct {
	Season   int            `json:"season"`
	Cost     ContractAmount `json:"cost"`
	Inferred bool           `json:"inferred"`
	Tag      string         `json:"tag"`
	Reason   string         `json:"reason"`
}

type ContractRow struct {
	ID          string         `json:"id"`
	FranchiseID string         `json:"franchiseId"`
	Team        string         `json:"team"`
	Name        string         `json:"name"`
	Position    string         `json:"position"`
	Salary      ContractAmount `json:"salary"`
	Expiration  int            `json:"expiration"`
	Status      string         `json:"status"`
	Info        string         `json:"info"`
}

type ContractTable struct {
	DirectorySource string         `json:"directorySource"`
	DirectoryAsOf   string         `json:"directoryAsOf"`
	Rows            []ContractRow  `json:"rows"`
	Teams           []ContractTeam `json:"teams"`
	Positions       []string       `json:"positions"`
}

type ContractTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlayerContract struct {
	Player ContractRow    `json:"player"`
	Years  []ContractYear `json:"years"`
}

type CapSummary struct {
	Team      string          `json:"team"`
	Season    int             `json:"season"`
	Cap       *ContractAmount `json:"cap"`
	Used      ContractAmount  `json:"used"`
	Room      *ContractAmount `json:"room"`
	CapSource string          `json:"capSource"`
	Future    []ContractYear  `json:"future"`
}

type SalaryAdjustment struct {
	ID          string         `json:"id"`
	Team        string         `json:"team"`
	FranchiseID string         `json:"franchiseId"`
	Amount      ContractAmount `json:"amount"`
	Description string         `json:"description"`
}

type SalaryAdjustments struct {
	Rows []SalaryAdjustment `json:"rows"`
}

type RawContractScope struct {
	Mine     string `json:"mine"`
	Team     string `json:"team"`
	Position string `json:"position"`
	Sort     string `json:"sort"`
	Desc     bool   `json:"desc"`
}

func (r RawContractScope) Validate() error {
	for _, id := range []string{r.Mine, r.Team} {
		if id != "" && (len(id) != 4 || strings.IndexFunc(id, func(c rune) bool { return c < '0' || c > '9' }) >= 0) {
			return fmt.Errorf("contracts scope: invalid franchise %q", id)
		}
	}
	if !slices.Contains([]string{"", "QB", "RB", "WR", "TE", "K", "DT", "DE", "LB", "CB", "S"}, r.Position) {
		return fmt.Errorf("contracts scope: invalid position %q", r.Position)
	}
	if !slices.Contains([]string{"name", "team", "position", "salary", "expiration"}, r.Sort) {
		return fmt.Errorf("contracts scope: invalid sort %q", r.Sort)
	}
	return nil
}

func contractAmount(m domain.Money) ContractAmount {
	text := m.String()
	if m%100 == 0 {
		text = strings.TrimSuffix(text, ".00")
	}
	return ContractAmount{Cents: m, Text: text}
}

func ContractRows(m state.MirrorSnapshot, directory Snapshot, names map[string]string) []ContractRow {
	players := make(map[string]Player)
	for _, p := range directory.Players.Value {
		players[p.ID.String()] = p
	}
	rows := make([]ContractRow, 0, len(m.Players))
	for _, p := range m.Players {
		d := players[p.MFLID]
		name := d.Name
		if name == "" {
			name = "Player " + p.MFLID
		}
		rows = append(rows, ContractRow{ID: p.MFLID, FranchiseID: p.FranchiseID, Team: contractTeamName(names, p.FranchiseID), Name: name, Position: string(d.Position), Salary: contractAmount(p.Salary), Expiration: p.ContractYear, Status: string(p.ContractStatus), Info: p.ContractInfo})
	}
	return rows
}

func contractTeamName(names map[string]string, id string) string {
	if names[id] != "" {
		return names[id]
	}
	return "Franchise " + id
}

func ContractsTable(rows []ContractRow, names map[string]string, scope RawContractScope) ContractTable {
	out := ContractTable{Rows: []ContractRow{}, Teams: []ContractTeam{}, Positions: []string{}}
	teams := make(map[string]string)
	positions := make(map[string]bool)
	for id, name := range names {
		teams[id] = name
	}
	for _, row := range rows {
		teams[row.FranchiseID] = row.Team
		if row.Position != "" {
			positions[row.Position] = true
		}
		if (scope.Team == "" || row.FranchiseID == scope.Team) && (scope.Position == "" || row.Position == scope.Position) {
			out.Rows = append(out.Rows, row)
		}
	}
	for id, name := range teams {
		out.Teams = append(out.Teams, ContractTeam{ID: id, Name: name})
	}
	for pos := range positions {
		out.Positions = append(out.Positions, pos)
	}
	slices.Sort(out.Positions)
	slices.SortFunc(out.Teams, func(a, b ContractTeam) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(out.Rows, func(a, b ContractRow) int {
		if (a.FranchiseID == scope.Mine) != (b.FranchiseID == scope.Mine) {
			if a.FranchiseID == scope.Mine {
				return -1
			}
			return 1
		}
		order := compareContract(a, b, scope.Sort)
		if scope.Desc {
			order = -order
		}
		if order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

func compareContract(a, b ContractRow, sort string) int {
	switch sort {
	case "salary":
		return cmp.Compare(a.Salary.Cents, b.Salary.Cents)
	case "expiration":
		return cmp.Compare(a.Expiration, b.Expiration)
	case "position":
		return strings.Compare(a.Position, b.Position)
	case "team":
		return strings.Compare(a.Team, b.Team)
	default:
		return strings.Compare(a.Name, b.Name)
	}
}

func ContractYears(row ContractRow, season int) []ContractYear {
	last := max(season, row.Expiration)
	years := make([]ContractYear, 0, last-season+1)
	for year := season; year <= last; year++ {
		inferred := year > season
		tag, reason := "Observed", "MFL annual salary for the mirror season"
		if inferred {
			tag, reason = "Inferred", "Flat-filled from MFL annual salary and expiration; future year cells are not observed"
		}
		years = append(years, ContractYear{Season: year, Cost: row.Salary, Inferred: inferred, Tag: tag, Reason: reason})
	}
	return years
}

func ContractsCap(rows []ContractRow, mine string, season int, used domain.Money, capAmount *domain.Money, names map[string]string) CapSummary {
	out := CapSummary{Team: contractTeamName(names, mine), Season: season, Used: contractAmount(used), CapSource: "Active MFL league configuration / commissioner override; configuration as-of time not held", Future: []ContractYear{}}
	if capAmount != nil {
		capValue, room := contractAmount(*capAmount), contractAmount(*capAmount-used)
		out.Cap, out.Room = &capValue, &room
	}
	future := make(map[int]domain.Money)
	for _, row := range rows {
		if row.FranchiseID != mine {
			continue
		}
		for _, year := range ContractYears(row, season) {
			if year.Season > season {
				future[year.Season] += year.Cost.Cents
			}
		}
	}
	for year, cents := range future {
		out.Future = append(out.Future, ContractYear{Season: year, Cost: contractAmount(cents), Inferred: true, Tag: "Inferred", Reason: "Sum of flat-filled annual salaries through expiration; gross commitments, no future roster discounts or adjustments assumed"})
	}
	slices.SortFunc(out.Future, func(a, b ContractYear) int { return cmp.Compare(a.Season, b.Season) })
	return out
}

func ContractsAdjustments(adjustments []state.Adjustment, names map[string]string) []SalaryAdjustment {
	out := make([]SalaryAdjustment, 0, len(adjustments))
	for _, a := range adjustments {
		out = append(out, SalaryAdjustment{ID: a.ID, Team: contractTeamName(names, a.FranchiseID), FranchiseID: a.FranchiseID, Amount: contractAmount(a.Amount), Description: a.Description})
	}
	return out
}
