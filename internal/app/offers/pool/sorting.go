package pool

import (
	"cmp"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Order is the sortable projection of a row; missing values always sort last.
type Order struct {
	ID, Name, Position, Team string
	Minimum                  domain.Money
	Engine, PPG, Lab         *float64
	Market                   [4]*float64
}

func (r Order) Number(key string) *float64 {
	switch key {
	case "engine":
		return r.Engine
	case "ppg":
		return r.PPG
	case "lab":
		return r.Lab
	default:
		return nil
	}
}

func numericOrder(x, y Order, key string) (int, bool) {
	xv, yv := x.Number(key), y.Number(key)
	if index := slices.Index([]string{"projected", "rank", "adp", "aav"}, key); index >= 0 {
		xv, yv = x.Market[index], y.Market[index]
	}
	if xv == nil && yv == nil {
		return 0, false
	}
	if xv == nil {
		return 1, true
	}
	if yv == nil {
		return -1, true
	}
	return cmp.Compare(*xv, *yv), false
}

func Compare(x, y Order, key string, desc bool) int {
	c, missing := numericOrder(x, y, key)
	if missing {
		return c
	}
	switch key {
	case "team":
		c = strings.Compare(x.Team, y.Team)
	case "position":
		c = cmp.Compare(PositionOrder(x.Position), PositionOrder(y.Position))
	case "minimum":
		c = cmp.Compare(x.Minimum, y.Minimum)
	}
	if c == 0 {
		c = cmp.Or(strings.Compare(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	}
	if desc {
		return -c
	}
	return c
}
