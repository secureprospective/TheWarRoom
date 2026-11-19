# TheWarRoom phone app — a plan for every stub in mobile mock v1

**Written:** 2026-11-19 by Tom (Claude on the Beelink), from five Luna inventory lanes. **Checked against:** `bundle/b9` `527d6ba9b`.
**Pairs with:** `MOBILE-V1-STUB-INVENTORY.md` (545 rows, one per control/number/state/sheet/gesture) and the desktop plan `docs/ui/v3/V3-STUB-PLAN.md`.

## What this is for
Mobile mock v1 (`docs/ui/showcase/v1/TheWarRoom-mobile-mock-v1.zip`; live field `twr-live-mobile-a3.html`) is **the target for the finished phone product, not a suggestion** (Christopher, 11-19). Desktop was built first because it carries most of the wiring; the phone reuses that wiring. Something new for the phone is almost always new because it is a phone, not because the data is missing.

Bundles 8–9 got this wrong: they squeezed desktop screens into a phone frame (`shell/PhoneShell.tsx` maps Live→Pulse, Team→HQ, Moves→War Room) and added ~1,000–1,200 lines of phone CSS and wrappers. That layer is retired tab by tab as the port lands (column "retires" in the inventory).

## The count
| State | Rows | Meaning |
|---|---|---|
| WIRED | 198 | Server read/command and client reader exist on b9. Build only the phone view. |
| PARTIAL | 230 | Owner exists. ~150 only warn "use the real read, not the mock's sample value"; **79 name a real missing piece** (gap list below). |
| NEW-MOBILE | 37 | Phone-only: sheets, tab bar, gestures, install, first-run tour, live field animation. |
| NEW | 14 | Missing everywhere, search recorded: PACE, NFL team records, timeouts, player-status feed, request store, team-count expansion, tour. |
| RULING | 36 | About 20 are already settled by the v3 Decisions (below); the rest fall under 7 open questions for Christopher. |
| MOCK | 30 | Prototype controls (Mock controls sheet, play buttons, sample leagues). Never built. |

## Rules for every port lane (binding; v3 rules 1–12 also apply)
1. **The mock is a picture.** Nothing from the mock's JS/CSS/HTML is pasted into `frontend/`. Match the look exactly; write the code in the real owner.
2. **Look the row up first.** Every port PR cites the inventory ids it implements. Use the reader and command the row names. A row's "do not" clause is binding.
3. **One phone app, one switch.** Phones (< 700 px, existing `PHONE_QUERY`) render the phone app; desktop renders unchanged. No per-screen `@media` patches on desktop components.
4. **Retire as you go.** A tab's port PR deletes the Bundle 8/9 phone code its rows list under "retires" and its now-dead tests. Net line count is reported in the PR.
5. **Same data, same words.** Readers come from `data/hostedReads.ts` + `query.ts`; commands go preview → confirm → execute → reread through the existing workflow path. No engine math in React (PACE, win %, odds, values are server reads).
6. **No sample data ships.** If no read produces it, show the read's own unavailable words, never the mock's numbers or names.
7. **Proof is side by side.** `node ~/fleet/bin/twr-phone-compare.mjs <tab> --width 390` and `--width 430` against the lab: no sideways scroll, every mock label present or explained by a row, tap targets ≥ 44 px (32 px player badges per ruling). Screenshot pair in the PR.
8. **New code needs its row.** A port lane that finds something with no row adds the row (with search evidence) before building it.

## Decisions already made (do not reopen)
| Topic | Ruling | Rows it settles |
|---|---|---|
| Server operations in Control Room | v3 Decision 3: commissioners see read-only server health; restore, feed pause/switch, upgrade are operator-only. Phone shows health text only. | ~14 `cr:server-*`, `act:backup-*`, `act:feed-*`, `cr:job-paused-feed`, `cr:restore-*` |
| Phone layout preference | Viewport width decides; no user "layout on phones" setting (b8plan, 11-18). | `cr:league-phone-layout` |
| Data on screen | Status facts only; no written news, no ROST %, no trending adds; win % and PACE are ours (server). | all player-status rows |
| Placed bids | Locked; no lower/withdraw (v3 Decision 1). | moves bid rows |
| Per-person Home layout | Server per-person layout (D58, built in b9 #388/#405). | `today:board-*` |
| New owner commands | v3 Decision 5: approve each one when its screen is next. | claim reorder/cancel, trade hold |
| Test accounts | Left unseated for alpha (11-19). | — |

## Open questions for Christopher
| # | Question | Rows |
|---|---|---|
| Q1 | **League switcher on phone.** The mock has one; the browser session admits one league (409 for multi-league accounts). Allow a member-safe list of my leagues + switch? | `live:league-*`, `shell:league-switch`, cross-league Today (`today:league-count`, `today:job-commish-seven`) |
| Q2 | **Manual Done / snooze on jobs.** Mock lets you tap Done or snooze a job. Real jobs settle only by their command. Allow a personal snooze (private, D57) and drop Done? | `today:job-flex-done`, `shell:toast-done`, `cr:job-later/done/wake/snoozed-list` |
| Q3 | **Owner-to-owner private messages** (nudge a quiet owner, seat message). | `act:quiet-nudge`, `cr:seat-message` |
| Q4 | **Trade hold 24 h** (review-window extend). | `act:trade-hold-24h`, `cr:trade-hold-reason` |
| Q5 | **Claim reorder / cancel / waiver bid sheet** (Decision 5, Moves is next). | `moves:claim-reorder`, `moves:claim-cancel`, `moves:waiver-bid-sheet` |
| Q6 | **League colour** setting. | `cr:league-color` |
| Q7 | **Freeze trades on a collusion complaint.** | `act:complaint-freeze-trades` |

## Real gaps (server or client work the port needs)
| Gap | Kind | Rows | Plan |
|---|---|---|---|
| G1 Live field (a3 Grip field, play animation, big play blast doors, drive bar, score reveal) | NEW-MOBILE client | `live:field-*`, `live:big-play`, `live:score-reveal`, `live:drive-bar` | Port the a3 picture into a React field component over `nfl.game.read`; biggest client job; own lane. |
| G2 Player-status feed (out/active/practice/depth, D14) | NEW server | `team:injury-facts`, `cr:activity-player-fact-*`, `league:activity-player-filter` | One hosted status read through the storage seam; renderer shared by Team, Live, League. |
| G3 Server small reads: PACE, NFL team records, timeouts, standings streak, open-bench count | NEW/PARTIAL server | `live:my-matchup-pace`, `live:nfl-record`, `live:nfl-timeouts`, `league:table-streak`, `moves:bench-capacity` | Add fields to existing reads (`results.read`, `nfl.games.read`, `standings.read`), not new endpoints. |
| G4 Watch from player card | PARTIAL client | `live:watch-player-*` | Reuse `player.watch.*`; add the action to the card. |
| G5 Six-game range | PARTIAL | `team:six-game-range`, `team:card-history` | Extend `recentPlayerForm` from 5 to 6 starts. |
| G6 Today compact tiles (activity, standings, calendar) | PARTIAL client | `today:piece-*` | Compact renderers over existing reads. |
| G7 Claim submission in Market | PARTIAL | `moves:claim-submit`, `moves:claim-row-data` | Reuse waiver workflow; Q5 covers reorder/cancel. |
| G8 Ask answers for NFL/game questions | PARTIAL | `live:ask-*` | Extend existing Ask routing; full assistant is D74 (later). |
| G9 Control Room producers: quiet owner / lineup hole, complaints, deputy permissions (D65), house-rule log (D71), CR assistant (D74), presets values | PARTIAL/NEW | `cr:situation-*`, `cr:job-quiet-*`, `cr:permission-*`, `cr:house-rules`, `cr:ask-*`, `cr:preset-*` | Later list, same as desktop v3; phone shows the read's unavailable words until then. |
| G10 First-run tour | NEW-MOBILE client | `shell:first-run-tour`, `shell:tour-*` | Small client-only tour; last. |

## Build order
| Wave | Lanes (parallel inside a wave) | Depends on |
|---|---|---|
| 0 | **Phone app frame:** phone app root under < 700 px; tab bar, app bar, sheets, toasts, badges per mock; retire `PhoneShell*`, `PhoneAsk*` wrappers. | — |
| 1 | **Today** (WIRED-heavy; G6) · **Team** (18 WIRED, 0 NEW; G4/G5) | 0 |
| 2 | **Moves** (G7, Q5) · **League** tab (G3 streak) | 0 |
| 3 | **Live** scoreboard + matchup + box score · **Live field** (G1) | 0 |
| 4 | **Control Room on phone** (126 WIRED; settled server-ops rows show health only) | 0 |
| ∥ | Server lanes alongside waves 1–3: G2 status feed, G3 small reads | — |
| last | Tour (G10); phone walk on S8 + iPhone as owner and commissioner | 1–4 |
