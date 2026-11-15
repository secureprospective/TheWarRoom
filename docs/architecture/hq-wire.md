# Franchise HQ v10 wiring

Visual authority: `docs/ui/mocks/franchise-hq-mock-v10.html`, unchanged. The HQ route uses
`HQShell` and the scoped copy of its CSS; other workspaces retain their existing shell.
The native account's hosted franchise opens the same shell. Its Control Room rail destination
returns to the existing hosted workflows/account surface. Existing reads now carry the MFL display name, complete-pair model win probability and native annual
contract schedules. Wails models were regenerated; no new bound method was introduced.

## Data and state owners

| Route | Responsibility, state and failure boundary |
|---|---|
| Feature `api.ts` adapters → existing reads → parsers → HQ presentation | No new endpoint, external refresh, scoring or cap business rule. Unknown values stay absent. MFL signed adjustments are never relabelled dead money. |
| `useHQRead` / existing `useContractRead` | Complete argument keys, current-scope checks, late-response fencing, season/move subscriptions and cleanup. Failed or pending reads do not borrow another selection's facts. |
| `LineupRoster` → `LineupEditor` → `HQBoard` | Existing command-registry draft owns membership; existing Go check owns legality. HQ supplies presentation, settings-derived counts and selected-player interaction. Pending draft checks do not claim the saved verdict is current. |
| `HQAct` → `Act` → existing `commissioner.surface` | Scoped local view callbacks; handlers unregister on unmount. Actual guarded team actions retain their existing Acts. No callback is serialized into a command argument. |
| Inspector → notes read/change → notes reread | The backend owns count, text and dated-note identity. Removing one note sends its read timestamp. No invented local note count or successful-write fallback. Native hosted seats never borrow local MFL private lists. |
| What-if/release → existing local session and release-plan APIs | Backend-authored figures and prices. Holding saves an existing local release plan, never submits a roster move to MFL. A native cut uses `contract.quote`, read-only. |
| `readNativeHQ` → existing hosted workflow admission → native reads | No bearer credentials in frontend state. Scope/season checks and matching revisions where supplied; inconsistent regions become unavailable. Selected-player cut quotes are separately scoped and revision-fenced. |
| Moves → existing receipts plus bid/release/contract plan reads | Deduplicated identities, explicit partial-source failure, withdrawn plans excluded. Native `needs_you` stays native, using the existing native receipt parser. No assumption that all-history moves constitute a season ledger. |

MFL projections expose player names, not IDs: only unique exact directory names are joined, and
only for the matching week. Missing projections are never zero. Matchup totals from M-018 require
complete inputs; player deltas join report entities to real projections. Form charts use five held finished-week rows from the extended existing research read; missing
weeks stay missing and known byes are hatched, never fabricated.

## Settings consumed

MFL: salary cap from snapshot/contract reads; M-091 roster size, IR size, taxi size, weekly starter
count, offensive/defensive starter counts and position minimum/maximum limits. Contract durations
and annual schedules come from roster/contracts reads; future annual amounts retain the read's
inference reason. Clock deadlines are read, never reconstructed from a weekday literal.

Native: `roster.limits`, `roster.reserveExcluded`, `ir.slots`, `taxi.slots`, `lineup.positions` and
`lineup.groups` from the rules reading. Reserve-inclusive roster limits subtract actual reserves for
the displayed active allowance. Cap limit, reserve salary charges, current/future commitments, dead
money and room come from the cap check/runway; its horizon follows backend contract settings.
Cut costs and cap before/after come from the native contract quote. No frontend pricing or preset
branch. There is no literal league cap, squad capacity, starter count or contract horizon.

## Verification and limits

Mapping/empty-state tests cover the plate, starters/bench, projections, byes, form, matchup,
inspector, usage, arc, notes, reserve tables, runway, moves, native reads and cut quote. Mounted tests
exercise note add/remove, bench selection/placement/filtering, stale replies and private-list
isolation. Trade-draft tests observe no submission and unchanged state after a failed roster read.
Settings fixtures change capacities, lineup counts and cap values. The shared release hold is
exercised for early cancellation and callback-only confirmation.

The bundle gate now follows MyMoves' actual static dependency graph instead of assuming the shared
rail chunk is named EnvelopeRail. It still rejects eager rail/form/fixture leaks; its deliberate
violations pass the self-test. No budget or security requirement was weakened.

Current delivery status and recorded-browser limits: `docs/build-handoffs/HQ-FIX-report.md`.
`hqPlacement` owns presentation-only compatible targeting, swaps, bounded cascade and autofill from
held projections. Existing MFL commands own draft membership, checks, persisted plans and handoff.
Native HQ uses `createHostedRoom` for fresh revision/evidence admission, previews and explicit hold
confirmation; failure does not create an offline queue. Named slot persistence remains unsupported
by the membership-only command shapes. No exact pixel or authenticated Wails acceptance is claimed.

The mounted mirror HQ owns a separate `createLiveState` scope and existing visibility-aware poll.
Unmount removes timer/listeners; failed feeds do not paint a new live verdict. Team game state means
LIVE, not proof that an individual is on the field. The existing Now preview composes
`powerrankings.WinProbability` only for a complete projected pair. Native annual schedules are an
additive cap-read field over the existing contract pricer schedule, fenced to the assets revision.
The native What-if quote is a single-player read, never mislabeled as a saved stacked plan.
