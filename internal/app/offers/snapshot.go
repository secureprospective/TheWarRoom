package offers

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func PersonalSnapshot(snap snapshot.Snapshot, loaded bool, mine string) (snapshot.Snapshot, error) {
	if !loaded {
		return snap, fmt.Errorf("personal: snapshot not loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool { return f.ID == mine }) {
		return snap, fmt.Errorf("personal: unknown franchise")
	}
	return snap, nil
}
