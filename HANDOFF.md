# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. E5 is reconciled and the golden is in; do not push
without authorization.

## Ruled definitions for the play-derived stats (E5, 2026-10-27)
A play counts only if it is live: not `play_deleted`, not a two-point try, and not nullified by
penalty (`play_type == no_play`). Official stats exclude all three.

- **carry (`rushing_attempt`)** — any live play with `rush_attempt == 1`. QB kneels count as carries,
  as they do in official stats; so do aborted snaps charged to the quarterback. Sleeper's `rush_att`,
  nflverse's weekly `carries` and this rule agree on all 61 E5 player-weeks, so the definition is
  not in dispute.
- **`rushing_tfl`** — a carry with `tackled_for_loss == 1`. A kneel is never a tackle for loss, and
  nor is an aborted snap: nobody tackled the runner, and the loss is already priced by the fumble
  rules. Convention chosen: nflverse's explicit `tackled_for_loss` flag, not Sleeper's
  `rush_tkl_loss`, which charges three aborted snaps over these five player-seasons. Two reasons,
  the second decisive: a tackle for loss should require a tackle, and the ruled prior-season rates
  (QB 78/2352, RB 1106/11832, WR 62/430, TE 7/62) were themselves counted with this flag, so the
  numerator for a player must match the numerator that set the rate. RTL therefore shares the carry
  definition above, as the rate formula requires.
- **`red_zone_carry`** — a carry snapped at `yardline_100 <= 20`.
- **`red_zone_pass_attempt`** — a live play snapped at `yardline_100 <= 20` that is an official
  individual pass attempt: `pass_attempt == 1` **and not a sack**. nflverse's `pass_attempt` means
  "pass play" and includes sacks; official individual attempts never do. Spikes and throwaways are
  official attempts and count.
- **`red_zone_target`** — the same gate, credited to the named receiver. A sack names no receiver.

**Red-zone boundary, the one place a convention had to be picked:** the 20-yard line is inside the
red zone (`<= 20`). That follows the NFL's and Pro-Football-Reference's "inside 20" tables and the
nflfastR convention the held data comes from. Sleeper's aggregates use `< 20`; that difference is 18
of the 21 plays the two sides dispute and is worth +0.16 to +0.32 per player-season. Christopher can
veto it; `<` would mean regenerating the E5 golden and the two rate constants.

## Where it stands
E5 is complete. The engine was wrong on one thing — it counted red-zone sacks as pass attempts
(Allen 5, Hurts 1) — and it had no `no_play` filter. Both are fixed in `plays.go`/`plays_edges.go`. The study
(`scripts/e5_study.py`) was wrong on two: it took its red-zone and tackled-for-loss counts from
Sleeper's weekly aggregates rather than counting plays, and it inherited Sleeper's `< 20` boundary.
It now counts from held play-by-play itself and prints every disputed play with its class.

`TestE5Golden` pins all five player-seasons off `testdata/e5_*` (regenerate with
`scripts/e5_fixtures.py`), runs ungated in 0.02s, and holds the ruled 0.01-per-player-season
tolerance. Engine and study agree to the cent on all five: Hunt 163.4400, McCaffrey 419.6000,
Hurts 326.5300, Allen 356.9000, Barkley 230.2800. No existing evidence digest changed:
`testdata/plays.csv` carries neither `play_type` nor `sack`, so both new filters are no-ops there.
The ruled 2024 rates still derive unchanged from the full cache (extras 104/292/450/353).

## Next move
E6.

## Blocked on
Nothing. One owner question stands open but does not block: the `<= 20` red-zone boundary above.

## Tried and rejected
Do not treat Sleeper's `rush_rz_att`/`pass_rz_att`/`rec_rz_tgt`/`rush_tkl_loss` as an oracle: they
are a vendor aggregate on a different boundary and a different loss rule, and using them as the
study's evidence is what made E5 look broken. Do not infer fantasy-roster membership or bye status
from weekly statistics. Do not require GSIS for direct Sleeper→MFL, create fake identities, or
inner-join away Sleeper-only weeks. Unknown IDs are not automatically irrelevant.

Debt left: E6; unknown Sleeper identity/roster coverage; the red-zone boundary is Claude's ruling,
not Christopher's. One gap stays open deliberately: `ReadPlays` does not require the `sack` column
for edge capability, so a held body that predates it still counts red-zone sacks as pass attempts.
Requiring it was tried and reverted — it makes `ReadPlays` refuse the pinned evidence bodies in
`internal/store/history` and `plays_edges_internal_test.go`, which is the worse failure. Fixing it
properly means re-pinning those bodies with the column.
