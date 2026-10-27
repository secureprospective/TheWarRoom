# Default32 scoring: edge stats

The preset is `internal/leaguerules/scoring_default32.go`. These points are per stat unit,
before the engine rounds each stat line to cents. Commissioners can change the rules. The rest
of the preset is described in [native scoring](native-scoring.md).

| Stat | Points | Positions | Source |
|---|---:|---|---|
| third_down_conversion | +0.25 | QB, RB, WR, TE | Play-by-play |
| fourth_down_conversion | +0.40 | QB, RB, WR, TE | Play-by-play |
| third_down_passing_conversion | +0.10 | QB | Play-by-play |
| fourth_down_passing_conversion | +0.15 | QB | Play-by-play |
| red_zone_carry | +0.08 | QB, RB, WR, TE | Play-by-play |
| red_zone_target | +0.08 | QB, RB, WR, TE | Play-by-play |
| red_zone_pass_attempt | +0.04 | QB | Play-by-play |
| rushing_yards_after_contact | +0.003 | QB, RB, WR, TE | Sleeper |
| receiving_yards_after_catch | +0.003 | QB, RB, WR, TE | Sleeper |
| receiving_air_yards | +0.002 | QB, RB, WR, TE | Sleeper |
| passing_air_yards | +0.001 | QB | Sleeper |
| rushing_broken_tackle | +0.08 | QB, RB, WR, TE | Sleeper |
| offensive_snap | +0.002 | QB, RB, WR, TE | Sleeper |
| receiving_drop | −0.30 | QB, RB, WR, TE | Sleeper |
| penalty | −0.25 | QB, RB, WR, TE, PK, DT, DE, LB, CB, S | Sleeper |
| penalty_yards | −0.01 | QB, RB, WR, TE, PK, DT, DE, LB, CB, S | Sleeper |
| defensive_sack_yards | +0.05 | DT, DE, LB, CB, S | Sleeper |
| special_teams_solo_tackle | +0.25 | DT, DE, LB, CB, S, PK | Sleeper |
| defensive_snap | +0.008 | DT, DE, LB, CB, S | Sleeper |
| special_teams_snap | +0.003 | DT, DE, LB, CB, S, PK | Sleeper |
| rushing_tfl | −0.3124 | QB, RB, WR, TE | Play-by-play |

The special-teams tackle is a top-up, not a replacement for the ordinary tackle rule.
Sleeper supplies only the listed owned stats; it does not overwrite nflverse or play-derived
counts. The remaining ordinary weekly stats come from nflverse. Play-by-play is the final source
for conversions, red-zone counts and rushing TFL, not Sleeper's aggregates.

## RTL

The weekly target is `−0.3124 × (TFL − r_pos × carries)`, with `r_pos` the prior full
regular-season league TFL rate at the position. The shipped preset cites 2025 play-by-play and
weekly positions for 2026, counting FB as RB. It stores −0.3124 per TFL and these rounded
position extras per carry: QB +0.0075, RB +0.0285, WR +0.0384, TE +0.0272. Those stored
extras, not an unrounded calculation, determine the actual result; each stat line is rounded
separately. They add to the ordinary +0.15 per carry. Rates do not update automatically.

## Play definitions

Only live plays count: deleted plays, two-point tries and `play_type == no_play` are skipped.
A carry is a live `rush_attempt == 1`; kneels and aborted snaps charged as rushes count.
A rushing TFL is a carry with `tackled_for_loss == 1`, not a loss inferred from yards.
Conversions credit the rusher or receiver and, on passing conversions, the passer.

The red zone is `yardline_100 <= 20`, including the 20-yard line. Carries use the same carry
rule. Pass attempts require `pass_attempt == 1` and no sack; targets use the same gate and
credit the named receiver. Spikes and throwaways count as attempts. The boundary awaits the
owner's confirmation. Older held play bodies without a `sack` column cannot exclude sacks;
see [handoff](../../HANDOFF.md).
