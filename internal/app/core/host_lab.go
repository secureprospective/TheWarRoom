package core

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// Lab returns the shared decode, or a fresh one for state built without NewApp.
// Its storage is immutable; feature hosts must detach what they expose.
func (s *State) Lab() (*lab.Data, error) {
	if s.LabData == nil {
		d, err := lab.Load()
		if err != nil {
			return nil, fmt.Errorf("load lab facts: %w", err)
		}
		return d, nil
	}
	return s.LabData()
}
