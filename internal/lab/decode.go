// Package lab computes historical Market readings from fingerprinted compiled facts.
package lab

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const DataSHA = "6a9995ad26b4e4c5bc63dc1b771f6c38c3af71af7493d70263f332aed3748830"

//go:embed testdata/lab.json.gz
var frozen []byte

type Data struct {
	OfferOwnerID   string
	Raw            RawData
	Trades         []Trade
	Legs           []Leg
	Delivered      []Delivery
	Draft          []DraftPick
	TeamSeasons    []TeamSeason
	Offers         []Offer
	Players        map[string]*Player
	Seasons        map[int]RawSeason
	Franchises     map[string]RawFranchise
	Years          []int
	LastFinished   int
	Built          string
	ArchiveThrough string
}

// Load decodes the frozen facts and checks their fingerprint; callers must not mutate the result.
func Load() (*Data, error) { return DecodeGzip(frozen, DataSHA) }

func DecodeGzip(compressed []byte, wantSHA string) (*Data, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open Lab gzip: %w", err)
	}
	b, readErr := io.ReadAll(io.LimitReader(r, 16<<20))
	closeErr := r.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read Lab gzip: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Lab gzip: %w", closeErr)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != wantSHA {
		return nil, fmt.Errorf("lab JSON fingerprint mismatch")
	}
	var raw RawData
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode Lab columns: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, fmt.Errorf("validate Lab: %w", err)
	}
	return decode(raw), nil
}

func (r RawData) Validate() error {
	if r.Schema != "league-lab-2" {
		return fmt.Errorf("unsupported Lab schema %q", r.Schema)
	}
	for _, stamp := range []string{r.Built, r.ArchiveThrough} {
		if _, err := time.Parse(time.RFC3339, stamp); err != nil {
			return fmt.Errorf("lab timestamp: %w", err)
		}
	}
	checks := []func() error{r.Trades.Validate, r.Legs.Validate, r.Delivered.Validate, r.Players.Validate, r.Draft.Validate, r.TeamSeasons.Validate, r.Offers.Validate}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	for _, id := range append(append([]int{}, r.Legs.Trade...), r.Delivered.Trade...) {
		if id < 0 || id >= len(r.Trades.Ts) {
			return fmt.Errorf("invalid trade link %d", id)
		}
	}
	if len(r.Seasons) == 0 {
		return fmt.Errorf("no seasons")
	}
	return nil
}

func decode(raw RawData) *Data {
	d := &Data{Raw: raw, Trades: raw.Trades.rows(), Legs: raw.Legs.rows(), Delivered: raw.Delivered.rows(), Draft: raw.Draft.rows(), TeamSeasons: raw.TeamSeasons.rows(), Offers: raw.Offers.rows(), Built: raw.Built, ArchiveThrough: raw.ArchiveThrough, Players: make(map[string]*Player), Seasons: make(map[int]RawSeason), Franchises: make(map[string]RawFranchise)}
	for _, p := range raw.Players.rows() {
		d.Players[p.ID] = &p
	}
	for _, s := range raw.Seasons {
		d.Seasons[s.Year] = s
		d.Years = append(d.Years, s.Year)
		if len(s.WeeksDone) >= s.RegularWeeks && s.Year > d.LastFinished {
			d.LastFinished = s.Year
		}
	}
	for _, f := range raw.Franchises {
		d.Franchises[f.ID] = f
	}
	for i := range d.Trades {
		d.Trades[i].ID = i
		d.Trades[i].Date = time.Unix(int64(d.Trades[i].Ts), 0).UTC()
		d.Trades[i].Legs = []Leg{}
	}
	for i := range d.Legs {
		l := &d.Legs[i]
		if l.From != d.Trades[l.Trade].A {
			l.Side = 1
		}
		if l.Pid != nil {
			l.Player = d.Players[*l.Pid]
		}
		if l.PickPid != nil {
			l.PickPlayer = d.Players[*l.PickPid]
		}
		d.Trades[l.Trade].Legs = append(d.Trades[l.Trade].Legs, *l)
	}
	for i := range d.Draft {
		p := &d.Draft[i]
		p.ID = i
		p.Player = d.Players[p.Pid]
		p.Band = band(p.Round, p.Slot)
	}
	return d
}

func band(round, slot int) string {
	if round == 1 {
		if slot <= 10 {
			return "early1"
		}
		return "late1"
	}
	if round == 2 {
		return "r2"
	}
	if round == 3 {
		return "r3"
	}
	return "r4"
}
