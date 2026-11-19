package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const HomeLayoutMaxBytes = 8 << 10

// RawHomeLayoutRead has no caller-selected identity; the hosted session supplies the account.
type RawHomeLayoutRead struct{}

func (RawHomeLayoutRead) Validate() error { return nil }

// RawHomeLayoutSet carries the only client-controlled part of an account layout write.
type RawHomeLayoutSet struct {
	Layout           json.RawMessage `json:"layout"`
	ExpectedRevision int64           `json:"expectedRevision"`
}

func (r RawHomeLayoutSet) Validate() error {
	if r.ExpectedRevision < 0 {
		return fmt.Errorf("layout revision must not be negative")
	}
	if _, err := ValidateHomeLayout(r.Layout); err != nil {
		return fmt.Errorf("home layout: %w", err)
	}
	return nil
}

// HomeLayoutState is one account's checked layout and its optimistic-write revision.
type HomeLayoutState struct {
	Layout   json.RawMessage `json:"layout"`
	Revision int64           `json:"revision"`
	Conflict bool            `json:"conflict,omitempty"`
}

// HomeLayoutRepository is the trusted account-scoped preference port used by hosted workflows.
type HomeLayoutRepository interface {
	ReadHomeLayout(context.Context, string) (HomeLayoutState, error)
	SetHomeLayout(context.Context, string, RawHomeLayoutSet) (HomeLayoutState, error)
}

type homeLayoutDocument struct {
	Mode  string          `json:"mode"`
	Tiles [][]string      `json:"tiles"`
	Over  []string        `json:"over"`
	Pins  map[string]bool `json:"pins"`
}

// ValidateHomeLayout checks bounds and every board identity before returning a normalized document.
func ValidateHomeLayout(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > HomeLayoutMaxBytes {
		return nil, fmt.Errorf("layout must be between 1 and %d bytes", HomeLayoutMaxBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document homeLayoutDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode layout: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("send one layout object")
	}
	if err := document.validate(); err != nil {
		return nil, err
	}
	if document.Tiles == nil {
		document.Tiles = [][]string{}
	}
	if document.Over == nil {
		document.Over = []string{}
	}
	if document.Pins == nil {
		document.Pins = map[string]bool{}
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode layout: %w", err)
	}
	if len(canonical) > HomeLayoutMaxBytes {
		return nil, fmt.Errorf("layout exceeds %d bytes", HomeLayoutMaxBytes)
	}
	return canonical, nil
}

func (d homeLayoutDocument) validate() error {
	if d.Mode != "default" && d.Mode != "custom" {
		return fmt.Errorf("unknown layout mode")
	}
	if len(d.Tiles)+len(d.Over)+len(d.Pins) > 40 {
		return fmt.Errorf("layout has more than 40 entries")
	}
	if err := d.validateBoards(); err != nil {
		return err
	}
	if d.Mode == "default" && (len(d.Tiles) != 0 || len(d.Over) != 0) {
		return fmt.Errorf("default layout cannot contain custom tiles")
	}
	if d.Mode == "custom" && !homeTilesPack(d.Tiles) {
		return fmt.Errorf("custom tiles do not fit the board")
	}
	return nil
}

func (d homeLayoutDocument) validateBoards() error {
	seen := make(map[string]bool, len(d.Tiles)+len(d.Over))
	for _, tile := range d.Tiles {
		if len(tile) != 2 || !validHomeTile(tile[0], tile[1]) {
			return fmt.Errorf("unknown board or tile size")
		}
		if seen[tile[0]] {
			return fmt.Errorf("board appears more than once")
		}
		seen[tile[0]] = true
	}
	for _, id := range d.Over {
		if !validHomeBoard(id) {
			return fmt.Errorf("unknown overflow board")
		}
		if seen[id] {
			return fmt.Errorf("board appears more than once")
		}
		seen[id] = true
	}
	for id, pinned := range d.Pins {
		if !validHomeBoard(id) || !pinned {
			return fmt.Errorf("unknown pinned board")
		}
	}
	return nil
}

func validHomeTile(id, span string) bool {
	if !validHomeBoard(id) {
		return false
	}
	switch id {
	case "due", "picks", "cap", "bids":
		return span == "1x1"
	case "race", "sc":
		return span == "1x1" || span == "2x1"
	case "pwr", "plan":
		return span == "1x1" || span == "1x2"
	case "mu", "head", "news", "wire":
		return span == "1x1" || span == "2x1" || span == "1x2"
	case "form":
		return span == "1x1" || span == "2x1"
	default:
		return false
	}
}

func validHomeBoard(id string) bool {
	switch id {
	case "due", "mu", "race", "head", "news", "wire", "sc", "pwr", "plan", "picks", "cap", "bids", "form":
		return strings.TrimSpace(id) == id
	default:
		return false
	}
}

func homeTilesPack(tiles [][]string) bool {
	var occupied [2][3]bool
	for _, tile := range tiles {
		width, height := 1, 1
		switch tile[1] {
		case "2x1":
			width = 2
		case "1x2":
			height = 2
		}
		placed := false
		for row := 0; row <= 2-height && !placed; row++ {
			for col := 0; col <= 3-width && !placed; col++ {
				free := true
				for y := row; y < row+height; y++ {
					for x := col; x < col+width; x++ {
						free = free && !occupied[y][x]
					}
				}
				if free {
					for y := row; y < row+height; y++ {
						for x := col; x < col+width; x++ {
							occupied[y][x] = true
						}
					}
					placed = true
				}
			}
		}
		if !placed {
			return false
		}
	}
	return true
}
