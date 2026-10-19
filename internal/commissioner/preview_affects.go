package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// PreviewAffectedFranchises adds the current franchise count without changing catalogue-reading counts.
func PreviewAffectedFranchises(raw RawCommand, records []Record, now time.Time, affects PreviewAffects) (PreviewAffects, error) {
	if len(records) == 0 {
		return affects, nil
	}
	targets := previewTargets(raw)
	directory, err := NativeDirectoryReading(raw.League, records, now)
	if err != nil {
		return PreviewAffects{}, err
	}
	if len(directory.Franchises) == 0 {
		return affects, nil
	}
	names := map[string]string{}
	for _, f := range directory.Franchises {
		names[f.Franchise] = f.Name
	}
	targets, err = nativePreviewTargets(raw, records, now, targets)
	if err != nil {
		return PreviewAffects{}, err
	}
	slices.Sort(targets)
	targets = slices.Compact(targets)
	labels := []string{}
	for _, target := range targets {
		if names[target] != "" {
			labels = append(labels, names[target])
		}
	}
	slices.Sort(labels)
	unit := "franchises"
	if len(labels) == 1 {
		unit = "franchise"
	}
	words := fmt.Sprintf("%d %s affected: %s", len(labels), unit, strings.Join(labels, ", "))
	if len(labels) == 0 {
		words = fmt.Sprintf("All %d franchises; the league readings and ledger change", len(names))
	}
	affects.Words = words
	return affects, nil
}

func nativePreviewTargets(raw RawCommand, records []Record, now time.Time, targets []string) ([]string, error) {
	if raw.Args.Override != nil && raw.Args.Override.Player != "" {
		franchise, err := playerFranchise(records, raw.Args.Override.Player)
		if err != nil {
			return nil, err
		}
		targets = append(targets, franchise)
	}
	if raw.Args.Trade != nil {
		targets = append(targets, raw.Args.Trade.Parties...)
		_, book, readErr := nativeAssetsState(raw.League, records, now)
		if readErr != nil {
			return nil, readErr
		}
		for _, t := range book.Trades {
			if t.ID == raw.Args.Trade.ID {
				targets = append(targets, t.Parties...)
			}
		}
	}

	return targets, nil
}

func previewTargets(raw RawCommand) []string {
	a := raw.Args
	switch {
	case a.People != nil:
		return []string{a.People.Franchise}
	case a.Override != nil:
		return []string{a.Override.Franchise}
	case a.Exception != nil:
		return []string{a.Exception.Franchise}
	case a.Lineup != nil:
		return []string{a.Lineup.Franchise}
	case a.Score != nil:
		return []string{a.Score.Franchise}
	case a.Standings != nil:
		return []string{a.Standings.Franchise}
	case a.Correction != nil:
		targets := []string{a.Correction.Franchise, a.Correction.Owner}
		if a.Correction.Scope != nil {
			targets = append(targets, a.Correction.Scope.Franchise)
		}
		return targets
	}
	return nil
}

func playerFranchise(records []Record, player string) (string, error) {
	assets, err := NativeAssetsReading(records[0].Command.League, records)
	if err != nil {
		return "", err
	}
	if !assets.Seeded {
		return "", nil
	}
	for _, p := range assets.Snapshot.Players {
		if p.ID == player {
			return p.Franchise, nil
		}
	}
	return "", nil
}

// NamePreviewFields uses the directory's current franchise names, including commissioner name changes.
func NamePreviewFields(fields []PreviewField, records []Record, now time.Time) error {
	if len(records) == 0 {
		return nil
	}
	directory, err := NativeDirectoryReading(records[0].Command.League, records, now)
	if err != nil {
		return err
	}
	names := []string{}
	for _, f := range directory.Franchises {
		names = append(names, f.Franchise, f.Name)
	}
	replacer := strings.NewReplacer(names...)
	for i := range fields {
		fields[i].Label = replacer.Replace(fields[i].Label)
		fields[i].Before = replacer.Replace(fields[i].Before)
		fields[i].After = replacer.Replace(fields[i].After)
	}

	return nil
}
