package commissioner

import (
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func scoreSourceLabel(authority string) string {
	switch authority {
	case source.NFLVerse:
		return "final · nflverse"
	case source.ESPNLive:
		return "live · ESPN"
	case source.Sleeper:
		return "filled · Sleeper"
	case source.ESPNLive + " + " + source.Sleeper:
		return "live · ESPN + filled · Sleeper"
	default:
		return authority
	}
}

func (out *NativePlayerScore) setSources(counted *source.LineRevision, base StatBaseline,
	corrections map[leaguerules.StatID]StatCorrection, delta leaguerules.Decimal) {
	out.FieldSources = map[leaguerules.StatID]string{}
	if counted != nil {
		out.Source = scoreSourceLabel(counted.Held.Authority)
		for id, owner := range source.FieldOwners(*counted) {
			out.FieldSources[id] = scoreSourceLabel(owner)
		}
	}
	if base.Entry != "" {
		out.Source = "commish override"
		for id := range out.FieldSources {
			out.FieldSources[id] = out.Source
		}
	}
	if len(corrections) > 0 || delta != 0 {
		if out.Source == "" {
			out.Source = "commish override"
		} else if out.Source != "commish override" {
			out.Source += " + commish override"
		}
		for id := range corrections {
			out.FieldSources[id] = "commish override"
		}
	}
}
