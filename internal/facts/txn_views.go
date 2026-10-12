package facts

// TradeAsset is every completed trade, one row per asset that moved, with names in words. Before
// 2017 MFL did not record picks in trades; the trade's note may.
type TradeAsset struct {
	TxnID        int64   `json:"txn_id"`
	Year         int64   `json:"year"`
	Ts           *int64  `json:"ts"`
	Day          *string `json:"day"`
	ByCommish    int64   `json:"by_commish"`
	FromFid      *string `json:"from_fid"`
	FromName     *string `json:"from_name"`
	ToFid        *string `json:"to_fid"`
	ToName       *string `json:"to_name"`
	Kind         string  `json:"kind"`
	Code         string  `json:"code"`
	Pid          *string `json:"pid"`
	Player       *string `json:"player"`
	Position     *string `json:"position"`
	PickYear     *int64  `json:"pick_year"`
	PickRound    *int64  `json:"pick_round"`
	PickSlot     *int64  `json:"pick_slot"`
	PickOrig     *string `json:"pick_orig"`
	PickOrigName *string `json:"pick_orig_name"`
	Comments     *string `json:"comments"`
}

// Offer is an offer and how it ended: TRADE_REJECTION, TRADE_REVOKE (withdrawn) or
// TRADE_OFFER_EXPIRED; TRADE_ACCEPT closes an offer that became a trade.
type Offer struct {
	TxnID     int64   `json:"txn_id"`
	Year      int64   `json:"year"`
	Ts        *int64  `json:"ts"`
	Day       *string `json:"day"`
	Type      string  `json:"type"`
	OfferedBy *string `json:"offered_by"`
	OfferedTo *string `json:"offered_to"`
	Comments  *string `json:"comments"`
	Expires   *int64  `json:"expires"`
}

// RosterMove is every roster move outside trades: adds, drops, roster loads, IR and taxi changes.
type RosterMove struct {
	TxnID     int64   `json:"txn_id"`
	Year      int64   `json:"year"`
	Ts        *int64  `json:"ts"`
	Day       *string `json:"day"`
	Type      string  `json:"type"`
	Role      string  `json:"role"`
	Fid       *string `json:"fid"`
	Pid       *string `json:"pid"`
	Player    *string `json:"player"`
	Position  *string `json:"position"`
	ByCommish int64   `json:"by_commish"`
}

type TradeAssetColumns struct {
	TxnID        []int64   `json:"txn_id"`
	Year         []int64   `json:"year"`
	Ts           []*int64  `json:"ts"`
	Day          []*string `json:"day"`
	ByCommish    []int64   `json:"by_commish"`
	FromFid      []*string `json:"from_fid"`
	FromName     []*string `json:"from_name"`
	ToFid        []*string `json:"to_fid"`
	ToName       []*string `json:"to_name"`
	Kind         []string  `json:"kind"`
	Code         []string  `json:"code"`
	Pid          []*string `json:"pid"`
	Player       []*string `json:"player"`
	Position     []*string `json:"position"`
	PickYear     []*int64  `json:"pick_year"`
	PickRound    []*int64  `json:"pick_round"`
	PickSlot     []*int64  `json:"pick_slot"`
	PickOrig     []*string `json:"pick_orig"`
	PickOrigName []*string `json:"pick_orig_name"`
	Comments     []*string `json:"comments"`
}

type OfferColumns struct {
	TxnID     []int64   `json:"txn_id"`
	Year      []int64   `json:"year"`
	Ts        []*int64  `json:"ts"`
	Day       []*string `json:"day"`
	Type      []string  `json:"type"`
	OfferedBy []*string `json:"offered_by"`
	OfferedTo []*string `json:"offered_to"`
	Comments  []*string `json:"comments"`
	Expires   []*int64  `json:"expires"`
}

type RosterMoveColumns struct {
	TxnID     []int64   `json:"txn_id"`
	Year      []int64   `json:"year"`
	Ts        []*int64  `json:"ts"`
	Day       []*string `json:"day"`
	Type      []string  `json:"type"`
	Role      []string  `json:"role"`
	Fid       []*string `json:"fid"`
	Pid       []*string `json:"pid"`
	Player    []*string `json:"player"`
	Position  []*string `json:"position"`
	ByCommish []int64   `json:"by_commish"`
}

// isRosterType: the transactions that move players on and off rosters outside trades.
func isRosterType(kind string) bool {
	switch kind {
	case "FREE_AGENT", "WAIVER", "LOAD_ROSTERS", "IR", "TAXI":
		return true
	}
	return false
}

// txnViews computes trade_asset, offer and roster_move from the transaction columns.
func (f Facts) txnViews() ([]TradeAsset, []Offer, []RosterMove) {
	t := f.Txn
	byID := make(map[int64]int, len(t.TxnID))
	var offers []Offer
	for i, id := range t.TxnID {
		byID[id] = i
		if t.Type[i] != "TRADE" && isOffer(t.Type[i]) {
			offers = append(offers, Offer{TxnID: id, Year: t.Year[i], Ts: t.Ts[i], Day: t.Day[i], Type: t.Type[i],
				OfferedBy: t.Fid[i], OfferedTo: t.Fid2[i], Comments: t.Comments[i], Expires: t.Expires[i]})
		}
	}
	names := f.franchiseNames()
	players := make(map[string]int, len(f.Player.Pid))
	for i, pid := range f.Player.Pid {
		players[pid] = i
	}
	player := func(pid *string) (*string, *string) {
		if i, ok := players[value(pid)]; ok && pid != nil {
			return ptr(f.Player.Name[i]), f.Player.Position[i]
		}
		return nil, nil
	}
	var trades []TradeAsset
	var moves []RosterMove
	a := f.TxnAsset
	for j, id := range a.TxnID {
		i, ok := byID[id]
		if !ok {
			continue
		}
		year := t.Year[i]
		name, position := player(a.Pid[j])
		switch {
		case t.Type[i] == "TRADE":
			trades = append(trades, TradeAsset{TxnID: id, Year: year, Ts: t.Ts[i], Day: t.Day[i], ByCommish: t.ByCommish[i],
				FromFid: a.FromFid[j], FromName: names.of(year, a.FromFid[j]), ToFid: a.ToFid[j], ToName: names.of(year, a.ToFid[j]),
				Kind: a.Kind[j], Code: a.Code[j], Pid: a.Pid[j], Player: name, Position: position, PickYear: a.PickYear[j],
				PickRound: a.PickRound[j], PickSlot: a.PickSlot[j], PickOrig: a.PickOrig[j],
				PickOrigName: names.of(year, a.PickOrig[j]), Comments: t.Comments[i]})
		case isRosterType(t.Type[i]):
			fid := a.ToFid[j]
			if fid == nil {
				fid = a.FromFid[j]
			}
			moves = append(moves, RosterMove{TxnID: id, Year: year, Ts: t.Ts[i], Day: t.Day[i], Type: t.Type[i], Role: a.Role[j],
				Fid: fid, Pid: a.Pid[j], Player: name, Position: position, ByCommish: t.ByCommish[i]})
		}
	}
	return trades, offers, moves
}

type franchiseNames map[yearFid]*string

func (f Facts) franchiseNames() franchiseNames {
	out := make(franchiseNames, len(f.FranchiseSeason.Fid))
	for i, year := range f.FranchiseSeason.Year {
		out[yearFid{year, f.FranchiseSeason.Fid[i]}] = f.FranchiseSeason.Name[i]
	}
	return out
}

// of is a franchise's name that season; a null id or a franchise not in the league is null.
func (n franchiseNames) of(year int64, fid *string) *string {
	if fid == nil {
		return nil
	}
	return n[yearFid{year, *fid}]
}
