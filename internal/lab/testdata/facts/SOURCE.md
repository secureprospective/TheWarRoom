# Independent Python facts reference

Reference: `session/league-history-pwa` commit
`b7213f63e52ecfa3950bc4bc650f47ea3aeabac9`, supplied read-only export.
The full reference was rebuilt offline on 2026-10-11 with Python 3.13;
`store/build.py` exited 0. Neither fixture contains Go-generated facts.

## Reduced golden

`facts.jsonl.gz` contains all 39 reference relations, 207,031 rows, and seasons
2013, 2020 and 2026. It is 1,487,539 B, SHA256
`5c2724cdb2ec0c4c19d413e8f5a399e826e031a5411d4c9d8e86b34ca28523e0`.
Each relation starts with its table, kind and ordered column names, followed by
JSON row arrays ordered by every column using SQLite's native ordering. Nulls,
string IDs, integers, Python round-trip REAL decimals and raw JSON are retained.
Deterministic gzip uses zero timestamp and no filename; installed `pigz -11`
provides Zopfli DEFLATE to keep every selected row below 1,500,000 B.

Selection happens AFTER the full independent reference build. Rows with `year`
or `snapshot_year` select the three seasons. Transaction-dependent relations
select through the original transaction IDs. Source/load notes select those
season prefixes plus root sources. Relations without a seasonal key retain all
rows: player identities reflect the latest full-archive identity, rule codes
remain global, and check/review annotations remain the full reference's checks.
This preserves original transaction IDs and human interpretations; a three-year
archive rebuild would renumber transactions and break historical pick chains.
2013's explained playoff standings and 2026's partial season remain represented.
`counts.json` records reduced counts; checks/exceptions/gap-proof describe the
full independent build, not newly computed reduced-archive checks.

Reproduce from the repo root into a NEW directory (Python 3 and pigz required):

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tools/labparity/facts-reference.py \
  --reference "$REFERENCE_EXPORT" --archive "$ARCHIVE_RAW" \
  --pins internal/lab/testdata/inputs.csv \
  --gaps internal/lab/testdata/missing-inputs.csv --output "$OUTPUT" \
  --seasons 2013,2020,2026
```

`--database "$INDEPENDENT_DB"` instead extracts an existing rebuilt reference;
its neighboring staging `selection.json` and reference store schema are required.
Omit `--seasons` to extract the full 1,161,195-row oracle, which is NOT checked in.
Its SHA256 is
`491d0e807ef040c9e6fa9fa0caecdede72a9a2289176d0a3bc114eea56bbf876`.

## Lossless pinned inputs for verify

`inputs.json.gz` is 3,643,488 B, SHA256
`2458008da347d21b6373374027e739f1e43f9ffd15061b2ba64cc89bea15ff93`.
It is independently produced by the same Python tool from pinned archive bytes,
NOT from either compiler's output. A shared dictionary of exact JSON lexical
spans and each input's token indices reconstruct byte-identical retained bodies;
Go checks reconstructed lengths and hashes before compiling.

All 1,477 available source identities are present. Bodies are retained for the
selected seasons' league/standings/rules/calendar/weekly-results/rosters, root
rule codes, and the full league/player/draft/transaction context needed by global
player identities and curated pick checks. Other bodies are deliberately absent
from this TEST projection; their metadata does not prove their unselected facts.
The full parity test replaces every body from the independently staged archive.
Together the golden and input fixture total 5,131,027 B, less than the deleted
8,701,947 B full golden alone. The runtime compiler reads neither fixture.

## Missing pins and curated input

The full build resolves 1,477 of 1,482 pins. The five exact approved missing
175-byte originals are listed in `../missing-inputs.csv` and `gap-proof.json`;
no substitute bytes are constructed. Their non-data classification is the
explicit engineering ruling, not an inference that size proves missing content.
The loader records available error sources but emits no business facts for them.

`../note_readings.csv` holds 469 human interpretations, not MFL records; SHA256
`b69a8d56549eb9680a803fc0b4f225bd3bbb2326cf8128a081294adef9517f7e`.
Confirmation remains 0, and draft checks are independently replayed in Go.
The twelve 2013 standings identities, bonus-game identities and all review-item
identities in `exceptions.json` remain reference evidence, not count allowances.

## Executable comparisons

`internal/lab/facts_reference_test.go` pins the golden and source identities.
`internal/facts/parity_test.go` compares every typed column of the twelve
implemented relations against the reduced oracle in normal verify. Imported
numbers are exact normalized rationals; IDs, timestamps, enums, nulls and raw
JSON strings compare exactly. The shared helper also supports derived floats
with 1e-9 absolute plus relative tolerance, with deliberate mismatch tests.

Full-source and full-column parity skip unless explicitly enabled:

```sh
TWR_ARCHIVE_PARITY=1 TWR_FACTS_ORACLE="$FULL_ORACLE" \
TWR_FACTS_ARCHIVE="$ARCHIVE_RAW" go test -race ./internal/facts ./internal/lab \
  -run 'TestFullReferenceParity|TestFullFactsReferencePinsMatchSourceRows' -v
```

The other reference relations remain independently present in the golden for
subsequent compiler stages; this stage asserts only the twelve named relations.
