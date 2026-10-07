package leaguefeed

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// ParseAssets reads an MFL asset list ("16289,DP_4_9,FP_0019_2027_1,BB_10.50,"); an
// unknown token is an error naming it.
func ParseAssets(list string) ([]Asset, error) {
	assets := make([]Asset, 0)
	if list == "" {
		return assets, nil
	}
	for _, token := range strings.Split(strings.TrimSuffix(list, ","), ",") {
		asset, err := parseAsset(token)
		if err != nil {
			return nil, fmt.Errorf("asset %q: %w", token, err)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func parseAsset(token string) (Asset, error) {
	parts := strings.Split(token, "_")
	switch {
	case len(parts) == 3 && parts[0] == "DP":
		// MFL writes both zero-based, padded ("DP_02_05") or not ("DP_4_9").
		if len(parts[1]) > 2 || len(parts[2]) > 2 {
			return Asset{}, fmt.Errorf("draft pick round and pick are at most two digits")
		}
		round, err := decimal(parts[1])
		if err != nil {
			return Asset{}, err
		}
		pick, err := decimal(parts[2])
		if err != nil {
			return Asset{}, err
		}
		return Asset{CurrentPick: &CurrentPick{Round: round + 1, Pick: pick + 1}}, nil
	case len(parts) == 4 && parts[0] == "FP":
		return parseFuture(parts)
	case len(parts) == 2 && parts[0] == "BB":
		cents, err := money(parts[1])
		if err != nil {
			return Asset{}, err
		}
		return Asset{BlindBidCents: &cents}, nil
	default:
		if _, err := decimal(token); err != nil {
			return Asset{}, fmt.Errorf("unknown token: %w", err)
		}
		id, err := playerid.New(token)
		if err != nil {
			return Asset{}, fmt.Errorf("player: %w", err)
		}
		return Asset{Player: &id}, nil
	}
}

func parseFuture(parts []string) (Asset, error) {
	if _, err := decimal(parts[1]); err != nil || len(parts[1]) != 4 {
		return Asset{}, fmt.Errorf("original franchise %q needs four digits", parts[1])
	}
	year, err := decimal(parts[2])
	if err != nil {
		return Asset{}, err
	}
	round, err := decimal(parts[3])
	if err != nil {
		return Asset{}, err
	}
	if len(parts[2]) != 4 || year < 1000 || round < 1 {
		return Asset{}, fmt.Errorf("future pick needs a four-digit year and positive round")
	}
	return Asset{FuturePick: &FuturePick{Franchise: parts[1], Year: year, Round: round}}, nil
}

func decimal(s string) (int, error) {
	if s == "" || strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return 0, fmt.Errorf("invalid unsigned decimal %q", s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("decimal %q: %w", s, err)
	}
	return n, nil
}

func money(s string) (int64, error) {
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid dollars %q", s)
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, fmt.Errorf("invalid cents %q", s)
		}
		fraction = parts[1] + strings.Repeat("0", 2-len(parts[1]))
	}
	if _, err := decimal(parts[0]); err != nil {
		return 0, err
	}
	if _, err := decimal(fraction); err != nil {
		return 0, err
	}
	cents, err := strconv.ParseInt(parts[0]+fraction, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money %q: %w", s, err)
	}
	return cents, nil
}
