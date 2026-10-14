package source

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// IdentityPort reads the app's existing player directory; it never infers a match from a name.
type IdentityPort interface {
	SourceID(context.Context, string, string) (string, error)
}

type RawNativeScope struct {
	Player string
	Season int
	Week   int
}

func (r RawNativeScope) Validate() error {
	id, err := playerid.New(r.Player)
	if err != nil {
		return fmt.Errorf("choose a league player: %w", err)
	}
	if id.String() != r.Player {
		return fmt.Errorf("use the player's full league id")
	}
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("choose an NFL season and week")
	}
	return nil
}

// NativeEvidence retains absence rather than manufacturing a stat line or reusing an older revision.
type NativeEvidence struct {
	GSIS    string
	Held    *HeldWeek
	Missing []string
}

func ReadNativeEvidence(ctx context.Context, identity IdentityPort, stats StatPort,
	raw RawNativeScope) (NativeEvidence, error) {
	if err := raw.Validate(); err != nil {
		return NativeEvidence{}, err
	}
	gsis, err := identity.SourceID(ctx, raw.Player, "gsis")
	if err != nil {
		return NativeEvidence{}, fmt.Errorf("match the league player to NFL statistics: %w", err)
	}
	out := NativeEvidence{GSIS: gsis, Missing: []string{}}
	if gsis == "" {
		out.Missing = append(out.Missing, "This player has no matching NFL statistics identity")
		return out, nil
	}
	scope := RawStatScope{GSIS: gsis, Season: raw.Season, Week: raw.Week}
	if err := scope.Validate(); err != nil {
		return NativeEvidence{}, fmt.Errorf("read matched NFL identity: %w", err)
	}
	revisions, err := stats.PlayerWeekStats(ctx, scope)
	if err != nil {
		return NativeEvidence{}, fmt.Errorf("read the player's held statistics: %w", err)
	}
	if len(revisions) == 0 {
		out.Missing = append(out.Missing, "No NFL statistics are held for this season and week")
		return out, nil
	}
	out.Held = &revisions[0]
	out.Missing = append(out.Missing, out.Held.Missing...)
	if out.Held.Week == nil && len(out.Missing) == 0 {
		out.Missing = append(out.Missing, "The newest held statistics do not report this player's week")
	}
	return out, nil
}
