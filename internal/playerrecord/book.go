package playerrecord

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

// Book is what one valuation reads: every record, and the production the model uses (league
// seasons, combine, college, contracts), all keyed by our ID. Nothing in it is keyed by an
// outside ID; MFL links resolve to our ID through ID.
type Book struct {
	Records map[string]Record // by our ID
	Data    model.Data        // by our ID, each player's identity the record's
	byMFL   map[string]string
}

// ID is our ID for an MFL link the book holds a record for.
func (b Book) ID(mfl string) (string, bool) {
	id, ok := b.byMFL[mfl]
	return id, ok
}

// NewBook files production observations under our ID and lays each record's identity over them.
// obs are keyed by MFL link, as the feeds stored them; resolve gives our ID for a link no record
// holds (a player with facts in history the record has not yet been written for). An
// observation whose link resolves to no one fails the book: dropping it would quietly change
// the population every player is scaled against.
func NewBook(records []Record, obs []model.Obs, lastWeek map[int]int, resolve func(mfl string) (string, bool)) (Book, error) {
	b := Book{Records: make(map[string]Record, len(records)), byMFL: make(map[string]string, len(records))}
	for _, r := range records {
		if _, dup := b.byMFL[r.MFL]; dup {
			return Book{}, fmt.Errorf("playerrecord: MFL link %s is on two records", r.MFL)
		}
		if _, dup := b.Records[r.ID]; dup {
			return Book{}, fmt.Errorf("playerrecord: player %s has two records", r.ID)
		}
		b.Records[r.ID], b.byMFL[r.MFL] = r, r.ID
	}
	keyed := make([]model.Obs, len(obs))
	for i, o := range obs {
		id, ok := b.byMFL[o.Player]
		if !ok {
			if id, ok = resolve(o.Player); !ok {
				return Book{}, fmt.Errorf("playerrecord: facts for MFL link %s belong to no player", o.Player)
			}
		}
		o.Player = id
		keyed[i] = o
	}
	b.Data = model.Build(keyed, lastWeek)
	for id, p := range b.Data.Players {
		if r, ok := b.Records[id]; ok {
			r.overlay(p)
		}
	}
	return b, nil
}

// overlay gives a history player the record's identity. The league scores a player at its own
// position (a DE's tackle is worth 2.5, an LB's 1.5), so a player MFL lists at a position the
// model scores moves to it, into that position's scale; one MFL lists elsewhere keeps nflverse's.
func (r Record) overlay(p *model.Player) {
	if r.Listed && slices.Contains(model.Positions(), r.Position) {
		p.Position = r.Position
	}
	p.Birth, p.Rookie, p.DraftPick = r.Birth, r.Rookie, float64(r.Draft.Overall)
	if r.Mislinked {
		p.Combine, p.Contracts = map[string]float64{}, map[int]map[string]float64{}
	}
}

// Player is the player the model values: his history facts under the record's identity, at the
// position the league lists him.
func (b Book) Player(id string) model.Player {
	r := b.Records[id]
	pl := model.Player{ID: id}
	if h := b.Data.Players[id]; h != nil {
		pl = *h
	} else {
		r.overlay(&pl)
	}
	pl.Position = r.Position
	return pl
}

// HeldID is a stand-in permanent ID for an MFL link, for a book built from held feeds with no
// identity store (an offline replay or a test). It is stable and distinct per link; it is never
// written anywhere.
func HeldID(mfl string) string {
	sum := sha256.Sum256([]byte("mfl:" + mfl))
	return "player." + hex.EncodeToString(sum[:16])
}

// HeldBook builds a book straight from held feeds: MFL's listing and history's observations, as
// of the moment they were read. It resolves records exactly as the stored record does, so a
// replay reads what a live run reads.
func HeldBook(listing Listing, obs []model.Obs, lastWeek map[int]int) (Book, error) {
	history := model.Build(obs, lastWeek).Players
	records, err := Assemble(listing, history, func(mfl string) (string, error) { return HeldID(mfl), nil })
	if err != nil {
		return Book{}, err
	}
	return NewBook(records, obs, lastWeek, func(mfl string) (string, bool) { return HeldID(mfl), true })
}

// Query is one valuation's read: production for seasons From..To as known at AsOf, the measures
// named, each season's weeks after LastWeek left out.
type Query struct {
	From, To int
	AsOf     time.Time
	Measures []string
	LastWeek map[int]int
}
