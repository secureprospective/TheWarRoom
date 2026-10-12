package facts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// contracts reads every contract MFL held at the pull (salaries.json) and the season's cap charges
// (salaryAdjustments.json), which repeat earlier cuts that still charge it.
func (c *compiler) contracts(year int64) error {
	src := seasonFile(year, "salaries")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, p := range d.get("salaries", "leagueUnit", "player").rows() {
		pid, err := keyID(p, "id", src)
		if err != nil {
			return err
		}
		c.facts.Contract = append(c.facts.Contract, Contract{
			Year: year, Pid: pid, Salary: p.get("salary").num(), ContractYear: nonempty(p.get("contractYear").str()),
			ContractStatus: nonempty(p.get("contractStatus").str()), ContractInfo: trimmed(p.get("contractInfo")), Src: src,
		})
	}
	src = seasonFile(year, "salaryAdjustments")
	if d, err = c.read(src); err != nil {
		return err
	}
	for _, a := range d.get("salaryAdjustments", "salaryAdjustment").rows() {
		id, err := keyID(a, "id", src)
		if err != nil {
			return err
		}
		c.facts.CapCharge = append(c.facts.CapCharge, CapCharge{
			Year: year, AdjID: id, Fid: a.get("franchise_id").str(), Amount: a.get("amount").num(),
			Description: a.get("description").str(), Ts: a.get("timestamp").whole(), Src: src,
		})
	}
	return nil
}

// draft reads every rookie-draft selection. MFL marks a skipped or forfeited pick "----".
func (c *compiler) draft(year int64) error {
	src := seasonFile(year, "draftResults")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, unit := range d.get("draftResults", "draftUnit").rows() {
		for _, p := range unit.get("draftPick").rows() {
			round, pick := p.get("round").whole(), p.get("pick").whole()
			if round == nil || pick == nil {
				return fmt.Errorf("%s: draft pick without round or pick", src)
			}
			pid := nonempty(p.get("player").str())
			if value(pid) == "----" {
				pid = nil
			}
			ts := p.get("timestamp")
			c.facts.DraftPick = append(c.facts.DraftPick, DraftPick{
				Year: year, Round: *round, Pick: *pick, Fid: p.get("franchise").str(), Pid: pid, Ts: ts.whole(),
				Day: easternDay(ts.str(), c.eastern), Comments: trimmed(p.get("comments")), Src: src,
			})
		}
	}
	return nil
}

// picksOwned reads who owned which future picks at the pull, and the picks listed as tradable
// assets with MFL's codes.
func (c *compiler) picksOwned(year int64) error {
	src := seasonFile(year, "futureDraftPicks")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, f := range d.get("futureDraftPicks", "franchise").rows() {
		fid, err := keyID(f, "id", src)
		if err != nil {
			return err
		}
		for _, p := range f.get("futureDraftPick").rows() {
			c.facts.PickOwned = append(c.facts.PickOwned, PickOwned{
				SnapshotYear: year, Source: "futureDraftPicks", Fid: fid, PickYear: p.get("year").whole(),
				Round: p.get("round").whole(), OrigFid: p.get("originalPickFor").str(), Src: src,
			})
		}
	}
	src = seasonFile(year, "assets")
	if d, err = c.read(src); err != nil {
		return err
	}
	for _, f := range d.get("assets", "franchise").rows() {
		fid, err := keyID(f, "id", src)
		if err != nil {
			return err
		}
		for _, key := range []string{"currentYearDraftPicks", "futureYearDraftPicks"} {
			for _, p := range f.get(key, "draftPick").rows() {
				code := p.get("pick").str()
				a := decodeAsset(value(code), year)
				c.facts.PickOwned = append(c.facts.PickOwned, PickOwned{
					SnapshotYear: year, Source: "assets", Fid: fid, PickYear: a.pickYear, Round: a.pickRound, Slot: a.pickSlot,
					OrigFid: a.pickOrig, Code: code, Description: p.get("description").str(), Src: src,
				})
			}
		}
	}
	return nil
}

// tradeBait reads what owners advertised.
func (c *compiler) tradeBait(year int64) error {
	src := seasonFile(year, "tradeBait")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, b := range d.get("tradeBaits", "tradeBait").rows() {
		fid := b.get("franchise_id").str()
		if fid == nil {
			return fmt.Errorf("%s: trade bait without a franchise", src)
		}
		c.facts.TradeBait = append(c.facts.TradeBait, TradeBait{
			Year: year, Fid: *fid, Ts: b.get("timestamp").whole(), WillGiveUp: nonempty(b.get("willGiveUp").str()),
			InExchangeFor: trimmed(b.get("inExchangeFor")), Src: src,
		})
	}
	return nil
}

// mflWide reads MFL-wide average draft position and auction value: the wider market, not this league.
func (c *compiler) mflWide(year int64) error {
	src := seasonFile(year, "adp")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for _, p := range d.get("adp", "player").rows() {
		pid, err := keyID(p, "id", src)
		if err != nil {
			return err
		}
		c.facts.MflAdp = append(c.facts.MflAdp, MflAdp{
			Year: year, Pid: pid, Rank: p.get("rank").whole(), AvgPick: p.get("averagePick").num(),
			MinPick: p.get("minPick").whole(), MaxPick: p.get("maxPick").whole(), Drafts: p.get("draftsSelectedIn").whole(), Src: src,
		})
	}
	src = seasonFile(year, "aav")
	if d, err = c.read(src); err != nil {
		return err
	}
	for _, p := range d.get("aav", "player").rows() {
		pid, err := keyID(p, "id", src)
		if err != nil {
			return err
		}
		c.facts.MflAav = append(c.facts.MflAav, MflAav{
			Year: year, Pid: pid, Rank: p.get("rank").whole(), AvgValue: p.get("averageValue").num(),
			MinValue: p.get("minValue").num(), MaxValue: p.get("maxValue").num(), Src: src,
		})
	}
	return nil
}

// isOffer: the transactions with two sides giving assets, trades and the offers around them.
func isOffer(kind string) bool {
	switch kind {
	case "TRADE", "TRADE_PROPOSAL", "TRADE_ACCEPT", "TRADE_REJECTION", "TRADE_REVOKE", "TRADE_OFFER_EXPIRED":
		return true
	}
	return false
}

// transactions numbers every logged transaction in a stable order (season, then position in MFL's
// file) and decodes each asset it moved.
func (c *compiler) transactions(year int64) error {
	src := seasonFile(year, "transactions")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	for i, x := range d.get("transactions", "transaction").rows() {
		c.txnID++
		kind := x.get("type").str()
		if kind == nil {
			return fmt.Errorf("%s: transaction %d has no type", src, i)
		}
		fid := nonempty(x.get("franchise").str())
		ts := x.get("timestamp")
		var byCommish int64
		if value(x.get("by_commish").str()) == "1" {
			byCommish = 1
		}
		c.facts.Txn = append(c.facts.Txn, Txn{
			TxnID: c.txnID, Year: year, Ts: ts.whole(), Day: easternDay(ts.str(), c.eastern), Type: *kind, Fid: fid,
			Fid2: nonempty(x.get("franchise2").str()), ByCommish: byCommish, Expires: x.get("expires").whole(),
			Comments: trimmed(x.get("comments")), Raw: x.dump(), Src: src, SrcIdx: int64(i),
		})
		c.txnAssets(x, *kind, fid, year)
	}
	return nil
}

// txnAssets decodes the asset lists of one transaction. An offer side's other franchise is MFL's
// franchise2 as given.
func (c *compiler) txnAssets(x RawValue, kind string, fid *string, year int64) {
	add := func(role, list string, from, to *string) {
		for _, code := range idList(list) {
			a := decodeAsset(code, year)
			c.facts.TxnAsset = append(c.facts.TxnAsset, TxnAsset{
				TxnID: c.txnID, Role: role, FromFid: from, ToFid: to, Kind: a.kind, Pid: a.pid, PickYear: a.pickYear,
				PickRound: a.pickRound, PickSlot: a.pickSlot, PickOrig: a.pickOrig, Code: a.code,
			})
		}
	}
	field := func(key string) string { return value(x.get(key).str()) }
	switch {
	case isOffer(kind):
		other := x.get("franchise2").str()
		add("sent", field("franchise1_gave_up"), fid, other)
		add("sent", field("franchise2_gave_up"), other, fid)
	case kind == "FREE_AGENT" || kind == "LOAD_ROSTERS":
		added, dropped, _ := strings.Cut(field("transaction"), "|")
		role := "add"
		if kind == "LOAD_ROSTERS" {
			role = "load"
		}
		add(role, added, nil, fid)
		add("drop", dropped, fid, nil)
	case kind == "WAIVER":
		add("add", field("added"), nil, fid)
		add("drop", field("dropped"), fid, nil)
	case kind == "IR":
		add("ir_on", field("deactivated"), fid, fid)
		add("ir_off", field("activated"), fid, fid)
	case kind == "TAXI":
		add("taxi_on", field("demoted"), fid, fid)
		add("taxi_off", field("promoted"), fid, fid)
	}
}

// idList reads MFL's comma-separated list in order, blanks dropped.
func idList(s string) []string {
	var out []string
	for _, id := range strings.Split(s, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

type asset struct {
	kind, code                    string
	pid, pickOrig                 *string
	pickYear, pickRound, pickSlot *int64
}

var (
	futurePick  = regexp.MustCompile(`^FP_([0-9]{4})_([0-9]{4})_([0-9]+)$`)
	currentPick = regexp.MustCompile(`^DP_([0-9]+)_([0-9]+)$`)
	playerCode  = regexp.MustCompile(`^[0-9]+$`)
)

// decodeAsset reads MFL's asset token. FP_<orig>_<year>_<round> is a future pick; DP_<r>_<s> is a
// pick in this season's draft counted from zero (DP_3_20 is round 4, pick 21); digits are a player.
func decodeAsset(code string, year int64) asset {
	code = strings.TrimSpace(code)
	a := asset{kind: "other", code: code}
	if m := futurePick.FindStringSubmatch(code); m != nil {
		a.kind, a.pickOrig, a.pickYear, a.pickRound = "pick", ptr(m[1]), atoi(m[2], 0), atoi(m[3], 0)
	} else if m := currentPick.FindStringSubmatch(code); m != nil {
		a.kind, a.pickYear, a.pickRound, a.pickSlot = "pick", ptr(year), atoi(m[1], 1), atoi(m[2], 1)
	} else if playerCode.MatchString(code) {
		a.kind, a.pid = "player", ptr(code)
	}
	return a
}

// atoi reads digits the patterns above already matched, plus an offset.
func atoi(digits string, offset int64) *int64 {
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil { // more digits than an int64 holds: no number
		return nil
	}
	return ptr(n + offset)
}

// easternDay is a Unix time as the calendar date MFL shows (US Eastern); zero or non-numeric is null.
func easternDay(ts *string, eastern *time.Location) *string {
	s := strings.TrimPrefix(value(ts), "-")
	if s == "" || !playerCode.MatchString(s) {
		return nil
	}
	return birthdate(ts, eastern)
}
