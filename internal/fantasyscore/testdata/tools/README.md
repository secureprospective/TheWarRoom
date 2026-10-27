# Fixture extractors

These scripts rebuild the native-scoring fixtures from held study data. They make no requests and
read only files on the Beelink; run each with `python3 -I` from this directory.

| Script | Writes | Reads |
|---|---|---|
| `default32_golden.py` | stdout → `../default32_2024.json` | `~/fleet/runs/config-study-p5-2026-10-26/` (`study4c.py`, its `legacy_scorer.py` and cache), `tools/league-history/data/mfl.db` (read-only) |
| `oline_line_fixtures.py` | `../../source/testdata/line_plays.csv`, `line_rushing.csv` | `~/fleet/runs/sol61-sol61-warroom-config-study-p3-2026-10-26/cache/` (2024 play-by-play, PFR advanced rushing, DynastyProcess ids), `mfl.db`, and `../oline_setB_2023_2025.csv` |

`../oline_setB_2023_2025.csv` is the study's `oline_scores_setB.csv` (config-study p3), copied whole.

`default32_golden.py` drops any row whose per-play chunk sum differs from the study's `CK2` or whose
40+ completions differ from weekly `passing_40` (it prints each to stderr); none did on 2026-10-26.
The study paths are absolute because the study runs live outside the repo; change them if they move
(`~/MOVED.md` records any move).
