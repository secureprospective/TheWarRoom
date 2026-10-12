# Independent numerical recordings (partial compiler delivery)

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

Sample counts: solve 8; forecast fit 5; inverse 4; folds 24; discounted 32; local-linear 8; rounding 73; forecast features 24; forecast-check recording 1 (6 checks); price fits 9 (1,893 shared trade inputs); delivered intervals 29. BacktestRecords holds 32 sampled original output records for inspection only: it lacks valuation inputs and has no Go comparison. It is **not** a per-trade valuation golden. No full Forecast implementation, per-trade valuer or facts-to-Lab document compiler is supplied by these kernels. This fixture therefore does not complete phase P2, P3 or P4.

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
