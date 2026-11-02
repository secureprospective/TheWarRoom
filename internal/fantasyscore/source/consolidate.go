package source

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ConsolidateFinal appends a derived revision; original sequences and digests never change.
// Only the newest live line fills a final's absent fields, never an older live snapshot.
func ConsolidateFinal(versions []LineRevision) []LineRevision {
	var final, live *LineRevision
	for i := range versions {
		v := &versions[i]
		if v.Held.Week == nil {
			continue
		}
		switch v.Held.Authority {
		case NFLVerse:
			if !v.Merged {
				final = v
			}
		case ESPNLive, Sleeper, ESPNLive + " + " + Sleeper:
			live = v
		}
	}
	if final == nil || live == nil {
		return versions
	}
	line, owners := MergeStatSources(map[string]fantasyscore.StatLine{
		NFLVerse: finalAuthorityLine(*final), ESPNLive: live.Held.Week.Line, Sleeper: final.Held.Week.Line,
	})
	filled := false
	for id, owner := range owners {
		if owner == ESPNLive {
			filled = true
			owners[id] = fieldOwner(*live, id)
		} else {
			owners[id] = fieldOwner(*final, id)
		}
	}
	if !filled {
		return versions
	}
	week := *final.Held.Week
	week.Line, week.FieldSources = line, owners
	held := final.Held
	held.Week = &week
	if live.Held.FetchedAt.After(held.FetchedAt) {
		held.FetchedAt = live.Held.FetchedAt
	}
	// Keep the final's completeness claims; live-only fields still identify their provisional source.
	sum := sha256.Sum256([]byte("merged:" + final.Digest + ":" + live.Held.Authority + ":" + live.Digest + ":" + LiveRevisionDigest(week)))
	digest := hex.EncodeToString(sum[:])
	held.SHA256 = digest
	if versions[len(versions)-1].Merged && versions[len(versions)-1].Digest == digest {
		return versions
	}
	out := slices.Clone(versions)
	return append(out, LineRevision{Seq: len(out), Digest: digest, Held: held, Merged: true})
}

func fieldOwner(v LineRevision, id leaguerules.StatID) string {
	if owner := v.Held.Week.FieldSources[id]; owner != "" {
		return owner
	}
	return v.Held.Authority
}

// FieldOwners copies attribution so a commissioner overlay cannot mutate held evidence.
func FieldOwners(v LineRevision) map[leaguerules.StatID]string {
	owners := maps.Clone(v.Held.Week.FieldSources)
	if owners == nil {
		owners = map[leaguerules.StatID]string{}
	}
	for id := range v.Held.Week.Line.Quantities {
		owners[id] = fieldOwner(v, id)
	}
	for id := range v.Held.Week.Line.Events {
		owners[id] = fieldOwner(v, id)
	}
	return owners
}

// A held import may already contain Sleeper fills; those are not nflverse authority.
func finalAuthorityLine(final LineRevision) fantasyscore.StatLine {
	line := final.Held.Week.Line
	line.Quantities, line.Events = maps.Clone(line.Quantities), maps.Clone(line.Events)
	for id := range line.Quantities {
		if fieldOwner(final, id) != NFLVerse {
			delete(line.Quantities, id)
		}
	}
	for id := range line.Events {
		if fieldOwner(final, id) != NFLVerse {
			delete(line.Events, id)
		}
	}
	return line
}
