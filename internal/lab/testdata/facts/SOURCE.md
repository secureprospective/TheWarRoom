# Independent facts reference (not Go compiler output)

Reference: `session/league-history-pwa` commit
`b7213f63e52ecfa3950bc4bc650f47ea3aeabac9`, read-only export supplied by Claude.
Rebuilt on 2026-10-11 with Python 3.13's standard library. The original checkout
was not modified; no git or network commands were used.

`tools/labparity/facts-reference.py` copies the reference's `store/build.py`,
`load.py`, `schema.sql`, `views.sql`, `checks.py`, `notes.py`, `cards.py` and
`note_readings.csv` into a disposable output directory, stages only the archive
bodies matching `internal/lab/testdata/inputs.csv`, and runs the copied
`store/build.py`. It verifies rebuilt `source_file` rows against the input pins.
It excludes card/search tables from extraction: they are not league facts.

Reproduce from the repo root, with a NEW output directory:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tools/labparity/facts-reference.py \
  --reference "$REFERENCE_EXPORT" --archive "$ARCHIVE_RAW" \
  --pins internal/lab/testdata/inputs.csv \
  --gaps internal/lab/testdata/missing-inputs.csv --output "$OUTPUT"
```

The reference build exited 0. This pins 1,482 archive inputs and stages 1,477.
Five approved missing originals are recorded in `gap-proof.json`; no bytes are
reconstructed or substituted. All five pins have size 175 B, the size of the
known login-error envelope. The 2014 assets and calendar pins share a SHA.
Their non-data classification is the brief's explicit engineering decision;
size alone does not prove the contents of unrecovered bytes. Reference
`load.py` records a source then drops `error` envelopes before business loading.
The rebuilt data has no 2013–2015 asset or calendar rows; calendar begins in 2016.

## File format

`facts.jsonl.gz`: deterministic gzip (`mtime=0`, empty filename), 39 relations,
1,161,195 rows. Each relation begins with a JSON object naming `table`, `kind`
and ordered `columns`; subsequent JSON arrays contain all its rows. Rows are
ordered by every column using SQLite's native ordering. Nulls are JSON null,
integers stay integers, ids stay strings, and REAL values retain Python's
round-trip decimal encoding. Raw JSON columns retain reference strings.
`counts.json` reports rows per relation. Source provenance is retained.

Gzip SHA256:
`491d0e807ef040c9e6fa9fa0caecdede72a9a2289176d0a3bc114eea56bbf876`.

`checks.json` is the independently run reference's nine check results.
`exceptions.json` pins every review item's full identity/description, the twelve
2013 playoff-standings identities, and the 505 three-point-bonus game identities
with scores. It is preparation for identity-based Go assertions, not a port of
those assertions. Reference count allowances must NOT be copied as Go policy.

## Human-curated input

`../note_readings.csv`: 469 readings, 37,943 B, byte-identical to the reference's
human-curated file. It is not an MFL export and remains unconfirmed where the
reference marks it unconfirmed. SHA256:
`b69a8d56549eb9680a803fc0b4f225bd3bbb2326cf8128a081294adef9517f7e`.
The reference staging manifest `selection.json` includes this hash separately
from the MFL pins. Runtime manifest integration is NOT implemented yet.

## What is and is not proved

`TestFactsReferencePinsMatchSourceRows` checks the golden fingerprint, relation
counts and every source_file row against the Go input pins plus the five exact
approved gaps. It does not compare Go-compiled fact columns: there is no Go
facts compiler in this change. Neither full nor reduced-season facts parity is
claimed. This fixture is never an input to a runtime facts compiler.
