package labcompile

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type Asset struct {
	Kind              string  `json:"kind"`
	ID                *string `json:"id,omitempty"`
	Orig              *string `json:"orig,omitempty"`
	Year, Round, Slot *int64
}
type Trade struct {
	Year, Ts     int64
	A, B         string
	AGave, BGave []Asset
	OneSided     bool
}
type Pick struct {
	Year, Round, Slot int64
	Fid, Pid          string
	Ts                int64
}
type Move struct {
	Year, Ts       int64
	Fid, Kind, Pid string
}
type pickKey struct{ Year, Round, Slot int64 }
type pickOwner struct {
	Fid         string
	Year, Round int64
}
type tradeSide struct {
	Txn int64
	Fid string
}

func readTrades(f facts.Facts) ([]Trade, error) {
	gave := map[tradeSide][]Asset{}
	for i, id := range f.TxnAsset.TxnID {
		a := Asset{Kind: "pick", Orig: f.TxnAsset.PickOrig[i], Year: f.TxnAsset.PickYear[i], Round: f.TxnAsset.PickRound[i], Slot: f.TxnAsset.PickSlot[i]}
		if f.TxnAsset.Kind[i] == "player" {
			a = Asset{Kind: "player", ID: f.TxnAsset.Pid[i]}
		}
		key := tradeSide{id, value(f.TxnAsset.FromFid[i])}
		gave[key] = append(gave[key], a)
	}
	indices := make([]int, 0, len(f.Txn.TxnID))
	for i := range f.Txn.TxnID {
		indices = append(indices, i)
	}
	slices.SortFunc(indices, func(a, b int) int {
		if f.Txn.TxnID[a] < f.Txn.TxnID[b] {
			return -1
		}
		if f.Txn.TxnID[a] > f.Txn.TxnID[b] {
			return 1
		}
		return 0
	})
	out := []Trade{}
	for _, i := range indices {
		if f.Txn.Type[i] != "TRADE" || value(f.Txn.Ts[i]) == 0 {
			continue
		}
		a, b := value(f.Txn.Fid[i]), value(f.Txn.Fid2[i])
		ag, bg := gave[tradeSide{f.Txn.TxnID[i], a}], gave[tradeSide{f.Txn.TxnID[i], b}]
		var raw RawTradeOrder
		if err := json.Unmarshal([]byte(f.Txn.Raw[i]), &raw); err != nil {
			return nil, fmt.Errorf("trade %d order: %w", f.Txn.TxnID[i], err)
		}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("trade %d order: %w", f.Txn.TxnID[i], err)
		}
		var err error
		ag, err = orderAssets(f, f.Txn.TxnID[i], a, raw.A, ag)
		if err != nil {
			return nil, err
		}
		bg, err = orderAssets(f, f.Txn.TxnID[i], b, raw.B, bg)
		if err != nil {
			return nil, err
		}
		out = append(out, Trade{f.Txn.Year[i], *f.Txn.Ts[i], a, b, ag, bg, len(ag) == 0 || len(bg) == 0})
	}
	return out, nil
}
func readDraft(f facts.Facts) ([]Pick, map[int64]draftSpan) {
	out := []Pick{}
	spans := map[int64]draftSpan{}
	for i, y := range f.DraftPick.Year {
		if f.DraftPick.Pid[i] == nil || value(f.DraftPick.Ts[i]) == 0 {
			continue
		}
		ts := *f.DraftPick.Ts[i]
		out = append(out, Pick{y, f.DraftPick.Round[i], f.DraftPick.Pick[i], value(f.DraftPick.Fid[i]), *f.DraftPick.Pid[i], ts})
		s, ok := spans[y]
		if !ok {
			s = draftSpan{ts, ts}
		}
		s.First = min(s.First, ts)
		s.Last = max(s.Last, ts)
		spans[y] = s
	}
	slices.SortFunc(out, func(a, b Pick) int {
		if a.Year != b.Year {
			return int(a.Year - b.Year)
		}
		if a.Round != b.Round {
			return int(a.Round - b.Round)
		}
		return int(a.Slot - b.Slot)
	})
	return out, spans
}
func readMoves(f facts.Facts) []Move {
	txns := map[int64]int{}
	for i, id := range f.Txn.TxnID {
		txns[id] = i
	}
	out := []Move{}
	for i, id := range f.TxnAsset.TxnID {
		ti := txns[id]
		if value(f.Txn.Ts[ti]) == 0 {
			continue
		}
		kind := ""
		switch f.Txn.Type[ti] {
		case "FREE_AGENT", "LOAD_ROSTERS", "WAIVER", "TAXI", "IR":
			kind = moveKind(f.TxnAsset.Role[i])
		}
		if kind != "" {
			out = append(out, Move{f.Txn.Year[ti], *f.Txn.Ts[ti], value(f.Txn.Fid[ti]), kind, value(f.TxnAsset.Pid[i])})
		}
	}
	return out
}
func moveKind(role string) string {
	switch role {
	case "add", "load":
		return "add"
	case "drop":
		return "drop"
	case "taxi_off":
		return "taxiUp"
	case "taxi_on":
		return "taxiDown"
	case "ir_on":
		return "irOn"
	case "ir_off":
		return "irOff"
	}
	return ""
}

type RawTradeOrder struct {
	A string `json:"franchise1_gave_up"`
	B string `json:"franchise2_gave_up"`
}

func (r RawTradeOrder) Validate() error {
	if strings.ContainsRune(r.A+r.B, '\x00') {
		return fmt.Errorf("trade asset list contains NUL")
	}
	return nil
}
func orderAssets(f facts.Facts, id int64, fid, raw string, assets []Asset) ([]Asset, error) {
	byCode := map[string]Asset{}
	j := 0
	for i, txn := range f.TxnAsset.TxnID {
		if txn == id && value(f.TxnAsset.FromFid[i]) == fid {
			byCode[f.TxnAsset.Code[i]] = assets[j]
			j++
		}
	}
	out := make([]Asset, 0, len(assets))
	for _, code := range strings.Split(raw, ",") {
		if code == "" {
			continue
		}
		a, ok := byCode[code]
		if !ok {
			return nil, fmt.Errorf("trade %d missing asset code %s", id, code)
		}
		out = append(out, a)
	}
	if len(out) != len(assets) {
		return nil, fmt.Errorf("trade %d asset order count %d != %d", id, len(out), len(assets))
	}
	return out, nil
}
