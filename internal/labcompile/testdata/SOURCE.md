# Independent numerical and Lab-method recordings

Reference commit: `b7213f63e52ecfa3950bc4bc650f47ea3aeabac9`, from the supplied `lab-reference/SOURCE.txt`. The reference is unmodified. Its full database was independently rebuilt by `tools/labparity/facts-reference.py` in the preceding phase, before any Go Lab code existed. The directory name `four-season-reference` describes its exported selection, not its database: the database contains the full archive.

Commands, from the ring-3-d worktree, entirely offline:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 \
 /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s1c/oracle_chain.py
PYTHONDONTWRITEBYTECODE=1 PYTHONHASHSEED=0 python3 \
 /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s1c/record_kernels.py
cp /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s1c/kernels.json.gz \
 internal/labcompile/testdata/kernels.json.gz
```

Both retained scripts use the read-only reference at `/home/chris/fleet/runs/warroom-ring3-2026-10-11/lab-reference/tools/league-history`. They open the full database read-only at `/home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s1b/four-season-reference/reference/tools/league-history/data/mfl.db`. They fix `built` and `archiveThrough` to the frozen document, and point page-exam input at an absent phase-local path. The clock is the only replacement affecting inputs. Passive wrappers invoke each original function once with unchanged arguments, return its original result unchanged, and record arguments/results. A final exact JSON-value assertion proves the recorded build equals the unrecorded P1 build.

Sampling: first half of each bounded reservoir plus `random.Random(1011)` reservoir sampling for the remainder; first forecast solve for each of the five horizons; all nine price fits (full, odd/even, early/late, spring/summer/season, cost). Shared price-input indices are lossless deduplication of the original sides. Only fields read by fit_prices are projected: value, class, cost. The forecast-check sample projects the QB group's five horizons and DB's fifth horizon; its pairs contain the reference's observed/calibrated-predicted values and the corresponding original check result. Forecast feature vectors and local-linear inputs are recorded directly. Nine additional explicit decimal tie cases use Python round, not Go. Price intervals are captured from the reference market result and the original full fit's SE.

Sample counts: solve 8; forecast fit 5; inverse 4; folds 24; discounted 32; local-linear 8; rounding 73; forecast features 24; forecast-check recording 1 (6 checks); price fits 9 (1,893 shared trade inputs); delivered intervals 29. BacktestRecords holds 32 sampled original output records for inspection only: it lacks valuation inputs and has no Go comparison. It is **not** a per-trade valuation golden. These kernels do not supply the full Forecast implementation or per-trade valuer. The document compiler now has the typed analysis seam below; value orchestration remains pending s2b. This numerical fixture alone does not prove value orchestration.

SHA256s:

| Input/output | SHA256 |
|---|---|
| full mfl.db | `c1aff6088b62d5d34fd88be9f094d117517f7c8b539b605f42a95b4b93d32dad` |
| build_lab.py | `cb9a42b1f231ac71b5e78c1755b45019295ebd02a735926b66690af6a89c1d29` |
| value.py | `76258da4468f0431be40f3dcac4aea4eb6eae6c8f1a2909d9fefa992de61ca26` |
| ledger.py | `f9a9082a3833af4a2e4dc39e0f186f498ffc1770145c833d73b5ed927588d1df` |
| frozen lab.json.gz | `ea25f32dadb42f9411ff63bc18dce596d9214a1371ab4a7e129ab2c8b9714890` |
| unrecorded reference JSON | `f2eaa91fb397adbd6662653971da721835e64ebdf5a7b9d93c333c4999b2af98` |
| kernels.json.gz (906,860 B) | `57fc268033f8e57725762b9b7ef5e6ea12a98ca650ffd7973479bb1a443561b0` |

P1 compared 508,219 scalar values, with exactly the expected absent-page-exam differences (asked 0 vs 2,295; matched 0 vs 2,291), and no other differences. These counts prove the frozen document's provenance, not Go end-to-end parity.

## Lab methods and document assembly (s2a)

The unchanged full reference ran again with passive recorders in `p3g2-s2a/record.py`.
The wrappers return each original result unchanged, including `value.analyse`.
They record trades by their index in the original transaction-id-ordered list,
not Python identities. The final document is asserted JSON-value-identical to
s1c's unrecorded P1 output. Both timestamps come from the frozen document;
`factexam.json` is absent. No reference compiler function is replaced with a
fabricated implementation, and no Go output is an oracle.

Reproduce offline from this worktree:

```sh
. /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s2a/env.sh
python3 /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s2a/record.py
python3 /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s2a/method_facts.py
cp /home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s2a/method-facts.json.gz \
 internal/labcompile/testdata/lab-methods.json.gz
TWR_ARCHIVE_PARITY=1 \
 TWR_FACTS_ARCHIVE=/home/chris/work/TheWarRoom/league-archive/raw \
 TWR_LAB_ANALYSIS=/home/chris/fleet/runs/warroom-ring3-2026-10-11/p3g2-s2a/analysis.json.gz \
 go test -count=1 -v ./internal/labcompile -run TestFullDocumentParity
```

Methods use the first 16 calls and a fixed-seed (1011) reservoir of 16 later
calls, supplemented with the first call of each observed phase, era,
normalized position and (position, year) tier boundary. All five eras,
all nine phases and all ten positions are covered. Sample counts:

- `week_at`: 32.
- `phase`: 36.
- `situation`: 32.
- `year_before`: 32.
- `cap_room`: 32.
- `contract`: 32.
- `contract_after`: 32.
- `tier_from_rank`: 153.
- `known_tier`: 32.
- `live_rank`: 32.
- `after_use`: 32.
- `era_of`: 14.
- `norm_pos`: 34.

`offer_outcomes` and `ledger.reconcile` are recorded in full, not sampled:
470 proposals / 940 offer inputs, and the complete replay check. Verify rebuilds
Go state from independent database fact projections inside the gzip, never
serialized Python Lab state. All names, positions, kickoff/game facts,
started lineups, rosters and cap charges needed by these methods are retained.
Player metadata projects only ids and draft years (the sampled methods do not
read birthdays or names). Player-week rows retain every regular, finished week
for each position/year needed by sampled known tiers and live ranks, plus every
sampled after-use player's score. Omitted columns are zero-filled only for
fields none of these methods read. The full-archive document test has no such
projection. Draft spans, original trades and ledger-changing add/drop moves
are reference inputs; discarded move kinds are ignored by `ledger.reconcile`
itself. The facts fixtures in `internal/lab/testdata/facts` are unchanged.

Document parity compares every top-level key against the frozen Lab and
round-trips the result through `lab.DecodeGzip`. Analysis is typed, validated,
phase-local and never checked in; `Compile` remains an explicit not-ported
error until s2b supplies Go value analysis. Source proofs use a byte-identical
copy of the already pinned facts check exceptions; detail strings reproduce
the reference's counted descriptions. Undefined empty-ledger rates, missing
snapshots, missing positions, and missing starter lines stay null, not zero.

Additional SHA256s:

| Artifact | SHA256 |
|---|---|
| record.py | `09df61a97989f0348d65f125e0226a500dc6ecf963d7e08f97974035f49f65ea` |
| method_facts.py | `c9fd853eba42afa1c172ff77465f5253c2373b412beba1dbce32c5390d7f07a1` |
| lab-methods.json.gz (880,459 B) | `7f2f1215ded45e55d1080c5299b09f335c13d8a4cc2a40a68ec9da35839a68e8` |
| analysis.json.gz (phase only) | `c54bd88abc64238c9a5239bd06802d5ff45ff0c6c919bf63b045edcbd20c4366` |
| check-exceptions.csv | `6c2792a3553321b204d0de85c48ebaf019a6dfb9cbd2e03e2bbb01ebffb61790` |
